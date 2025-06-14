package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

type APIUsageStats struct {
	CallCount     int
	ErrorCount    int
	TotalDuration time.Duration
	mu            sync.Mutex
}

func (s *APIUsageStats) AddCall(duration time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CallCount++
	s.TotalDuration += duration
	if err != nil {
		s.ErrorCount++
	}
}

func (s *APIUsageStats) Print(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	avg := time.Duration(0)
	if s.CallCount > 0 {
		avg = s.TotalDuration / time.Duration(s.CallCount)
	}
	log.Printf("%s API usage: calls=%d, errors=%d, total_time=%v, avg_time=%v", prefix, s.CallCount, s.ErrorCount, s.TotalDuration, avg)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file (this is okay if env vars are set elsewhere)")
	}

	dbPath := "news.db"
	conn, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil { // Ensure DB connection is closed at the end
			log.Printf("Error closing database connection: %v", err)
		}
	}()

	llmClient, err := llm.NewLLMClient(conn) // Assuming NewLLMClient doesn't also return a scoreManager
	if err != nil {
		log.Fatalf("Failed to initialize LLM Client: %v", err)
	}

	config, err := llm.LoadCompositeScoreConfig()
	if err != nil {
		log.Fatalf("Failed to load composite score config: %v", err)
	}
	llmModelsForAnalysis := config.Models // Renamed for clarity from just 'models'
	if len(llmModelsForAnalysis) == 0 {
		log.Fatalf("No models defined in config for LLM analysis")
	}

	// Instantiate ScoreManager and its dependencies
	cache := llm.NewCache()
	calculator := &llm.DefaultScoreCalculator{}
	// Using a longer cleanup interval as this is a batch job
	progressMgr := llm.NewProgressManager(10 * time.Minute)
	scoreManager := llm.NewScoreManager(conn, cache, calculator, progressMgr)

	const batchSize = 10 // Reduced batch size for potentially more logging/granularity during testing
	const workerCount = 4

	var totalArticlesProcessed, totalLLMScoresGenerated, totalCompositeScoresUpdated int
	apiStats := &APIUsageStats{}

	offset := 0
	for {
		articlesToProcess, fetchErr := db.FetchArticles(conn, "", "", batchSize, offset)
		if fetchErr != nil {
			log.Fatalf("Failed to fetch articles: %v", fetchErr)
		}
		if len(articlesToProcess) == 0 {
			log.Println("No more articles to process.")
			break
		}

		currentBatchArticleIDs := make([]int64, 0, len(articlesToProcess))
		for _, article := range articlesToProcess {
			currentBatchArticleIDs = append(currentBatchArticleIDs, article.ID)
		}
		log.Printf("Processing batch of %d articles. IDs: %v", len(articlesToProcess), currentBatchArticleIDs)

		// Stage 1: Collect individual LLM scores for all articles in the batch
		log.Printf("Stage 1: Collecting individual LLM scores for %d articles...", len(articlesToProcess))
		var wg sync.WaitGroup
		articleCh := make(chan db.Article, len(articlesToProcess)) // Buffered channel

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				log.Printf("[Worker %d] Started", workerID)
				for article := range articleCh {
					log.Printf("[Worker %d] Analyzing article ID %d (%s) for individual LLM scores...", workerID, article.ID, article.Title)
					var scoresGeneratedForThisArticle int

					// Placeholder IDs: const idForInvalidTest = 7; const idForZeroConfTest = 8;
					// Actual IDs will be confirmed after running run_integration_setup_s3 again.
					// We will need to be vigilant and potentially re-apply this edit with correct IDs if they change.

					for _, modelCfg := range llmModelsForAnalysis {
						start := time.Now()						// Inner func for error handling and stats
						analysisErr := func() error {
							llmScoreResponse, errAn := llmClient.AnalyzeContent(article.ID, article.Content, modelCfg.ModelName, modelCfg.URL, scoreManager)
							apiStats.AddCall(time.Since(start), errAn)
							if errAn != nil {
								log.Printf("[Worker %d] Error analyzing article %d with model %s: %v", workerID, article.ID, modelCfg.ModelName, errAn)
								return errAn
							}
							_, errIns := db.InsertLLMScore(conn, llmScoreResponse)
							if errIns != nil {
								log.Printf("[Worker %d] Error inserting LLM score for article %d model %s: %v", workerID, article.ID, modelCfg.ModelName, errIns)
								return errIns
							}
							return nil
						}()
						if analysisErr == nil {
							scoresGeneratedForThisArticle++
						}
					} // End models loop

					log.Printf("[Worker %d] Finished LLM analysis for article ID %d. "+
						"Generated %d individual scores (skipped if test ID).", workerID, article.ID, scoresGeneratedForThisArticle)
					// Safely update global counter (though could be done after wg.Wait for simplicity if only counting total LLM scores)
					apiStats.mu.Lock() // Assuming APIUsageStats has a mutex for its counters if updated concurrently here
					totalLLMScoresGenerated += scoresGeneratedForThisArticle
					apiStats.mu.Unlock()
				} // End article channel loop
				log.Printf("[Worker %d] Finished", workerID)
			}(i)
		}

		for _, article := range articlesToProcess {
			articleCh <- article
		}
		close(articleCh)
		wg.Wait()
		log.Printf("Stage 1: Collection of individual LLM scores for batch complete.")

		// Stage 2: Calculate and store composite scores for the processed batch
		log.Printf("Stage 2: Calculating and storing composite scores for %d articles...", len(articlesToProcess))
		for _, article := range articlesToProcess {
			log.Printf("Fetching LLM scores for article ID %d to compute composite score...", article.ID)
			fetchedLLMScores, fetchLLMErr := db.FetchLLMScores(conn, article.ID)
			if fetchLLMErr != nil {
				log.Printf("[ERROR] Failed to fetch LLM scores for article ID %d: %v. Skipping composite score calculation.", article.ID, fetchLLMErr)
				continue
			}
			if len(fetchedLLMScores) == 0 {
				log.Printf("[WARN] No LLM scores found for article ID %d. Skipping composite score calculation.", article.ID)
				// This might happen if all individual LLM analyses failed to produce a storable score in Stage 1
				// Or if the article had no models configured (though config load checks this globally)
				// We still need to update its status to reflect failure if it was pending.
				// Calling UpdateArticleScore with empty scores should trigger ErrAllPerspectivesInvalid if appropriate.
			}

			log.Printf("Calculating composite score for article ID %d (%s) "+
				"using %d fetched LLM scores...", article.ID, article.Title, len(fetchedLLMScores))
			_, _, compErr := scoreManager.UpdateArticleScore(article.ID, fetchedLLMScores, config)
			if compErr != nil {
				// ScoreManager.UpdateArticleScore already logs details and updates status to an error state
				log.Printf("[ERROR] Failed to compute or store composite score for article ID %d: %v", article.ID, compErr)
				// The status is updated by ScoreManager, so no explicit status update here on error is needed
			} else {
				log.Printf("[INFO] Successfully computed and stored composite score for article ID %d.", article.ID)
				totalCompositeScoresUpdated++
			}
		} // End loop for composite scoring for the batch
		log.Printf("Stage 2: Composite score processing for batch complete.")

		totalArticlesProcessed += len(articlesToProcess)
		offset += len(articlesToProcess)
		log.Printf("Batch processed. Total articles processed so far: %d. Moving to next offset: %d", totalArticlesProcessed, offset)

		// Optional: Add a small delay between batches if needed for API rate limits or DB load
		// time.Sleep(1 * time.Second)
	} // End main processing loop (batches)

	fmt.Printf("\n--- Scoring Job Complete ---\n")
	fmt.Printf("Total articles processed (fetched in batches): %d\n", totalArticlesProcessed)
	fmt.Printf("Total individual LLM scores generated: %d\n", totalLLMScoresGenerated)
	fmt.Printf("Total composite scores successfully updated: %d\n", totalCompositeScoresUpdated)
	apiStats.Print("LLM Analysis API (AnalyzeContent)")
}

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

	// Handle both return values from NewLLMClient
	llmClient, err := llm.NewLLMClient(conn)
	if err != nil {
		log.Fatalf("Failed to initialize LLM Client: %v", err)
	}

	// Load config-driven models
	config, err := llm.LoadCompositeScoreConfig()
	if err != nil {
		log.Fatalf("Failed to load composite score config: %v", err)
	}
	models := config.Models
	if len(models) == 0 {
		log.Fatalf("No models defined in config")
	}

	const batchSize = 20
	const workerCount = 4

	var totalArticles, totalScores int
	apiStats := &APIUsageStats{}

	offset := 0
	for {
		articles, err := db.FetchArticles(conn, "", "", batchSize, offset)
		if err != nil {
			log.Fatalf("Failed to fetch articles: %v", err)
		}
		if len(articles) == 0 {
			break
		}

		articleIDs := make([]int64, 0, len(articles))
		for _, article := range articles {
			articleIDs = append(articleIDs, article.ID)
		}
		log.Printf("Scoring articles with IDs: %v", articleIDs)

		// Parallel processing with worker pool
		var wg sync.WaitGroup
		articleCh := make(chan db.Article)

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for article := range articleCh {
					for _, m := range models {
						start := time.Now()
						err := func() error {
							score, err := llmClient.AnalyzeContent(article.ID, article.Content, m.ModelName, m.URL)
							apiStats.AddCall(time.Since(start), err)
							if err != nil {
								log.Printf("Error analyzing article %d with model %s: %v", article.ID, m.ModelName, err)
								return err
							}
							_, err = db.InsertLLMScore(conn, score)
							if err != nil {
								log.Printf("Error inserting LLM score for article %d model %s: %v", article.ID, m.ModelName, err)
							}
							return err
						}()
						if err == nil {
							// Only count successful scores
							totalScores++
						}
					}
				}
			}()
		}

		for _, article := range articles {
			articleCh <- article
		}
		close(articleCh)
		wg.Wait()

		totalArticles += len(articles)
		offset += len(articles)
	}

	fmt.Printf("Scoring complete.\n")
	fmt.Printf("Total articles scored: %d\n", totalArticles)
	fmt.Printf("Total scores generated: %d\n", totalScores)
	apiStats.Print("LLM")
}

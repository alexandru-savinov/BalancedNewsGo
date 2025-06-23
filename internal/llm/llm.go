package llm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	// "math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models" // Added import
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
)

// ErrInvalidLLMResponse represents an invalid response from LLM service
var ErrInvalidLLMResponse = apperrors.New("Invalid response from LLM service", "llm_service_error")

// HTTP timeout for LLM requests
const defaultLLMTimeout = 30 * time.Second

// PromptVariant defines a prompt template with few-shot examples
type PromptVariant struct {
	ID       string
	Template string
	Examples []string
	Model    string // Model name for this variant
	URL      string // API endpoint URL
}

// GeneratePrompt formats the prompt template with content
func (pv *PromptVariant) FormatPrompt(content string) string {
	examplesText := strings.Join(pv.Examples, "\n")
	return fmt.Sprintf("%s\n%s\nArticle:\n%s", pv.Template, examplesText, content)
}

// DefaultPromptVariant is the standard prompt template for analyzing articles
var DefaultPromptVariant = PromptVariant{
	ID: "default",
	Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) " +
		"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', " +
		"and 'confidence'. Do not include any other text or formatting.",
	Examples: []string{
		`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
		`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
		`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
	},
}

// Returns (compositeScore, confidence, error)
func ComputeCompositeScoreWithConfidence(scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	return ComputeCompositeScoreWithConfidenceFixed(scores, cfg)
}

func ComputeCompositeScore(scores []db.LLMScore, cfg *CompositeScoreConfig) float64 {
	score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, cfg)
	if err != nil {
		log.Printf("[ERROR] Error computing composite score: %v", err)
		return 0.0
	}
	return score
}

func parseLLMAPIResponse(body []byte) (string, error) {
	var directResponse struct {
		Text   string `json:"text"`
		Result string `json:"result"`
		Output string `json:"output"`
	}

	if err := json.Unmarshal(body, &directResponse); err == nil {
		if directResponse.Text != "" {
			return directResponse.Text, nil
		}
		if directResponse.Result != "" {
			return directResponse.Result, nil
		}
		if directResponse.Output != "" {
			return directResponse.Output, nil
		}
	}

	var genericResponse map[string]interface{}
	if err := json.Unmarshal(body, &genericResponse); err == nil {
		if errorField, ok := genericResponse["error"].(map[string]interface{}); ok {
			if message, ok := errorField["message"].(string); ok {
				errType := errorField["type"]
				errCode := errorField["code"]
				if strings.Contains(strings.ToLower(message), "rate limit") {
					return "", ErrBothLLMKeysRateLimited
				}
				return "", apperrors.HandleError(
					fmt.Errorf("API error: %s (type: %v, code: %v)", message, errType, errCode),
					"LLM service error response",
				)
			}
		}
	}

	var standardResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &standardResp); err != nil {
		return "", apperrors.HandleError(err, "failed to parse LLM response")
	}

	if len(standardResp.Choices) == 0 || standardResp.Choices[0].Message.Content == "" {
		return "", ErrInvalidLLMResponse
	}

	return standardResp.Choices[0].Message.Content, nil
}

// LLMClient provides methods to analyze articles using language models
type LLMClient struct {
	client     *http.Client
	cache      *Cache
	db         *sqlx.DB
	llmService LLMService
	config     *CompositeScoreConfig
}

// ArticleAnalysis represents the full analysis results for an article
type ArticleAnalysis struct {
	ArticleID       int64                  `json:"article_id"`
	Scores          []db.LLMScore          `json:"scores"`
	CompositeScore  float64                `json:"composite_score"`
	Confidence      float64                `json:"confidence"`
	CategoryScores  map[string]float64     `json:"category_scores"`
	DetailedResults map[string]interface{} `json:"detailed_results"`
	CreatedAt       time.Time              `json:"created_at"`
}

func (c *LLMClient) SetHTTPLLMTimeout(timeout time.Duration) {
	httpService, ok := c.llmService.(*HTTPLLMService)
	if ok && httpService != nil && httpService.client != nil {
		httpService.client.SetTimeout(timeout)
	}
}

// GetHTTPLLMTimeout returns the current HTTP timeout for the LLM service.
// It defaults to defaultLLMTimeout if the specific service or client is not configured as expected.
func (c *LLMClient) GetHTTPLLMTimeout() time.Duration {
	httpService, ok := c.llmService.(*HTTPLLMService)
	// Ensure all parts of the chain are non-nil before dereferencing
	if ok && httpService != nil && httpService.client != nil && httpService.client.GetClient() != nil {
		return httpService.client.GetClient().Timeout
	}
	// Fallback to the package-level default LLM timeout if not specifically set or accessible
	log.Printf("[GetHTTPLLMTimeout] Warning: Could not retrieve specific timeout from HTTPLLMService, "+
		"returning default: %v", defaultLLMTimeout)
	return defaultLLMTimeout
}

func NewLLMClient(dbConn *sqlx.DB) (*LLMClient, error) {
	cache := NewCache()

	// Get OpenRouter configuration
	primaryKey := os.Getenv("LLM_API_KEY")
	backupKey := os.Getenv("LLM_API_KEY_SECONDARY")
	baseURL := os.Getenv("LLM_BASE_URL")

	// Return error for missing primary key instead of panicking
	if primaryKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY not set")
	}

	// Load configuration from file
	config, err := LoadCompositeScoreConfig()
	if err != nil {
		log.Printf("[ERROR] Failed to load composite score config: %v", err)
		// Return the error instead of panicking
		return nil, fmt.Errorf("failed to load composite score config: %w", err)
	}

	// Create resty client with timeout
	restyClient := resty.New()
	restyClient.SetTimeout(defaultLLMTimeout)

	// Disable keep-alive in test environments to prevent hanging processes
	if os.Getenv("TEST_MODE") == "true" || os.Getenv("NO_AUTO_ANALYZE") == "true" {
		restyClient.SetTransport(&http.Transport{
			DisableKeepAlives: true,
		})
	}

	// Initialize service with OpenRouter configuration
	service := NewHTTPLLMService(restyClient, primaryKey, backupKey, baseURL)

	return &LLMClient{
		client:     &http.Client{},
		cache:      cache,
		db:         dbConn,
		llmService: service,
		config:     config,
	}, nil // Return nil error on success
}

// GetConfig returns the loaded configuration for the client.
func (c *LLMClient) GetConfig() *CompositeScoreConfig {
	// Maybe add logic here to load config if nil?
	// For now, just return the stored config.
	return c.config
}

func (c *LLMClient) analyzeContent(articleID int64, content string, model string) (*db.LLMScore, error) {
	log.Printf("[analyzeContent] Entry: articleID=%d, model=%s", articleID, model)
	contentHash := hashContent(content)

	if cached, ok := c.cache.Get(contentHash, model); ok {
		return cached, nil
	}

	// Load composite score config to get the model configuration
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load composite score config: %w", err)
	}

	// Find the model in the configuration to get its URL and perspective
	var modelConfig *ModelConfig
	for _, m := range cfg.Models {
		if m.ModelName == model {
			modelConfig = &m
			break
		}
	}

	if modelConfig == nil {
		return nil, fmt.Errorf("model %s not found in configuration", model)
	}

	generalPrompt := PromptVariant{
		ID: "default",
		Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) " +
			"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', " +
			"and 'confidence'. Do not include any other text or formatting.",
		Examples: []string{
			`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
			`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
			`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
		},
		Model: modelConfig.ModelName,
		URL:   modelConfig.URL,
	}

	scoreVal, explanation, confidence, _, err := c.callLLM(articleID, model, generalPrompt, content)
	if err != nil {
		return nil, err
	}

	meta := fmt.Sprintf(`{"explanation": %q, "confidence": %.3f, "perspective": %q}`,
		explanation, confidence, modelConfig.Perspective)

	score := &db.LLMScore{
		ArticleID: articleID,
		Model:     model,
		Score:     scoreVal,
		Metadata:  meta,
		CreatedAt: time.Now(),
		Version:   1, // Set version explicitly as integer
	}

	c.cache.Set(contentHash, model, score)

	return score, nil
}

func (c *LLMClient) ProcessUnscoredArticles() error {
	query := `
	SELECT a.* FROM articles a
	WHERE NOT EXISTS (
		SELECT 1 FROM llm_scores s
		WHERE s.article_id = a.id
	)
	`
	var articles []db.Article
	if err := c.db.Select(&articles, query); err != nil {
		return err
	}

	for _, article := range articles {
		if err := c.AnalyzeAndStore(&article); err != nil {
			log.Printf("Failed to analyze article ID %d: %v", article.ID, err)
		}
	}

	return nil
}

func (c *LLMClient) AnalyzeAndStore(article *db.Article) error {
	if c.config == nil || len(c.config.Models) == 0 {
		log.Printf("[ERROR] LLMClient config is nil or has no models defined")
		return fmt.Errorf("LLMClient config is nil or has no models defined")
	}

	var lastErr error

	for _, m := range c.config.Models {
		log.Printf("[DEBUG][AnalyzeAndStore] Article %d | Perspective: %s | ModelName passed: %s | URL: %s",
			article.ID, m.Perspective, m.ModelName, m.URL)
		score, err := c.analyzeContent(article.ID, article.Content, m.ModelName)
		if err != nil {
			log.Printf("Error analyzing article %d with model %s: %v", article.ID, m.ModelName, err)
			lastErr = fmt.Errorf("error analyzing article %d with model %s: %w", article.ID, m.ModelName, err)
			continue
		}

		_, err = db.InsertLLMScore(c.db, score)
		if err != nil {
			log.Printf("Error inserting LLM score for article %d model %s: %v", article.ID, m.ModelName, err)
			lastErr = fmt.Errorf("failed to insert LLM score: %w", err)
			// Don't break here, try other models
		}
	}

	return lastErr // Return the last error encountered
}

// ReanalyzeArticle performs a complete reanalysis of an article using all configured models
func (c *LLMClient) ReanalyzeArticle(ctx context.Context, articleID int64, scoreManager *ScoreManager) error {
	log.Printf("[ReanalyzeArticle %d] Starting reanalysis", articleID)
	if scoreManager != nil {
		scoreManager.SetProgress(articleID, &models.ProgressState{
			Status:  "InProgress",
			Step:    "Starting reanalysis",
			Message: "Initiating reanalysis process.",
			Percent: 5,
		})
	}
	tx, err := c.db.Beginx()
	if err != nil {
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Begin Transaction", Message: "Failed to begin database transaction", Error: err.Error()})
		}
		return fmt.Errorf("failed to begin transaction for reanalysis of article %d: %w", articleID, err)
	}

	// Defer a function to handle commit or rollback
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[ReanalyzeArticle %d] Recovered from panic: %v. Rolling back transaction.", articleID, p)
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("[ReanalyzeArticle %d] Error rolling back transaction after panic: %v", articleID, rbErr)
			}
			panic(p) // Re-throw panic after Rollback
		} else if err != nil {
			log.Printf("[ReanalyzeArticle %d] Error occurred: %v. Rolling back transaction.", articleID, err)
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("[ReanalyzeArticle %d] Error rolling back transaction: %v (original error: %v)", articleID, rbErr, err)
			}
		} else {
			log.Printf("[ReanalyzeArticle %d] Committing transaction.", articleID)
			commitErr := tx.Commit()
			if commitErr != nil {
				log.Printf("[ReanalyzeArticle %d] Error committing transaction: %v", articleID, commitErr)
				err = fmt.Errorf("failed to commit transaction for reanalysis of article %d: %w", articleID, commitErr) // Assign commitErr to err
				if scoreManager != nil {
					scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Commit Transaction", Message: "Failed to commit transaction", Error: err.Error()})
				}
			} else {
				log.Printf("[ReanalyzeArticle %d] Transaction committed successfully.", articleID)
			}
		}
	}()

	log.Printf("[ReanalyzeArticle %d] Deleting existing non-ensemble scores", articleID)
	if scoreManager != nil {
		scoreManager.SetProgress(articleID, &models.ProgressState{
			Status:  "InProgress",
			Step:    "Deleting old scores",
			Message: "Removing previous non-ensemble analysis data.",
			Percent: 10,
		})
	}
	_, delErr := tx.ExecContext(ctx, "DELETE FROM llm_scores WHERE article_id = ? AND model != 'ensemble'", articleID)
	if delErr != nil {
		err = fmt.Errorf("failed to delete existing non-ensemble scores for article %d: %w", articleID, delErr)
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Delete Scores", Message: "Failed to delete existing non-ensemble scores", Error: err.Error()})
		}
		return err // Defer will handle rollback
	}

	var article db.Article
	log.Printf("[ReanalyzeArticle %d] Fetching article data", articleID)
	if scoreManager != nil {
		scoreManager.SetProgress(articleID, &models.ProgressState{
			Status:  "InProgress",
			Step:    "Fetching article data",
			Message: "Loading article content.",
			Percent: 15,
		})
	}
	fetchArticleErr := tx.GetContext(ctx, &article, "SELECT * FROM articles WHERE id = ?", articleID)
	if fetchArticleErr != nil {
		err = fmt.Errorf("failed to fetch article %d for reanalysis: %w", articleID, fetchArticleErr)
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Fetch Article", Message: "Failed to fetch article data", Error: err.Error()})
		}
		return err // Defer will handle rollback
	}

	log.Printf("[ReanalyzeArticle %d] Fetched article: Title='%.50s'", articleID, article.Title)
	if c.config == nil {
		log.Printf("[ReanalyzeArticle %d] Error: LLMClient config is not loaded.", articleID)
		var loadErr error
		c.config, loadErr = LoadCompositeScoreConfig()
		if loadErr != nil {
			err = fmt.Errorf("LLMClient config is not loaded and fallback failed for article %d: %w", articleID, loadErr)
			if scoreManager != nil {
				scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Load Config", Message: "Failed to load LLM configuration", Error: loadErr.Error()})
			}
			return err // Defer will handle rollback
		}
		log.Printf("[ReanalyzeArticle %d] Loaded config via fallback.", articleID)
	}
	cfg := c.config
	totalModels := len(cfg.Models)
	currentModelNum := 0

	for _, modelConfig := range cfg.Models {
		currentModelNum++
		modelProgressPercent := 15 + int(float64(currentModelNum)/float64(totalModels)*50.0)
		log.Printf("[ReanalyzeArticle %d] Calling analyzeContent for model: %s", articleID, modelConfig.ModelName)
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{
				Status:  "InProgress",
				Step:    fmt.Sprintf("Analyzing with %s", modelConfig.ModelName),
				Message: fmt.Sprintf("Processing with model %d of %d.", currentModelNum, totalModels),
				Percent: modelProgressPercent,
			})
		}

		scoreDataStruct, analyzeErr := c.analyzeContent(article.ID, article.Content, modelConfig.ModelName)
		if analyzeErr != nil {
			log.Printf("[ReanalyzeArticle %d] Error from analyzeContent for %s: %v", articleID, modelConfig.ModelName, analyzeErr)
			if scoreManager != nil {
				scoreManager.SetProgress(articleID, &models.ProgressState{
					Status:  "InProgress", // Still in progress, but this model failed
					Step:    fmt.Sprintf("Error with %s", modelConfig.ModelName),
					Message: fmt.Sprintf("Failed to analyze with %s: %v", modelConfig.ModelName, analyzeErr),
					Percent: modelProgressPercent,
					Error:   analyzeErr.Error(),
				})
			}
			// Decide if we should continue with other models or return.
			// For now, let's continue, and the error will be part of the final aggregation if needed.
			// If a specific model error should halt the whole process, 'return analyzeErr' here.
			continue // Continue to the next model
		}
		log.Printf("[ReanalyzeArticle %d] analyzeContent successful for: %s. Score: %.2f", articleID, modelConfig.ModelName, scoreDataStruct.Score)

		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{
				Status:  "InProgress",
				Step:    fmt.Sprintf("Storing score for %s", modelConfig.ModelName),
				Message: fmt.Sprintf("Saving result from model %s.", modelConfig.ModelName),
				Percent: modelProgressPercent + 2, // Arbitrary small increment
			})
		}

		// Ensure Version and CreatedAt are set. analyzeContent should handle this.
		if scoreDataStruct.Version == 0 {
			scoreDataStruct.Version = 1 // Default version if not set
		}
		if scoreDataStruct.CreatedAt.IsZero() {
			scoreDataStruct.CreatedAt = time.Now().UTC()
		}

		log.Printf("[ReanalyzeArticle %d] Attempting to insert/update score for model %s using db.InsertLLMScore (transactional)", articleID, modelConfig.ModelName)
		_, insertErr := db.InsertLLMScore(tx, scoreDataStruct) // Use tx and *db.LLMScore
		if insertErr != nil {
			err = apperrors.Wrap(insertErr, fmt.Sprintf("failed to insert/update score for model %s for article %d", modelConfig.ModelName, articleID), "db_insert_error")
			log.Printf("[ReanalyzeArticle %d] %v", articleID, err)
			if scoreManager != nil {
				scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Insert Score", Message: fmt.Sprintf("Failed to insert score for %s", modelConfig.ModelName), Error: err.Error()})
			}
			return err // Defer will handle rollback
		}
		log.Printf("[ReanalyzeArticle %d] Successfully inserted/updated score for article %d, model %s", articleID, articleID, modelConfig.ModelName)
	}

	// If loop completed, err might still be set by a previous non-fatal error or a commit failure in defer.
	// If a fatal error occurred in the loop and returned, this part is skipped.
	if err != nil {
		return err
	}

	log.Printf("[ReanalyzeArticle %d] Calculating composite score after individual model scoring.", articleID) // Corrected log
	if scoreManager != nil {
		scoreManager.SetProgress(articleID, &models.ProgressState{
			Status:  "InProgress",
			Step:    "Calculating composite score",
			Message: "Aggregating results for final score.",
			Percent: 80,
		})
	}

	var currentScores []db.LLMScore
	fetchScoresErr := tx.SelectContext(ctx, &currentScores, "SELECT * FROM llm_scores WHERE article_id = ? AND model != 'ensemble'", articleID)
	if fetchScoresErr != nil {
		err = fmt.Errorf("failed to fetch scores from transaction for composite calculation for article %d: %w", articleID, fetchScoresErr)
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Fetch Scores for Composite", Message: "Failed to fetch scores for composite calculation", Error: err.Error()})
		}
		return err // Defer will rollback
	}
	log.Printf("[ReanalyzeArticle %d] Found %d non-ensemble scores in transaction for composite calculation.", articleID, len(currentScores)) // Corrected log

	finalScore, confidence, calcErr := ComputeCompositeScoreWithConfidenceFixed(currentScores, cfg)
	if calcErr != nil {
		log.Printf("[ReanalyzeArticle %d] Error calculating composite score: %v. Proceeding with zero values.", articleID, calcErr)
		finalScore = 0
		confidence = 0
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{
				Status:  "InProgress", // Or "Warning" if such a state exists
				Step:    "Composite Score Calculation Error",
				Message: fmt.Sprintf("Error calculating composite score: %v. Proceeding with zero values.", calcErr),
				Percent: 85,
				Error:   calcErr.Error(), // Log the error but don't make the whole reanalysis fail
			})
		}
	}

	subResults := make([]map[string]interface{}, 0, len(currentScores))
	for _, s := range currentScores {
		var currentSubConfidence float64 = 0.0
		var explanation string = ""
		var metaOut map[string]interface{}
		if s.Metadata != "" {
			if unmarshalErr := json.Unmarshal([]byte(s.Metadata), &metaOut); unmarshalErr == nil {
				if confVal, ok := metaOut["confidence"].(float64); ok {
					currentSubConfidence = confVal
				}
				if explVal, ok := metaOut["explanation"].(string); ok {
					explanation = explVal
				}
			} else {
				log.Printf("[ReanalyzeArticle %d] Error unmarshalling metadata for model %s score ID %d: %v", articleID, s.Model, s.ID, unmarshalErr)
			}
		}
		perspective := MapModelToPerspective(s.Model, cfg)
		if perspective == "" {
			perspective = "unknown"
		}
		subResults = append(subResults, map[string]interface{}{
			"model":       s.Model,
			"score":       s.Score,
			"confidence":  currentSubConfidence,
			"explanation": explanation,
			"perspective": perspective,
		})
	}

	ensembleMetaMap := map[string]any{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"sub_results": subResults,
		"final_aggregation": map[string]any{
			"weighted_mean": finalScore,
			"variance":      1.0 - confidence,
			"confidence":    confidence,
		},
	}
	metaBytes, marshalErr := json.Marshal(ensembleMetaMap)
	if marshalErr != nil {
		err = fmt.Errorf("failed to marshal ensemble metadata for article %d: %w", articleID, marshalErr)
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Marshal Ensemble Metadata", Message: "Failed to marshal ensemble metadata", Error: err.Error()})
		}
		return err // Defer will rollback
	}

	ensembleLLMScore := &db.LLMScore{
		ArticleID: articleID,
		Model:     "ensemble",
		Score:     finalScore,
		Metadata:  string(metaBytes),
		Version:   1,
		CreatedAt: time.Now().UTC(),
	}

	log.Printf("[ReanalyzeArticle %d] Attempting to insert/update ensemble score using db.InsertLLMScore (transactional)", articleID)
	_, ensembleInsertErr := db.InsertLLMScore(tx, ensembleLLMScore)
	if ensembleInsertErr != nil {
		err = fmt.Errorf("failed to insert/update ensemble score for article %d: %w", articleID, ensembleInsertErr)
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Store Ensemble Score", Message: "Failed to store ensemble score", Error: err.Error()})
		}
		return err // Defer will rollback
	}
	log.Printf("[ReanalyzeArticle %d] Successfully inserted/updated ensemble score for article %d.", articleID, articleID)

	log.Printf("[ReanalyzeArticle %d] Updating article table with composite score and status in transaction.", articleID)
	if scoreManager != nil {
		scoreManager.SetProgress(articleID, &models.ProgressState{
			Status:  "InProgress",
			Step:    "Updating article table",
			Message: "Updating the main article with the new score.",
			Percent: 95,
		})
	}
	_, updateErr := tx.ExecContext(ctx,
		"UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'llm', status = 'processed' WHERE id = ?",
		finalScore, confidence, articleID)
	if updateErr != nil {
		err = fmt.Errorf("failed to update article score and status for article %d: %w", articleID, updateErr)
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{Status: "Error", Step: "Update Article Score", Message: "Failed to update article score in main table", Error: err.Error(), Percent: 98})
		}
		return err // Defer will rollback
	}
	log.Printf("[ReanalyzeArticle %d] Successfully updated article table in transaction for article %d.", articleID, articleID)

	log.Printf("[ReanalyzeArticle %d] Reanalysis operations within transaction complete. Preparing to commit.", articleID)
	if scoreManager != nil {
		scoreManager.SetProgress(articleID, &models.ProgressState{
			Status:  "InProgress", // Will change to "Completed" after successful commit
			Step:    "Finalizing",
			Message: "Reanalysis process near completion.",
			Percent: 99,
		})
	}

	return err // err will be nil if all ops succeeded, or will contain commitErr if commit failed in defer
}

func (c *LLMClient) AnalyzeContent(articleID int64, content string, model string, url string, scoreManager *ScoreManager) (*db.LLMScore, error) { // Add scoreManager
	return c.analyzeContent(articleID, content, model)
}

func (c *LLMClient) GetArticle(articleID int64) (db.Article, error) {
	var article db.Article
	err := c.db.Get(&article, "SELECT * FROM articles WHERE id = ?", articleID)
	return article, err
}

func (c *LLMClient) DeleteScores(articleID int64) error {
	_, err := c.db.Exec("DELETE FROM llm_scores WHERE article_id = ?", articleID)
	return err
}

func (c *LLMClient) FetchScores(articleID int64) ([]db.LLMScore, error) {
	return db.FetchLLMScores(c.db, articleID)
}

// ScoreWithModel uses a single model to score content
func (c *LLMClient) ScoreWithModel(article *db.Article, modelName string) (float64, error) {
	log.Printf("[DEBUG][CONFIDENCE] ScoreWithModel called for article %d with model %s", article.ID, modelName)

	// Create a prompt variant with the specified model
	promptVariant := DefaultPromptVariant
	promptVariant.Model = modelName

	// Find the model in the configuration to get its URL
	if c.config != nil {
		for _, m := range c.config.Models {
			if m.ModelName == modelName {
				promptVariant.URL = m.URL
				break
			}
		}
	}

	// Use the LLM service directly to handle rate limiting properly
	score, confidence, err := c.llmService.ScoreContent(context.Background(), promptVariant, article)

	if err != nil {
		// Specifically check for rate limit errors first
		if errors.Is(err, ErrBothLLMKeysRateLimited) {
			return 0, ErrBothLLMKeysRateLimited
		}

		// Check for service unavailable
		if strings.Contains(strings.ToLower(err.Error()), "503") ||
			strings.Contains(strings.ToLower(err.Error()), "service unavailable") {
			return 0, apperrors.New("LLM service unavailable", "llm_service_error")
		}

		// For any other errors, return a more descriptive error with llm_service_error code
		return 0, apperrors.Wrap(err, "llm_service_error", fmt.Sprintf("scoring with model %s failed", modelName))
	}

	// If db is nil, return the score directly (used in tests)
	if c.db == nil {
		log.Printf("[DEBUG][CONFIDENCE] Test mode detected (nil db), returning score without storage: %.4f", score)
		return score, nil
	}

	// Create and store the score in the database
	explanation := "Generated by model " + modelName // We don't have explanation from the interface
	meta := fmt.Sprintf(`{"explanation": %q, "confidence": %.3f}`, explanation, confidence)
	llmScore := &db.LLMScore{
		ArticleID: article.ID,
		Model:     modelName,
		Score:     score,
		Metadata:  meta,
		CreatedAt: time.Now(),
		Version:   1, // Set version explicitly as integer
	}

	log.Printf("[DEBUG][CONFIDENCE] Successfully scored and stored: article=%d, model=%s, score=%.4f",
		article.ID, modelName, score)

	_, err = db.InsertLLMScore(c.db, llmScore)
	if err != nil {
		log.Printf("[ERROR][CONFIDENCE] Failed to store score in database: %v", err)
		// Don't return error here, just log it. The score itself was obtained.
	}

	return score, nil
}

func (c *LLMClient) StoreEnsembleScore(article *db.Article) (float64, error) {
	scores, err := c.FetchScores(article.ID)
	if err != nil {
		return 0, fmt.Errorf("fetching scores for article %d: %w", article.ID, err)
	}

	if c.config == nil {
		log.Printf("[StoreEnsembleScore %d] Error: LLMClient config is nil.", article.ID)
		// Attempt fallback load
		var loadErr error
		c.config, loadErr = LoadCompositeScoreConfig()
		if loadErr != nil {
			log.Printf("[StoreEnsembleScore %d] Error loading config fallback: %v", article.ID, loadErr)
			return 0, fmt.Errorf("LLMClient config is nil and fallback failed: %w", loadErr)
		}
	}

	score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(scores, c.config)
	if err != nil {
		log.Printf("Error calculating composite score for article %d: %v", article.ID, err)
		// Proceed with zero score in case of error
		score = 0
		confidence = 0
	}

	// Prepare metadata for ensemble score
	subResults := make([]map[string]interface{}, 0, len(scores))
	for _, s := range scores {
		// Skip ensemble model itself
		if strings.ToLower(s.Model) == "ensemble" {
			continue
		}

		// Extract confidence and explanation from metadata
		var confidence float64 = 0.0
		var explanation string = ""
		var meta map[string]interface{}

		if s.Metadata != "" {
			if err := json.Unmarshal([]byte(s.Metadata), &meta); err == nil {
				if confVal, ok := meta["confidence"].(float64); ok {
					confidence = confVal
				}
				if explVal, ok := meta["explanation"].(string); ok {
					explanation = explVal
				}
			}
		}

		// Map model to perspective
		perspective := MapModelToPerspective(s.Model, c.config)
		if perspective == "" {
			perspective = "unknown"
		}

		subResults = append(subResults, map[string]interface{}{
			"model":       s.Model,
			"score":       s.Score,
			"confidence":  confidence,
			"explanation": explanation,
			"perspective": perspective,
		})
	}

	meta := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"sub_results": subResults,
		"final_aggregation": map[string]interface{}{
			"weighted_mean": score,
			"variance":      1.0 - confidence,
			"confidence":    confidence,
		},
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		log.Printf("Warning: Failed to marshal metadata: %v", err)
		metaBytes = []byte("{}")
	}
	ensembleScore := &db.LLMScore{
		ArticleID: article.ID,
		Model:     "ensemble",
		Score:     score,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
		Version:   1, // Set version explicitly to match schema expectation (as an integer)
	}

	_, err = c.db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at, version)
		VALUES (:article_id, :model, :score, :metadata, :created_at, :version) ON CONFLICT(article_id, model) DO UPDATE SET
		score = EXCLUDED.score,
		metadata = EXCLUDED.metadata,
		created_at = EXCLUDED.created_at,
		version = EXCLUDED.version`, ensembleScore)
	if err != nil {
		return score, fmt.Errorf("inserting/updating ensemble score for article %d: %w", article.ID, err)
	}

	// Also update the main article table
	err = db.UpdateArticleScore(c.db, article.ID, score, confidence)
	if err != nil {
		return score, fmt.Errorf("updating article score for article %d: %w", article.ID, err)
	}

	return score, nil
}

func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

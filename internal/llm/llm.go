package llm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
)

// ErrInvalidLLMResponse represents an invalid response from LLM service
var ErrInvalidLLMResponse = apperrors.New("Invalid response from LLM service", "llm_service_error")

// HTTP timeout for LLM requests
const defaultLLMTimeout = 30 * time.Second

const (
	LabelUnknown = "unknown"
	LabelLeft    = "left"
	LabelRight   = "right"
	LabelNeutral = "neutral"
)

type ModelConfig struct {
	Perspective string `json:"perspective"`
	ModelName   string `json:"modelName"`
	URL         string `json:"url"`
}

type CompositeScoreConfig struct {
	Formula          string             `json:"formula"`
	Weights          map[string]float64 `json:"weights"`
	MinScore         float64            `json:"min_score"`
	MaxScore         float64            `json:"max_score"`
	DefaultMissing   float64            `json:"default_missing"`
	HandleInvalid    string             `json:"handle_invalid"`
	ConfidenceMethod string             `json:"confidence_method"`
	MinConfidence    float64            `json:"min_confidence"`
	MaxConfidence    float64            `json:"max_confidence"`
	Models           []ModelConfig      `json:"models"`
}

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
		"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
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

func isInvalid(f float64) bool {
	return math.IsNaN(f) || math.IsInf(f, 0)
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
			"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
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

	for _, m := range c.config.Models {
		log.Printf("[DEBUG][AnalyzeAndStore] Article %d | Perspective: %s | ModelName passed: %s | URL: %s", article.ID, m.Perspective, m.ModelName, m.URL)
		score, err := c.analyzeContent(article.ID, article.Content, m.ModelName)
		if err != nil {
			log.Printf("Error analyzing article %d with model %s: %v", article.ID, m.ModelName, err)
			continue
		}

		_, err = db.InsertLLMScore(c.db, score)
		if err != nil {
			log.Printf("Error inserting LLM score for article %d model %s: %v", article.ID, m.ModelName, err)
		}
	}

	return nil
}

// ReanalyzeArticle performs a complete reanalysis of an article using all configured models
func (c *LLMClient) ReanalyzeArticle(articleID int64) error {
	log.Printf("[ReanalyzeArticle %d] Starting reanalysis", articleID)
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}

	log.Printf("[ReanalyzeArticle %d] Deleting existing scores", articleID)
	_, err = tx.Exec("DELETE FROM llm_scores WHERE article_id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}
		return err
	}

	var article db.Article
	log.Printf("[ReanalyzeArticle %d] Fetching article data", articleID)
	err = tx.Get(&article, "SELECT * FROM articles WHERE id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}
		return err
	}

	log.Printf("[ReanalyzeArticle %d] Fetched article: Title='%.50s'", articleID, article.Title)
	log.Printf("[ReanalyzeArticle %d] Starting analysis loop for models", articleID)

	// Use the client's already loaded config
	if c.config == nil {
		log.Printf("[ReanalyzeArticle %d] Error: LLMClient config is not loaded.", articleID)
		// Try loading it again as a fallback?
		var loadErr error
		c.config, loadErr = LoadCompositeScoreConfig()
		if loadErr != nil {
			log.Printf("[ReanalyzeArticle %d] Error loading config fallback: %v", articleID, loadErr)
			_ = tx.Rollback() // Attempt rollback before returning
			return fmt.Errorf("LLMClient config is not loaded and fallback failed: %w", loadErr)
		}
		log.Printf("[ReanalyzeArticle %d] Loaded config via fallback.", articleID)
	}
	cfg := c.config // Use the client's config

	// Try each model in sequence
	for _, modelConfig := range cfg.Models {
		log.Printf("[ReanalyzeArticle %d] Calling analyzeContent for model: %s", articleID, modelConfig.ModelName)
		score, analyzeErr := c.analyzeContent(article.ID, article.Content, modelConfig.ModelName)
		if analyzeErr != nil {
			log.Printf("[ReanalyzeArticle %d] Error from analyzeContent for %s: %v", articleID, modelConfig.ModelName, analyzeErr)
			continue
		}
		log.Printf("[ReanalyzeArticle %d] analyzeContent successful for: %s. Score: %.2f", articleID, modelConfig.ModelName, score.Score)

		_, insertErr := tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata)
			VALUES (:article_id, :model, :score, :metadata)`, score)
		if insertErr != nil {
			log.Printf("[ReanalyzeArticle %d] Error inserting score for %s: %v", articleID, modelConfig.ModelName, insertErr)
			// Decide whether to continue or rollback/fail
			// continue // Option 1: Log and continue with other models
			if rbErr := tx.Rollback(); rbErr != nil { // Option 2: Rollback and fail
				log.Printf("[ReanalyzeArticle %d] tx.Rollback() failed after insert error: %v", articleID, rbErr)
			}
			return fmt.Errorf("inserting score for %s: %w", modelConfig.ModelName, insertErr)
		} else {
			log.Printf("[ReanalyzeArticle %d] Successfully inserted score for: %s", articleID, modelConfig.ModelName)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("[ReanalyzeArticle %d] Error committing transaction: %v", articleID, err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Calculate composite score AFTER successful commit
	scores, err := c.FetchScores(article.ID)
	if err != nil {
		log.Printf("[ReanalyzeArticle %d] Error fetching scores post-commit: %v", articleID, err)
		return fmt.Errorf("failed to fetch scores for ensemble calculation post-commit: %w", err)
	}

	finalScore, confidence, err := ComputeCompositeScoreWithConfidenceFixed(scores, cfg)
	if err != nil {
		// Don't return error, just log. Store 0 if calculation fails.
		log.Printf("[ReanalyzeArticle %d] Error calculating composite score post-commit: %v. Storing 0.", articleID, err)
		finalScore = 0
		confidence = 0
	}

	// Store ensemble score in llm_scores table
	meta := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"final_aggregation": map[string]interface{}{
			"weighted_mean": finalScore,
			"variance":      1.0 - confidence, // Example variance calculation
			"confidence":    confidence,
		},
	}
	metaBytes, _ := json.Marshal(meta) // Error handling omitted for brevity
	ensembleScore := &db.LLMScore{
		ArticleID: articleID,
		Model:     "ensemble",
		Score:     finalScore,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}

	_, err = c.db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
		VALUES (:article_id, :model, :score, :metadata, :created_at) ON CONFLICT(article_id, model) DO UPDATE SET score = excluded.score, metadata = excluded.metadata, created_at = excluded.created_at`, ensembleScore)
	if err != nil {
		log.Printf("[ReanalyzeArticle %d] Error inserting/updating ensemble score post-commit: %v", articleID, err)
		// Decide if this is fatal or just a warning
		// return fmt.Errorf("failed to insert/update ensemble score: %w", err)
	}

	// Update the main article score
	err = db.UpdateArticleScore(c.db, articleID, finalScore, confidence)
	if err != nil {
		log.Printf("[ReanalyzeArticle %d] Error updating article score post-commit: %v", articleID, err)
		// Decide if this is fatal or just a warning
		// return fmt.Errorf("failed to update article score: %w", err)
	}

	log.Printf("[ReanalyzeArticle %d] Completed successfully. Score: %.2f, Confidence: %.2f", articleID, finalScore, confidence)
	return nil
}

func (c *LLMClient) AnalyzeContent(articleID int64, content string, model string, url string) (*db.LLMScore, error) {
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
		return 0, fmt.Errorf("calculating composite score for article %d: %w", article.ID, err)
	}

	// Store ensemble score in llm_scores table
	meta := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"final_aggregation": map[string]interface{}{
			"weighted_mean": score,
			"variance":      1.0 - confidence,
			"confidence":    confidence,
		},
	}
	metaBytes, _ := json.Marshal(meta)
	ensembleScore := &db.LLMScore{
		ArticleID: article.ID,
		Model:     "ensemble",
		Score:     score,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}

	_, err = c.db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
		VALUES (:article_id, :model, :score, :metadata, :created_at) ON CONFLICT(article_id, model) DO UPDATE SET score = excluded.score, metadata = excluded.metadata, created_at = excluded.created_at`, ensembleScore)
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

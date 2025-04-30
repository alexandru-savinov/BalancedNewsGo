package llm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
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

// compositeScoreConfig holds test override when len(Models)==0
var compositeScoreConfig *CompositeScoreConfig

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
func ComputeCompositeScoreWithConfidence(scores []db.LLMScore) (float64, float64, error) {
	// Use override config in tests if provided, else use built-in defaults
	var cfg *CompositeScoreConfig
	if compositeScoreConfig != nil {
		cfg = compositeScoreConfig
		// clear override for next invocation
		compositeScoreConfig = nil
	} else {
		cfg = &CompositeScoreConfig{
			Formula:          "average",
			Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
			MinScore:         -1.0,
			MaxScore:         1.0,
			DefaultMissing:   0.0,
			HandleInvalid:    "default",
			ConfidenceMethod: "count_valid",
		}
	}

	// Map scores to perspectives
	scoreMap := map[string]float64{"left": cfg.DefaultMissing, "center": cfg.DefaultMissing, "right": cfg.DefaultMissing}
	validModels := make(map[string]bool)
	sumScores := 0.0
	sumWeights := 0.0
	for _, s := range scores {
		model := strings.ToLower(s.Model)
		var perspective string
		switch model {
		case "left":
			perspective = "left"
		case "center":
			perspective = "center"
		case "right":
			perspective = "right"
		default:
			continue
		}
		val := s.Score
		if cfg.HandleInvalid == "ignore" && (isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore) {
			continue
		}
		if isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore {
			val = cfg.DefaultMissing
		}
		w := 1.0
		if cfg.Formula == "weighted" {
			if weight, ok := cfg.Weights[perspective]; ok {
				w = weight
			}
		}
		scoreMap[perspective] = val
		sumScores += val * w
		sumWeights += w
		validModels[perspective] = true
	}
	// Compute composite
	var composite float64
	switch cfg.Formula {
	case "weighted":
		if sumWeights > 0 {
			composite = sumScores / sumWeights
		}
	default: // average and others: simple average over three perspectives
		composite = (scoreMap["left"] + scoreMap["center"] + scoreMap["right"]) / 3.0
	}
	// Compute confidence
	var confidence float64
	switch cfg.ConfidenceMethod {
	case "count_valid":
		confidence = float64(len(validModels)) / 3.0
	case "spread":
		// normalize spread across configured score range
		span := cfg.MaxScore - cfg.MinScore
		if span > 0 {
			// difference between max and min perspective scores
			vals := []float64{scoreMap["left"], scoreMap["center"], scoreMap["right"]}
			sort.Float64s(vals)
			confidence = (vals[2] - vals[0]) / span
		}
	default:
		confidence = float64(len(validModels)) / 3.0
	}
	return composite, confidence, nil
}

func ComputeCompositeScore(scores []db.LLMScore) float64 {
	score, _, _ := ComputeCompositeScoreWithConfidence(scores)
	return score
}

func isInvalid(f float64) bool {
	return (f != f) || (f > 1e10) || (f < -1e10)
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

func NewLLMClient(dbConn *sqlx.DB) *LLMClient {
	cache := NewCache()

	// Get OpenRouter configuration
	primaryKey := os.Getenv("LLM_API_KEY")
	backupKey := os.Getenv("LLM_API_KEY_SECONDARY")
	baseURL := os.Getenv("LLM_BASE_URL")

	// Replace fatal exit with panic for missing primary key to satisfy tests
	if primaryKey == "" {
		panic("ERROR: LLM_API_KEY not set")
	}

	config := &CompositeScoreConfig{
		Formula:          "average",
		Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
		MinScore:         -1e10,
		MaxScore:         1e10,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.0,
		MaxConfidence:    1.0,
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
	}
}

func (c *LLMClient) analyzeContent(articleID int64, content string, model string) (*db.LLMScore, error) {
	log.Printf("[analyzeContent] Entry: articleID=%d, model=%s", articleID, model)
	contentHash := hashContent(content)

	if cached, ok := c.cache.Get(contentHash, model); ok {
		return cached, nil
	}

	var err error

	generalPrompt := PromptVariant{
		ID: "default",
		Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) " +
			"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
		Examples: []string{
			`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
			`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
			`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
		},
	}

	scoreVal, explanation, confidence, _, err := c.callLLM(articleID, model, generalPrompt, content)
	if err != nil {
		return nil, err
	}

	meta := fmt.Sprintf(`{"explanation": %q, "confidence": %.3f}`, explanation, confidence)

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
	cfg := &CompositeScoreConfig{
		Formula:          "average",
		Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
		MinScore:         -1e10,
		MaxScore:         1e10,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.0,
		MaxConfidence:    1.0,
	}

	for _, m := range cfg.Models {
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
	for _, m := range compositeScoreConfig.Models {
		log.Printf("[ReanalyzeArticle %d] Calling analyzeContent for model: %s", articleID, m.ModelName)
		score, err := c.analyzeContent(article.ID, article.Content, m.ModelName)
		if err != nil {
			log.Printf("[ReanalyzeArticle %d] Error from analyzeContent for %s: %v", articleID, m.ModelName, err)
			continue
		}
		log.Printf("[ReanalyzeArticle %d] analyzeContent successful for: %s. Score: %.2f", articleID, m.ModelName, score.Score)

		_, err = tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata)
			VALUES (:article_id, :model, :score, :metadata)`, score)
		if err != nil {
			log.Printf("[ReanalyzeArticle %d] Error inserting score for %s: %v", articleID, m.ModelName, err)
		} else {
			log.Printf("[ReanalyzeArticle %d] Successfully inserted score for: %s", articleID, m.ModelName)
		}
	}

	scores, err := c.FetchScores(article.ID)
	if err != nil {
		log.Printf("[ReanalyzeArticle %d] Error fetching scores: %v", articleID, err)
		return fmt.Errorf("failed to fetch scores for ensemble calculation: %w", err)
	}

	finalScore, confidence, err := ComputeCompositeScoreWithConfidence(scores)
	if err != nil {
		log.Printf("[ReanalyzeArticle %d] Error computing composite score: %v", articleID, err)
		return fmt.Errorf("failed to compute composite score: %w", err)
	}

	meta := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"final_aggregation": map[string]interface{}{
			"weighted_mean": finalScore,
			"variance":      1.0 - confidence,
		},
	}
	metaBytes, _ := json.Marshal(meta)

	ensembleScore := &db.LLMScore{
		ArticleID: article.ID,
		Model:     "ensemble",
		Score:     finalScore,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}

	_, err = tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
		VALUES (:article_id, :model, :score, :metadata, :created_at)`, ensembleScore)
	if err != nil {
		log.Printf("[ReanalyzeArticle %d] Error inserting ensemble score: %v", articleID, err)
		return fmt.Errorf("failed to insert ensemble score: %w", err)
	}

	err = db.UpdateArticleScore(c.db, article.ID, finalScore, confidence)
	if err != nil {
		log.Printf("[ReanalyzeArticle %d] Error updating article score: %v", articleID, err)
		return fmt.Errorf("failed to update article score: %w", err)
	}

	log.Printf("[ReanalyzeArticle %d] Successfully completed reanalysis with score: %.2f, confidence: %.2f",
		articleID, finalScore, confidence)
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
	// Create a prompt variant with the specified model
	promptVariant := DefaultPromptVariant
	promptVariant.Model = modelName

	// Use the LLM service directly to handle rate limiting properly
	score, _, err := c.llmService.ScoreContent(context.Background(), promptVariant, article)

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

	return score, nil
}

func (c *LLMClient) StoreEnsembleScore(article *db.Article) (float64, error) {
	scores, err := c.FetchScores(article.ID)
	if err != nil {
		return 0.0, fmt.Errorf("failed to fetch scores for ensemble calculation: %w", err)
	}

	finalScore, confidence, err := ComputeCompositeScoreWithConfidence(scores)
	if err != nil {
		return 0.0, fmt.Errorf("failed to compute composite score: %w", err)
	}

	meta := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"final_aggregation": map[string]interface{}{
			"weighted_mean": finalScore,
			"variance":      1.0 - confidence,
		},
	}
	metaBytes, _ := json.Marshal(meta)

	ensembleScore := &db.LLMScore{
		ArticleID: article.ID,
		Model:     "ensemble",
		Score:     finalScore,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}

	_, err = c.db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
		VALUES (:article_id, :model, :score, :metadata, :created_at)`, ensembleScore)
	if err != nil {
		return finalScore, fmt.Errorf("failed to insert ensemble score: %w", err)
	}

	updateErr := db.UpdateArticleScore(c.db, article.ID, finalScore, confidence)
	if updateErr != nil {
		return finalScore, fmt.Errorf("failed to update article score: %w", updateErr)
	}

	return finalScore, nil
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

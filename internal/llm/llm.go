package llm

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
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

var (
	compositeScoreConfig     *CompositeScoreConfig
	compositeScoreConfigOnce sync.Once
)

func LoadCompositeScoreConfig() (*CompositeScoreConfig, error) {
	var err error
	compositeScoreConfigOnce.Do(func() {
		const configPath = "configs/composite_score_config.json"
		f, e := os.Open(configPath)
		if e != nil {
			err = fmt.Errorf("opening composite score config %q: %w", configPath, e)
			return
		}
		defer f.Close()
		decoder := json.NewDecoder(f)
		var cfg CompositeScoreConfig
		if e := decoder.Decode(&cfg); e != nil {
			err = fmt.Errorf("decoding composite score config %q: %w", configPath, e)
			return
		}
		if len(cfg.Models) == 0 {
			err = fmt.Errorf("composite score config %q loaded but contains no models", configPath)
			return
		}
		compositeScoreConfig = &cfg
	})
	if err != nil {
		return nil, err
	}
	return compositeScoreConfig, nil
}

// Returns (compositeScore, confidence, error)
func ComputeCompositeScoreWithConfidence(scores []db.LLMScore) (float64, float64, error) {
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return 0, 0, fmt.Errorf("loading composite score config: %w", err)
	}

	scoreMap := map[string]*float64{
		"left":   nil,
		"center": nil,
		"right":  nil,
	}
	validCount := 0
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	validModels := make(map[string]bool)
	for _, s := range scores {
		model := strings.ToLower(s.Model)
		if model == LabelLeft || model == "left" {
			model = "left"
		} else if model == LabelRight || model == "right" {
			model = "right"
		} else if model == "center" {
			model = "center"
		} else {
			continue
		}
		val := s.Score
		if cfg.HandleInvalid == "ignore" && (isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore) {
			continue
		}
		if isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore {
			val = cfg.DefaultMissing
		}
		scoreMap[model] = &val
		validCount++
		validModels[model] = true
	}

	if validCount == 0 {
		return 0, 0, fmt.Errorf("no valid model scores to compute composite score (input count: %d)", len(scores))
	}

	for k, v := range scoreMap {
		score := cfg.DefaultMissing
		if v != nil {
			score = *v
		}
		w := 1.0
		if cfg.Formula == "weighted" {
			if weight, ok := cfg.Weights[k]; ok {
				w = weight
			}
		}
		weightedSum += score * w
		weightTotal += w
		sum += score
	}

	var composite float64
	switch cfg.Formula {
	case "average":
		composite = sum / 3.0
	case "weighted":
		if weightTotal > 0 {
			composite = weightedSum / weightTotal
		} else {
			composite = 0.0
		}
	case "min":
		composite = minNonNil(scoreMap, cfg.DefaultMissing)
	case "max":
		composite = maxNonNil(scoreMap, cfg.DefaultMissing)
	default:
		composite = sum / 3.0
	}

	composite = 1.0 - abs(composite)

	var confidence float64
	switch cfg.ConfidenceMethod {
	case "count_valid":
		confidence = float64(len(validModels)) / 3.0
	case "spread":
		confidence = 1.0 - scoreSpread(scoreMap)
	default:
		confidence = float64(len(validModels)) / 3.0
	}
	if confidence < cfg.MinConfidence {
		confidence = cfg.MinConfidence
	}
	if confidence > cfg.MaxConfidence {
		confidence = cfg.MaxConfidence
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

func minNonNil(m map[string]*float64, def float64) float64 {
	min := def
	first := true
	for _, v := range m {
		if v != nil {
			if first || *v < min {
				min = *v
				first = false
			}
		}
	}
	return min
}

func maxNonNil(m map[string]*float64, def float64) float64 {
	max := def
	first := true
	for _, v := range m {
		if first || (v != nil && *v > max) {
			if v != nil {
				max = *v
			}
			first = false
		}
	}
	return max
}

func scoreSpread(m map[string]*float64) float64 {
	vals := []float64{}
	for _, v := range m {
		if v != nil {
			vals = append(vals, *v)
		}
	}
	if len(vals) < 2 {
		return 0.0
	}
	sort.Float64s(vals)
	return vals[len(vals)-1] - vals[0]
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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

type LLMClient struct {
	client     *http.Client
	cache      *Cache
	db         *sqlx.DB
	llmService LLMService
	config     *CompositeScoreConfig
}

func (c *LLMClient) SetHTTPLLMTimeout(timeout time.Duration) {
	httpService, ok := c.llmService.(*HTTPLLMService)
	if ok && httpService != nil && httpService.client != nil {
		httpService.client.Timeout = timeout
	}
}

func NewLLMClient(dbConn *sqlx.DB) *LLMClient {
	client := &http.Client{}
	cache := NewCache()

	provider := os.Getenv("LLM_PROVIDER")

	var service LLMService

	config, err := LoadCompositeScoreConfig()
	if err != nil {
		log.Fatalf("Failed to load composite score config: %v", err)
	}
	if config == nil || len(config.Models) == 0 {
		log.Fatalf("Composite score config loaded but is nil or contains no models.")
	}
	log.Printf("Loaded composite score config with %d models.", len(config.Models))

	switch provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Fatal("ERROR: OPENAI_API_KEY not set, cannot use OpenAI LLM provider")
		}

		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-3.5-turbo"
		}

		service = NewHTTPLLMService(client)
	case "openrouter":
		apiKey := os.Getenv("LLM_API_KEY")
		if apiKey == "" {
			log.Fatal("ERROR: LLM_API_KEY not set, cannot use OpenRouter LLM provider")
		}
		service = NewHTTPLLMService(client)
	default:
		log.Fatalf("ERROR: LLM_PROVIDER '%s' unknown, cannot initialize LLM service", provider)
	}

	return &LLMClient{
		client:     client,
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

	var score *db.LLMScore
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

	score = &db.LLMScore{
		ArticleID: articleID,
		Model:     model,
		Score:     scoreVal,
		Metadata:  meta,
		CreatedAt: time.Now(),
	}

	score.ArticleID = articleID
	score.Model = model
	score.CreatedAt = time.Now()

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
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load composite score config: %w", err)
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
	log.Printf("[ReanalyzeArticle %d] Loading composite score config", articleID)
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load composite score config: %w", err)
	}

	log.Printf("[ReanalyzeArticle %d] Starting analysis loop for %d models", articleID, len(cfg.Models))
	for _, m := range cfg.Models {
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
	score, _, _, _, err := c.callLLM(article.ID, modelName, DefaultPromptVariant, article.Content)
	if err != nil {
		if errors.Is(err, ErrBothLLMKeysRateLimited) {
			return 0, ErrBothLLMKeysRateLimited
		}
		if strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "Service Unavailable") {
			return 0, ErrLLMServiceUnavailable
		}
		return 0, apperrors.HandleError(err, fmt.Sprintf("scoring with model %s failed", modelName))
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

package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
)

func ComputeCompositeScore(scores []db.LLMScore) float64 {
	var left, center, right *float64

	for _, s := range scores {
		switch strings.ToLower(s.Model) {
		case "left":
			left = &s.Score
		case "center":
			center = &s.Score
		case "right":
			right = &s.Score
		}
	}

	// Default to 0 if missing
	l := 0.0
	if left != nil {
		l = *left
	}
	c := 0.0
	if center != nil {
		c = *center
	}
	r := 0.0
	if right != nil {
		r = *right
	}

	avg := (l + c + r) / 3.0

	// Composite favors balance (closer to center = higher score)
	composite := 1.0 - abs(avg)

	return composite
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

type BiasConfig struct {
	Categories          []string            `json:"categories"`
	ConfidenceThreshold float64             `json:"confidence_threshold"`
	KeywordHeuristics   map[string][]string `json:"keyword_heuristics"`
}

type BiasResult struct {
	Category    string  `json:"category"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

var (
	PromptTemplate string
	BiasCfg        BiasConfig
)

func LoadPromptTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func LoadBiasConfig(path string) (BiasConfig, error) {
	var cfg BiasConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(data, &cfg)

	return cfg, err
}

func init() {
	var err error

	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	promptPath := os.Getenv("PROMPT_TEMPLATE_PATH")
	if promptPath != "" {
		log.Printf("Using prompt template path from environment: %s", promptPath)
		PromptTemplate, err = LoadPromptTemplate(promptPath)
		if err != nil {
			log.Fatalf("Failed to load prompt template from environment path: %v", err)
		}
	} else {
		PromptTemplate, err = LoadPromptTemplate("configs/prompt_template.txt")
		if err != nil {
			log.Printf("Error loading prompt template from configs/prompt_template.txt: %v", err)
			log.Fatalf("Failed to load prompt template: %v", err)
		}
	}

	BiasCfg, err = LoadBiasConfig("configs/bias_config.json")
	if err != nil {
		log.Printf("Error loading bias config from configs/bias_config.json: %v", err)
		log.Fatalf("Failed to load bias config: %v", err)
	}
}

// LLM API endpoints.
var (
	LeftModelURL   = "http://localhost:8001/analyze"
	CenterModelURL = "https://api.openai.com/v1/chat/completions"
	RightModelURL  = "http://localhost:8003/analyze"
)

// Cache key: hash(content) + model.
type cacheKey struct {
	ContentHash string
	Model       string
}

type Cache struct {
	mu    sync.RWMutex
	store map[cacheKey]*db.LLMScore
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[cacheKey]*db.LLMScore),
	}
}

func (c *Cache) Get(contentHash, model string) (*db.LLMScore, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	score, ok := c.store[cacheKey{contentHash, model}]

	return score, ok
}

func (c *Cache) Set(contentHash, model string, score *db.LLMScore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[cacheKey{contentHash, model}] = score
}

// Hash function placeholder (can replace with real hash).
func hashContent(content string) string {
	// For simplicity, use content itself (not recommended for production)
	return content
}

type LLMService interface {
	Analyze(content string) (*db.LLMScore, error)
}

type MockLLMService struct{}

func (m *MockLLMService) Analyze(content string) (*db.LLMScore, error) {
	score := 0.5 // fixed or random score
	metadata := `{"mock": true}`

	return &db.LLMScore{
		Model:    "mock",
		Score:    score,
		Metadata: metadata,
	}, nil
}

type OpenAILLMService struct {
	client *resty.Client
	model  string
	apiKey string
}

func NewOpenAILLMService(client *resty.Client, model, apiKey string) *OpenAILLMService {
	return &OpenAILLMService{
		client: client,
		model:  model,
		apiKey: apiKey,
	}
}

func (o *OpenAILLMService) Analyze(content string) (*db.LLMScore, error) {
	prompt := strings.Replace(PromptTemplate, "{{ARTICLE_CONTENT}}", content, 1)

	var resp *resty.Response
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = o.callOpenAI(prompt)
		if err == nil && resp != nil && resp.IsSuccess() {
			return o.processOpenAIResponse(resp, content)
		}

		log.Printf("OpenAI API call failed (attempt %d): %v", attempt, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return nil, errors.New("OpenAI API call failed after retries")
}

func (o *OpenAILLMService) callOpenAI(prompt string) (*resty.Response, error) {
	url := "https://api.openai.com/v1/chat/completions"

	req := o.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+o.apiKey)

	body := map[string]interface{}{
		"model":       o.model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  300,
		"temperature": 0.7,
	}
	req.SetBody(body)

	log.Printf("[OpenAI] Sending real API request to %s with model %s", url, o.model)

	resp, err := req.Post(url)

	if err != nil {
		log.Printf("[OpenAI] API request error: %v", err)
		return nil, err
	}

	log.Printf("[OpenAI] API response status: %d", resp.StatusCode())
	log.Printf("[OpenAI] API response body: %.500s", resp.String())

	if !resp.IsSuccess() {
		return resp, errors.New("API response not successful")
	}

	return resp, nil
}

func (o *OpenAILLMService) processOpenAIResponse(resp *resty.Response, content string) (*db.LLMScore, error) {
	contentResp, err := parseOpenAIResponse(resp.Body())
	if err != nil {
		log.Printf("[Analyze] Failed to parse OpenAI response: %v\nRaw response:\n%s", err, string(resp.Body()))
		return nil, err
	}

	biasResult := parseBiasResult(contentResp)
	heuristicCat := heuristicCategory(content)

	uncertain := false
	if biasResult.Category == "unknown" || biasResult.Confidence < 0.3 {
		uncertain = true
	}

	metadataMap := map[string]interface{}{
		"raw_response":       contentResp,
		"parsed_bias":        biasResult,
		"heuristic_category": heuristicCat,
		"uncertain":          uncertain,
	}
	metadataBytes, err := json.Marshal(metadataMap)
	if err != nil {
		log.Printf("Failed to marshal metadata: %v", err)
	}

	return &db.LLMScore{
		Model:    o.model,
		Score:    biasResult.Confidence,
		Metadata: string(metadataBytes),
	}, nil
}

func (o *OpenAILLMService) RobustAnalyze(content string) (*db.LLMScore, error) {
	var validScores []*db.LLMScore
	var attempts int

	for attempts < 10 && len(validScores) < 5 {
		score, err := o.Analyze(content)
		attempts++

		if err == nil && score != nil {
			var meta struct {
				ParsedBias struct {
					Category   string  `json:"Category"`
					Confidence float64 `json:"Confidence"`
				} `json:"parsed_bias"`
			}
			if err := json.Unmarshal([]byte(score.Metadata), &meta); err != nil {
				log.Printf("[RobustAnalyze] Excluding response due to metadata parse error: %v", err)
				continue
			}
			cat := strings.ToLower(strings.TrimSpace(meta.ParsedBias.Category))
			conf := meta.ParsedBias.Confidence
			if cat == "" || cat == "unknown" || conf <= 0 {
				log.Printf("[RobustAnalyze] Excluding response with category='%s' confidence=%.2f", cat, conf)
				continue
			}
			validScores = append(validScores, score)
		} else {
			log.Printf("[RobustAnalyze] Attempt %d failed: %v", attempts, err)
		}
	}

	if len(validScores) < 5 {
		return nil, fmt.Errorf("RobustAnalyze: only %d valid responses after %d attempts", len(validScores), attempts)
	}

	// Extract scores
	scores := make([]float64, 5)
	for i, s := range validScores {
		scores[i] = s.Score
	}

	// Sort scores
	sort.Float64s(scores)

	// Average middle 3
	sum := scores[1] + scores[2] + scores[3]
	avg := sum / 3.0

	// Use metadata from middle score (median)
	medianScore := validScores[2]

	return &db.LLMScore{
		Model:    o.model,
		Score:    avg,
		Metadata: medianScore.Metadata,
	}, nil
}

func parseOpenAIResponse(body []byte) (string, error) {
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &openaiResp); err == nil {
		if len(openaiResp.Choices) == 0 {
			return "", errors.New("no choices in OpenAI response")
		}
		return openaiResp.Choices[0].Message.Content, nil
	}

	// If parsing as chat completion failed, try parsing as error response
	var openaiErr struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Param   string `json:"param"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err2 := json.Unmarshal(body, &openaiErr); err2 == nil && openaiErr.Error.Message != "" {
		return "", fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)", openaiErr.Error.Message, openaiErr.Error.Type, openaiErr.Error.Code)
	}

	// Log raw response for debugging
	log.Printf("Failed to parse OpenAI response as chat completion or error.\nRaw response:\n%s", string(body))
	return "", errors.New("invalid OpenAI API response format")
}

func parseBiasResult(contentResp string) BiasResult {
	var biasResult BiasResult
	if err := json.Unmarshal([]byte(contentResp), &biasResult); err != nil {
		log.Printf("Failed to parse LLM JSON response: %v", err)
		biasResult = BiasResult{}
	}

	// Validate category
	validCategory := false
	for _, cat := range BiasCfg.Categories {
		if biasResult.Category == cat {
			validCategory = true
			break
		}
	}
	if !validCategory {
		biasResult.Category = "unknown"
	}

	// Validate confidence
	if biasResult.Confidence < 0 || biasResult.Confidence > 1 {
		biasResult.Confidence = 0
	}

	return biasResult
}

func heuristicCategory(content string) string {
	contentLower := strings.ToLower(content)

	for cat, keywords := range BiasCfg.KeywordHeuristics {
		for _, kw := range keywords {
			if strings.Contains(contentLower, strings.ToLower(kw)) {
				return cat
			}
		}
	}

	return "unknown"
}

type LLMClient struct {
	client     *resty.Client
	cache      *Cache
	db         *sqlx.DB
	llmService LLMService
}

func NewLLMClient(dbConn *sqlx.DB) *LLMClient {
	client := resty.New()
	cache := NewCache()

	provider := os.Getenv("LLM_PROVIDER")

	var service LLMService

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

		service = NewOpenAILLMService(client, model, apiKey)
	default:
		log.Fatal("ERROR: LLM_PROVIDER not set or unknown, cannot initialize LLM service")
	}

	return &LLMClient{
		client:     client,
		cache:      cache,
		db:         dbConn,
		llmService: service,
	}
}

func (c *LLMClient) analyzeContent(articleID int64, content string, model string, _ string) (*db.LLMScore, error) {
	contentHash := hashContent(content)

	// Check cache
	if cached, ok := c.cache.Get(contentHash, model); ok {
		return cached, nil
	}

	var score *db.LLMScore
	var err error

	score, err = c.EnsembleAnalyze(content)
	if err != nil {
		return nil, err
	}

	score.ArticleID = articleID
	score.Model = model
	score.CreatedAt = time.Now()

	// Cache it
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
	models := []struct {
		name string
		url  string
	}{
		{"left", LeftModelURL},
		{"center", CenterModelURL},
		{"right", RightModelURL},
	}

	for _, m := range models {
		score, err := c.analyzeContent(article.ID, article.Content, m.name, m.url)
		if err != nil {
			log.Printf("Error analyzing article %d with model %s: %v", article.ID, m.name, err)

			continue
		}

		_, err = db.InsertLLMScore(c.db, score)
		if err != nil {
			log.Printf("Error inserting LLM score for article %d model %s: %v", article.ID, m.name, err)
		}
	}

	return nil
}

func (c *LLMClient) ReanalyzeArticle(articleID int64) error {
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}

	// Delete existing scores
	_, err = tx.Exec("DELETE FROM llm_scores WHERE article_id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}

		return err
	}

	var article db.Article

	err = tx.Get(&article, "SELECT * FROM articles WHERE id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}

		return err
	}

	models := []struct {
		name string
		url  string
	}{
		{"left", LeftModelURL},
		{"center", CenterModelURL},
		{"right", RightModelURL},
	}

	for _, m := range models {
		score, err := c.analyzeContent(article.ID, article.Content, m.name, m.url)
		if err != nil {
			log.Printf("Error reanalyzing article %d with model %s: %v", article.ID, m.name, err)

			continue
		}

		_, err = tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata) 
			VALUES (:article_id, :model, :score, :metadata)`, score)
		if err != nil {
			log.Printf("Error inserting reanalysis score for article %d model %s: %v", article.ID, m.name, err)
		}
	}

	return tx.Commit()
}

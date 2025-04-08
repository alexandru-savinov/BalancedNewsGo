package llm

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

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
	PromptTemplate, err = LoadPromptTemplate("configs/prompt_template.txt")
	if err != nil {
		log.Fatalf("Failed to load prompt template: %v", err)
	}
	BiasCfg, err = LoadBiasConfig("configs/bias_config.json")
	if err != nil {
		log.Fatalf("Failed to load bias config: %v", err)
	}
}

// LLM API endpoints
var (
	LeftModelURL   = "http://localhost:8001/analyze"
	CenterModelURL = "https://api.openai.com/v1/chat/completions"
	RightModelURL  = "http://localhost:8003/analyze"
)

// Cache key: hash(content) + model
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

// Hash function placeholder (can replace with real hash)
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

	url := "https://api.openai.com/v1/chat/completions"

	var resp *resty.Response
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
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

		resp, err = req.Post(url)

		if err == nil && resp.IsSuccess() {
			var openaiResp struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.Unmarshal(resp.Body(), &openaiResp); err != nil {
				return nil, err
			}
			if len(openaiResp.Choices) == 0 {
				return nil, errors.New("no choices in OpenAI response")
			}
			contentResp := openaiResp.Choices[0].Message.Content

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

			// Heuristic cross-check
			heuristicCategory := "unknown"
			contentLower := strings.ToLower(content)
			for cat, keywords := range BiasCfg.KeywordHeuristics {
				for _, kw := range keywords {
					if strings.Contains(contentLower, strings.ToLower(kw)) {
						heuristicCategory = cat
						break
					}
				}
				if heuristicCategory != "unknown" {
					break
				}
			}

			metadataMap := map[string]interface{}{
				"raw_response":      contentResp,
				"parsed_bias":       biasResult,
				"heuristic_category": heuristicCategory,
			}
			metadataBytes, _ := json.Marshal(metadataMap)

			return &db.LLMScore{
				Model:    o.model,
				Score:    biasResult.Confidence,
				Metadata: string(metadataBytes),
			}, nil
		}

		log.Printf("OpenAI API call failed (attempt %d): %v", attempt, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return nil, errors.New("OpenAI API call failed after retries")
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
			log.Println("Warning: OPENAI_API_KEY not set, falling back to mock LLM service")
			service = &MockLLMService{}
			break
		}
		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-3.5-turbo"
		}
		service = NewOpenAILLMService(client, model, apiKey)
	default:
		service = &MockLLMService{}
	}

	return &LLMClient{
		client:     client,
		cache:      cache,
		db:         dbConn,
		llmService: service,
	}
}

type apiResponse struct {
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}


func (c *LLMClient) analyzeContent(articleID int64, content string, model string, url string) (*db.LLMScore, error) {
	contentHash := hashContent(content)

	// Check cache
	if cached, ok := c.cache.Get(contentHash, model); ok {
		return cached, nil
	}

	score, err := c.llmService.Analyze(content)
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
		tx.Rollback()
		return err
	}

	var article db.Article
	err = tx.Get(&article, "SELECT * FROM articles WHERE id = ?", articleID)
	if err != nil {
		tx.Rollback()
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

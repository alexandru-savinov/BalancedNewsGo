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

// LLM API endpoints
var (
	LeftModelURL   = "http://localhost:8001/analyze"
	CenterModelURL = "http://localhost:8002/analyze"
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

type LLMClient struct {
	client *resty.Client
	cache  *Cache
	db     *sqlx.DB
}

func NewLLMClient(dbConn *sqlx.DB) *LLMClient {
	return &LLMClient{
		client: resty.New(),
		cache:  NewCache(),
		db:     dbConn,
	}
}

type apiResponse struct {
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (c *LLMClient) callAPI(url string, content string) (*apiResponse, error) {
	var resp *resty.Response
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(map[string]string{"text": content}).
			Post(url)

		if err == nil && resp.IsSuccess() {
			var apiResp apiResponse
			if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
				return nil, err
			}
			return &apiResp, nil
		}

		log.Printf("API call to %s failed (attempt %d): %v", url, attempt, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return nil, errors.New("API call failed after retries")
}

func (c *LLMClient) analyzeContent(articleID int64, content string, model string, url string) (*db.LLMScore, error) {
	contentHash := hashContent(content)

	// Check cache
	if cached, ok := c.cache.Get(contentHash, model); ok {
		return cached, nil
	}

	apiResp, err := c.callAPI(url, content)
	if err != nil {
		return nil, err
	}

	metaBytes, _ := json.Marshal(apiResp.Metadata)

	score := &db.LLMScore{
		ArticleID: articleID,
		Model:     model,
		Score:     apiResp.Score,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}

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

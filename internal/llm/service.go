package llm

import (
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// PromptVariant represents a specific prompt template
type PromptVariant struct {
	ID       string
	Template string
	Examples []string
}

// DefaultPromptVariant is the standard prompt used for scoring
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

// LLMService defines the interface for LLM providers
type LLMService interface {
	ScoreContent(content string, model string) (float64, string, float64, error)
	callLLMAPIWithKey(model string, prompt string) (*resty.Response, error)
}

// HTTPLLMService implements the LLM service interface using HTTP APIs
type HTTPLLMService struct {
	client   *resty.Client
	provider string
	endpoint string
	cache    *Cache
}

// NewHTTPLLMService creates a new HTTP-based LLM service
func NewHTTPLLMService(client *resty.Client) *HTTPLLMService {
	provider := getenv("LLM_PROVIDER", "openai")
	endpoint := getenv("LLM_API_ENDPOINT", "https://api.openai.com/v1/chat/completions")

	client.SetTimeout(defaultLLMTimeout)

	return &HTTPLLMService{
		client:   client,
		provider: provider,
		endpoint: endpoint,
	}
}

// Cache represents an in-memory cache for LLM responses
type Cache struct {
	data map[string]map[string]*CacheEntry
}

type CacheEntry struct {
	Score     *LLMScore
	Timestamp time.Time
}

// NewCache creates a new LLM response cache
func NewCache() *Cache {
	return &Cache{
		data: make(map[string]map[string]*CacheEntry),
	}
}

// Get retrieves a cached score
func (c *Cache) Get(contentHash string, model string) (*LLMScore, bool) {
	if modelCache, ok := c.data[contentHash]; ok {
		if entry, ok := modelCache[model]; ok {
			if time.Since(entry.Timestamp) < 24*time.Hour {
				return entry.Score, true
			}
		}
	}
	return nil, false
}

// Set stores a score in the cache
func (c *Cache) Set(contentHash string, model string, score *LLMScore) {
	if _, ok := c.data[contentHash]; !ok {
		c.data[contentHash] = make(map[string]*CacheEntry)
	}
	c.data[contentHash][model] = &CacheEntry{
		Score:     score,
		Timestamp: time.Now(),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

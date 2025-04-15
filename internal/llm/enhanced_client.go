package llm

import (
	"fmt"
	"log"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
)

// EnhancedLLMClient is an improved version of LLMClient with better error handling
type EnhancedLLMClient struct {
	dbConn      *sqlx.DB
	httpService *EnhancedHTTPLLMService
	cache       *Cache
}

// NewEnhancedLLMClient creates a new LLM client with enhanced error handling
func NewEnhancedLLMClient(dbConn *sqlx.DB) *EnhancedLLMClient {
	client := resty.New().
		SetTimeout(60 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	return &EnhancedLLMClient{
		dbConn:      dbConn,
		httpService: NewEnhancedHTTPLLMService(client),
		cache:       NewCache(),
	}
}

// ScoreWithModel scores an article with a specific model
func (c *EnhancedLLMClient) ScoreWithModel(articleID int64, content, modelName string) (*db.LLMScore, error) {
	// Start performance tracking
	start := time.Now()
	defer func() {
		api.LogPerformance(fmt.Sprintf("ScoreWithModel-%s", modelName), start)
	}()

	// Check cache first
	contentHash := hashContent(content)
	if cachedScore, found := c.cache.Get(contentHash, modelName); found {
		log.Printf("[ScoreWithModel] Cache hit for article %d with model %s", articleID, modelName)
		return cachedScore, nil
	}

	// Not in cache, call the LLM API
	score, err := c.httpService.AnalyzeWithModel(modelName, content)
	if err != nil {
		// Check if it's a provider error
		if providerErr, ok := err.(*ProviderError); ok {
			// Log detailed provider error
			log.Printf("[ScoreWithModel] Provider error for article %d with model %s: %v",
				articleID, modelName, providerErr)

			// Handle rate limiting specially
			if providerErr.IsRateLimitError() {
				resetTime, hasReset := providerErr.GetRateLimitReset()
				if hasReset {
					log.Printf("[ScoreWithModel] Rate limit will reset at: %s", resetTime.Format(time.RFC3339))
				}
			}
		} else {
			// Log generic error
			log.Printf("[ScoreWithModel] Error for article %d with model %s: %v",
				articleID, modelName, err)
		}
		return nil, err
	}

	// Store in cache
	c.cache.Set(contentHash, modelName, score)

	// Store in database
	score.ArticleID = articleID
	err = db.StoreLLMScore(c.dbConn, score)
	if err != nil {
		log.Printf("[ScoreWithModel] Failed to store score in database: %v", err)
		// Continue anyway, as we have the score in memory
	}

	return score, nil
}

// EnhancedEnsembleAnalyze performs ensemble analysis with enhanced error handling
func (c *EnhancedLLMClient) EnhancedEnsembleAnalyze(articleID int64, content string) (float64, error) {
	// Load the composite score configuration
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return 0, fmt.Errorf("failed to load composite score config: %w", err)
	}

	// Track warnings for partial success
	var warnings []api.WarningInfo

	// Track scores for each model
	var scores []db.LLMScore

	// Track if all models failed with rate limiting
	allRateLimited := true

	// Process each model in the configuration
	for _, modelCfg := range cfg.Models {
		score, err := c.ScoreWithModel(articleID, content, modelCfg.ModelName)
		if err != nil {
			// Check if it's a rate limit error
			if providerErr, ok := err.(*ProviderError); ok && providerErr.IsRateLimitError() {
				warnings = append(warnings, api.WarningInfo{
					Code:    api.WarnRateLimit,
					Message: fmt.Sprintf("Model %s is rate limited", modelCfg.ModelName),
				})
				continue
			} else {
				// At least one model failed with a non-rate-limit error
				allRateLimited = false

				warnings = append(warnings, api.WarningInfo{
					Code:    api.WarnModelError,
					Message: fmt.Sprintf("Model %s failed: %v", modelCfg.ModelName, err),
				})
				continue
			}
		}

		// If we got here, at least one model succeeded
		allRateLimited = false
		scores = append(scores, *score)
	}

	// If all models were rate limited, return a specific error
	if len(scores) == 0 && allRateLimited {
		return 0, &ProviderError{
			StatusCode: api.StatusTooManyRequests,
			Message:    "All LLM models are rate limited",
			Provider:   "ensemble",
			Type:       "rate_limit_error",
		}
	}

	// If we have no scores at all, return an error
	if len(scores) == 0 {
		return 0, fmt.Errorf("all models failed to score the article")
	}

	// Compute the composite score
	compositeScore, confidence, err := ComputeCompositeScoreWithConfidence(scores)
	if err != nil {
		return 0, fmt.Errorf("failed to compute composite score: %w", err)
	}

	// Store the ensemble score in the database
	ensembleScore := &db.EnsembleScore{
		ArticleID:  articleID,
		Score:      compositeScore,
		Confidence: confidence,
		ModelCount: len(scores),
	}

	err = db.StoreEnsembleScore(c.dbConn, ensembleScore)
	if err != nil {
		log.Printf("[EnhancedEnsembleAnalyze] Failed to store ensemble score: %v", err)
		// Continue anyway, as we have the score in memory
	}

	// Log warnings if any
	if len(warnings) > 0 {
		log.Printf("[EnhancedEnsembleAnalyze] Article %d scored with %d/%d models. Warnings: %d",
			articleID, len(scores), len(cfg.Models), len(warnings))
	} else {
		log.Printf("[EnhancedEnsembleAnalyze] Article %d scored successfully with all %d models",
			articleID, len(scores))
	}

	return compositeScore, nil
}

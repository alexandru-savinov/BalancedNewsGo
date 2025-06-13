package client

import (
	"context"
	"sync"
	"time"

	rawclient "github.com/alexandru-savinov/BalancedNewsGo/internal/api/client"
)

// GetArticles retrieves articles with caching
func (c *APIClient) GetArticles(ctx context.Context, params ArticlesParams) ([]Article, error) {
	// Build cache key from parameters
	cacheKey := buildCacheKey("articles", params.Source, params.Leaning, params.Limit, params.Offset)

	// Check cache first
	if cached, found := c.getCached(cacheKey); found {
		if articles, ok := cached.([]Article); ok {
			return articles, nil
		}
	}

	// Cache miss - call API with retry logic
	var articles []Article
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		rawParams := rawclient.ArticlesParams{
			Source:  params.Source,
			Leaning: params.Leaning,
			Limit:   params.Limit,
			Offset:  params.Offset,
		}

		rawArticles, err := c.raw.ArticlesAPI.GetArticles(ctx, rawParams)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		// Convert to our model
		articles = convertArticles(rawArticles)
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Cache successful response
	c.setCached(cacheKey, articles)
	return articles, nil
}

// GetArticle retrieves a single article with caching
func (c *APIClient) GetArticle(ctx context.Context, id int64) (*Article, error) {
	cacheKey := buildCacheKey("article", id)

	// Check cache first
	if cached, found := c.getCached(cacheKey); found {
		if article, ok := cached.(*Article); ok {
			return article, nil
		}
	}

	// Cache miss - call API with retry logic
	var article *Article
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		rawArticle, err := c.raw.ArticlesAPI.GetArticle(ctx, id)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		// Convert to our model
		article = convertArticle(rawArticle)
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Cache successful response
	c.setCached(cacheKey, article)
	return article, nil
}

// GetArticleSummary retrieves article summary with caching
func (c *APIClient) GetArticleSummary(ctx context.Context, id int64) (string, error) {
	cacheKey := buildCacheKey("summary", id)

	// Check cache first
	if cached, found := c.getCached(cacheKey); found {
		if summary, ok := cached.(string); ok {
			return summary, nil
		}
	}

	// Cache miss - call API with retry logic
	var summary string
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		result, err := c.raw.ArticlesAPI.GetArticleSummary(ctx, id)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		summary = result
		break
	}

	if lastErr != nil {
		return "", lastErr
	}

	// Cache successful response
	c.setCached(cacheKey, summary)
	return summary, nil
}

// GetArticleBias retrieves article bias analysis with caching
func (c *APIClient) GetArticleBias(ctx context.Context, id int64) (*ScoreResponse, error) {
	cacheKey := buildCacheKey("bias", id)

	// Check cache first
	if cached, found := c.getCached(cacheKey); found {
		if bias, ok := cached.(*ScoreResponse); ok {
			return bias, nil
		}
	}

	// Cache miss - call API with retry logic
	var bias *ScoreResponse
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		rawBias, err := c.raw.ArticlesAPI.GetArticleBias(ctx, id)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		// Convert to our model
		bias = convertScoreResponse(rawBias)
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Cache successful response
	c.setCached(cacheKey, bias)
	return bias, nil
}

// GetArticleEnsemble retrieves ensemble details with caching
func (c *APIClient) GetArticleEnsemble(ctx context.Context, id int64) (interface{}, error) {
	cacheKey := buildCacheKey("ensemble", id)

	// Check cache first
	if cached, found := c.getCached(cacheKey); found {
		return cached, nil
	}

	// Cache miss - call API with retry logic
	var ensemble interface{}
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		result, err := c.raw.ArticlesAPI.GetArticleEnsemble(ctx, id)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		ensemble = result
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Cache successful response
	c.setCached(cacheKey, ensemble)
	return ensemble, nil
}

// CreateArticle creates a new article (no caching for writes)
func (c *APIClient) CreateArticle(ctx context.Context, req CreateArticleRequest) (*CreateArticleResponse, error) {
	rawReq := rawclient.CreateArticleRequest{
		Source:  req.Source,
		PubDate: req.PubDate,
		URL:     req.URL,
		Title:   req.Title,
		Content: req.Content,
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		rawResp, err := c.raw.ArticlesAPI.CreateArticle(ctx, rawReq)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		// Convert response
		resp := &CreateArticleResponse{
			ArticleID: rawResp.ArticleID,
			Status:    rawResp.Status,
		}
		return resp, nil
	}

	return nil, lastErr
}

// ReanalyzeArticle triggers article reanalysis (no caching for operations)
func (c *APIClient) ReanalyzeArticle(ctx context.Context, id int64, req *ManualScoreRequest) (string, error) {
	var rawReq *rawclient.ManualScoreRequest
	if req != nil {
		rawReq = &rawclient.ManualScoreRequest{
			Score:    req.Score,
			Source:   req.Source,
			Comments: req.Comments,
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		result, err := c.raw.LLMApi.ReanalyzeArticle(ctx, id, rawReq)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		// Invalidate related caches
		c.invalidateArticleCache(id)
		return result, nil
	}

	return "", lastErr
}

// GetFeedHealth retrieves feed health status with caching
func (c *APIClient) GetFeedHealth(ctx context.Context) (FeedHealth, error) {
	cacheKey := "feed_health"

	// Check cache first
	if cached, found := c.getCached(cacheKey); found {
		if health, ok := cached.(FeedHealth); ok {
			return health, nil
		}
	}

	// Cache miss - call API with retry logic
	var health FeedHealth
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		rawHealth, err := c.raw.FeedsApi.GetFeedHealth(ctx)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		health = FeedHealth(rawHealth)
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Cache successful response with shorter TTL for health checks
	c.cache.Store(cacheKey, &cacheEntry{
		value:     health,
		expiresAt: time.Now().Add(10 * time.Second), // Shorter TTL for health
	})
	return health, nil
}

// TriggerRefresh triggers feed refresh (no caching for operations)
func (c *APIClient) TriggerRefresh(ctx context.Context) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.RetryDelay * time.Duration(attempt))
		}

		result, err := c.raw.FeedsApi.TriggerRefresh(ctx)
		if err != nil {
			lastErr = c.translateError(err)
			continue
		}

		return result, nil
	}

	return "", lastErr
}

// invalidateArticleCache removes cached entries related to a specific article
func (c *APIClient) invalidateArticleCache(articleID int64) {
	// Remove specific article caches
	c.cache.Delete(buildCacheKey("article", articleID))
	c.cache.Delete(buildCacheKey("summary", articleID))
	c.cache.Delete(buildCacheKey("bias", articleID))
	c.cache.Delete(buildCacheKey("ensemble", articleID))

	// Remove article list caches (they might include this article)
	c.cache.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if keyStr[:8] == "articles" {
			c.cache.Delete(key)
		}
		return true
	})
}

// GetCacheStats returns cache statistics
func (c *APIClient) GetCacheStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_entries":   0,
		"expired_entries": 0,
	}

	totalEntries := 0
	expiredEntries := 0

	c.cache.Range(func(key, value interface{}) bool {
		totalEntries++
		if entry, ok := value.(*cacheEntry); ok && entry.isExpired() {
			expiredEntries++
		}
		return true
	})

	stats["total_entries"] = totalEntries
	stats["expired_entries"] = expiredEntries
	return stats
}

// ClearCache removes all cached entries
func (c *APIClient) ClearCache() {
	c.cache = &sync.Map{}
}

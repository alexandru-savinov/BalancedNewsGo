# NewsBalancer Political Analysis Development Roadmap

## Executive Summary

This development roadmap provides a comprehensive guide for implementing LLM-based political analysis in the NewsBalancer application. The implementation will enable the app to analyze news articles from multiple political perspectives (Left, Center, Right) using a single LLM with different prompts, creating a politically balanced news feed for users.

Based on our analysis of the current codebase and your requirements, we've created a structured approach that prioritizes backend implementation first, focuses on a simple Left/Center/Right political spectrum with -1 to 1 scoring, and aims to deliver an MVP quickly with at least one Moldavian RSS feed.

## Project Overview

### Current State
The NewsBalancer application currently has:
- A Gin backend with SQLite database accessed via sqlx
- RSS feed fetching using gofeed, scheduled with robfig/cron
- LLM microservices for political analysis (Left, Center, Right) accessed via HTTP
- API endpoints for articles and scores
- No frontend UI at this stage

### Target State
After implementation, NewsBalancer will have:
- Political analysis from three perspectives (Left, Center, Right)
- Extended database schema to store multiple political scores
- Enhanced API endpoints for retrieving politically balanced content
- Basic visualization of political scores in the UI
- At least one Moldavian RSS feed integration

## Implementation Strategy

The implementation follows a three-phase approach:

### Phase 1: Core Backend Implementation (Days 1-10)
Focus on building the fundamental political analysis capabilities:
- Create prompt templates for Left, Center, and Right perspectives
- Extend the analysis service to support multiple political analyses
- Update database schema to store multiple scores
- Integrate language detection
- Update RSS processing pipeline

### Phase 2: API and Frontend Integration (Days 11-20)
Connect the political analysis to the user interface:
- Improve API fallback: when no valid LLM scores are available, the API now returns `"composite_score": null` and includes `"status": "scoring_unavailable"` instead of zeros, to clearly indicate unavailable scoring data.
- **Frontend now detects these cases and displays "Scoring unavailable" or "No data" instead of zeros, with UI elements adjusted accordingly.**
- Extend API endpoints to support political filtering and sorting
- Update templates to display political scores
- Implement basic filtering controls
- Create simple score visualization
- Perform integration testing

### Phase 3: Testing, Refinement, and Deployment (Days 21-30)
Ensure quality and prepare for production:
- Conduct user acceptance testing
- Refine prompts based on testing results
- Optimize performance and resource usage
- Deploy to production
- Set up monitoring

## Key Components

### 1. Political Analysis Service
The core component that analyzes news articles from multiple political perspectives:
```go
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
```

### 2. Database Schema Extensions
New and modified database models to support political analysis:
```go
type Article struct {
	ID        int64     `db:"id"`
	Source    string    `db:"source"`
	PubDate   time.Time `db:"pub_date"`
	URL       string    `db:"url"`
	Title     string    `db:"title"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
}

type LLMScore struct {
	ID        int64     `db:"id"`
	ArticleID int64     `db:"article_id"`
	Model     string    `db:"model"` // "left", "center", "right"
	Score     float64   `db:"score"`
	Metadata  string    `db:"metadata"`
	CreatedAt time.Time `db:"created_at"`
}
```

### 3. API Endpoints
Extended API to support political filtering and sorting:
```go
router.GET("/api/articles", func(c *gin.Context) {
	source := c.Query("source")
	leaning := c.Query("leaning")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
		return
	}
	c.JSON(http.StatusOK, articles)
})
```

### 4. RSS Feed Processing
Updated RSS processing to include political analysis:
```go
func (c *Collector) FetchAndStore() {
	parser := gofeed.NewParser()

	for _, feedURL := range c.FeedURLs {
		feed, err := parser.ParseURL(feedURL)
		if err != nil {
			log.Printf("[RSS] Failed to parse feed %s: %v", feedURL, err)
			continue
		}

		for _, item := range feed.Items {
			if !isValidItem(item) {
				continue
			}

			exists, err := db.ArticleExistsByURL(c.DB, item.Link)
			if err != nil || exists {
				continue
			}

			pubTime := time.Now()
			if item.PublishedParsed != nil {
				pubTime = *item.PublishedParsed
			}

			article := &db.Article{
				Source:  feed.Title,
				PubDate: pubTime,
				URL:     item.Link,
				Title:   item.Title,
				Content: item.Content,
			}

			_, err = db.InsertArticle(c.DB, article)
			if err != nil {
				log.Printf("[RSS] Failed to insert article: %v", err)
			}
		}
	}
}
```

## Technical Requirements

### Software Dependencies
- **Core**: Gin, sqlx, resty, gofeed, robfig/cron

### Environment Variables
- DATABASE_URL
- RSS_FEED_URLS
- LEFT_LLM_URL
- CENTER_LLM_URL
- RIGHT_LLM_URL

## Implementation Timeline

### Sprint 1: Core Backend Implementation (Days 1-10)
- **Milestone 1.1**: Political Analysis Service (Days 1-3)
- **Milestone 1.2**: Database Schema Updates (Days 4-6)
- **Milestone 1.3**: RSS Feed Integration (Days 7-10)

### Sprint 2: API and Frontend Integration (Days 11-20)
- **Milestone 2.1**: API Endpoint Extensions (Days 11-14)
- **Milestone 2.2**: Basic Frontend Updates (Days 15-18)
- **Milestone 2.3**: Integration Testing (Days 19-20)

### Sprint 3: Testing, Refinement, and Deployment (Days 21-30)
- **Milestone 3.1**: User Acceptance Testing (Days 21-23)
- **Milestone 3.2**: Refinement and Optimization (Days 24-27)
- **Milestone 3.3**: Deployment and Monitoring (Days 28-30)

## Key Challenges and Mitigations

### 1. Increased API Costs
**Challenge**: Each article requires three separate LLM analyses.
**Mitigation**: Use OpenRouter's free tier, implement caching, batch processing.

### 2. Prompt Engineering Complexity
**Challenge**: Creating effective prompts for different political perspectives.
**Mitigation**: Start with simple prompts, test with known biased articles, iteratively refine.

### 3. Performance Impact
**Challenge**: Tripling LLM analysis calls could impact performance.
**Mitigation**: Implement async processing, background job queue, batch processing.

## Future Enhancements

After the MVP implementation, consider these enhancements:

### Phase 1: Refinement (Weeks 7-8)
- Refine prompts based on user feedback
- Optimize performance and resource usage

### Phase 2: Enhanced Features (Weeks 9-12)
- Add confidence metrics and justifications for scores
- Enhance UI with advanced visualizations

### Phase 3: Multiple LLM Integration (Weeks 13-16)
- Implement different LLM models for different perspectives
- Compare performance across models

## Implementation Files

This roadmap is accompanied by the following detailed documents:

1. **Implementation Gaps Analysis** - Detailed analysis of current implementation and required changes
2. **Implementation Plan** - Comprehensive plan with code examples for each component
3. **Technical Requirements** - Detailed technical requirements and resource needs
4. **Timeline and Milestones** - Detailed timeline with specific deliverables and evaluation criteria

## Conclusion

This development roadmap provides a clear path to implementing LLM-based political analysis in the NewsBalancer application. By following this structured approach, you can efficiently deliver a politically balanced news feed that analyzes content from multiple perspectives.

The implementation prioritizes backend functionality first, focuses on a simple political spectrum, and aims for a quick MVP delivery. The modular design allows for future enhancements, including more sophisticated political analysis, improved UI, and integration of multiple LLM models.

With this roadmap, you have a comprehensive guide to transform NewsBalancer into a tool that helps users access news from balanced political perspectives.

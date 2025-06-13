package api

import (
	"context"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/jmoiron/sqlx"
)

// InternalAPIClient provides API functionality without HTTP overhead
// This allows template handlers to use the same business logic as HTTP API handlers
// while maintaining the API-first architecture
type InternalAPIClient struct {
	dbConn *sqlx.DB
}

// NewInternalAPIClient creates a new internal API client
func NewInternalAPIClient(dbConn *sqlx.DB) *InternalAPIClient {
	return &InternalAPIClient{
		dbConn: dbConn,
	}
}

// ArticlesParams represents parameters for article queries
type InternalArticlesParams struct {
	Source  string
	Leaning string
	Limit   int
	Offset  int
}

// InternalArticle represents an article in the internal API
type InternalArticle struct {
	ID             int64   `json:"id"`
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	URL            string  `json:"url"`
	Source         string  `json:"source"`
	PubDate        string  `json:"pub_date"`
	CompositeScore float64 `json:"composite_score"`
	Confidence     float64 `json:"confidence"`
	ScoreSource    string  `json:"score_source"`
	Bias           string  `json:"bias"`
	Summary        string  `json:"summary"`
}

// GetArticles fetches articles using the same logic as the HTTP API handler
func (c *InternalAPIClient) GetArticles(ctx context.Context, params InternalArticlesParams) ([]InternalArticle, error) {
	// Use the same logic as getArticlesHandler but return data directly
	source := params.Source
	if source == "all" || source == "" {
		source = ""
	}

	leaning := params.Leaning
	if leaning == "all" || leaning == "" {
		leaning = ""
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	offset := params.Offset
	if offset < 0 {
		offset = 0
	} // Fetch articles from database using the same method as the HTTP handler
	dbArticles, err := db.FetchArticles(c.dbConn, source, leaning, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert to internal format
	articles := make([]InternalArticle, len(dbArticles))
	for i, dbArticle := range dbArticles {
		// Handle nullable CompositeScore and Confidence fields
		var compositeScore float64
		var confidence float64
		var scoreSource string

		if dbArticle.CompositeScore != nil {
			compositeScore = *dbArticle.CompositeScore
		}

		if dbArticle.Confidence != nil {
			confidence = *dbArticle.Confidence
		}

		if dbArticle.ScoreSource != nil {
			scoreSource = *dbArticle.ScoreSource
		}

		articles[i] = InternalArticle{
			ID:             dbArticle.ID,
			Title:          dbArticle.Title,
			Content:        dbArticle.Content,
			URL:            dbArticle.URL,
			Source:         dbArticle.Source,
			PubDate:        dbArticle.PubDate.Format("2006-01-02 15:04:05"),
			CompositeScore: compositeScore,
			Confidence:     confidence,
			ScoreSource:    scoreSource,
			Summary:        "", // No summary field in the database model
		}
		// Determine bias label based on composite score
		if dbArticle.CompositeScore != nil {
			if *dbArticle.CompositeScore < -0.1 {
				articles[i].Bias = "left"
			} else if *dbArticle.CompositeScore > 0.1 {
				articles[i].Bias = "right"
			} else {
				articles[i].Bias = "center"
			}
		} else {
			articles[i].Bias = "unknown"
		}
	}

	return articles, nil
}

// GetArticle fetches a single article by ID
func (c *InternalAPIClient) GetArticle(ctx context.Context, id int64) (*InternalArticle, error) {
	dbArticle, err := db.FetchArticleByID(c.dbConn, id)
	if err != nil {
		return nil, err
	}

	// Handle nullable CompositeScore and Confidence fields
	var compositeScore float64
	var confidence float64
	var scoreSource string

	if dbArticle.CompositeScore != nil {
		compositeScore = *dbArticle.CompositeScore
	}

	if dbArticle.Confidence != nil {
		confidence = *dbArticle.Confidence
	}

	if dbArticle.ScoreSource != nil {
		scoreSource = *dbArticle.ScoreSource
	}

	article := &InternalArticle{
		ID:             dbArticle.ID,
		Title:          dbArticle.Title,
		Content:        dbArticle.Content,
		URL:            dbArticle.URL,
		Source:         dbArticle.Source,
		PubDate:        dbArticle.PubDate.Format("2006-01-02 15:04:05"),
		CompositeScore: compositeScore,
		Confidence:     confidence,
		ScoreSource:    scoreSource,
		Summary:        "", // No summary field in the database model
	}
	// Determine bias label
	if dbArticle.CompositeScore != nil {
		if *dbArticle.CompositeScore < -0.1 {
			article.Bias = "left"
		} else if *dbArticle.CompositeScore > 0.1 {
			article.Bias = "right"
		} else {
			article.Bias = "center"
		}
	} else {
		article.Bias = "unknown"
	}

	return article, nil
}

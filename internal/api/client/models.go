package client

import "time"

// Article represents an article
type Article struct {
	ArticleID      int64     `json:"article_id,omitempty"`
	Title          string    `json:"Title,omitempty"`
	Content        string    `json:"Content,omitempty"`
	URL            string    `json:"URL,omitempty"`
	Source         string    `json:"Source,omitempty"`
	PubDate        time.Time `json:"PubDate,omitempty"`
	CreatedAt      time.Time `json:"CreatedAt,omitempty"`
	CompositeScore float64   `json:"CompositeScore,omitempty"`
	Confidence     float64   `json:"Confidence,omitempty"`
	ScoreSource    string    `json:"ScoreSource,omitempty"`
	BiasLabel      string    `json:"BiasLabel,omitempty"`
	AnalysisNotes  string    `json:"AnalysisNotes,omitempty"`
}

// CreateArticleRequest represents the request to create an article
type CreateArticleRequest struct {
	Source  string `json:"source"`
	PubDate string `json:"pub_date"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// CreateArticleResponse represents the response after creating an article
type CreateArticleResponse struct {
	ArticleID int64  `json:"article_id,omitempty"`
	Status    string `json:"status,omitempty"`
}

// ArticlesParams represents parameters for fetching articles
type ArticlesParams struct {
	Source  string `json:"source,omitempty"`
	Leaning string `json:"leaning,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

// ScoreResponse represents a bias score response
type ScoreResponse struct {
	ArticleID int64                   `json:"article_id,omitempty"`
	Score     float64                 `json:"score,omitempty"`
	Details   []IndividualScoreResult `json:"details,omitempty"`
}

// IndividualScoreResult represents individual scoring details
type IndividualScoreResult struct {
	Model      string  `json:"model,omitempty"`
	Score      float64 `json:"score,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Reasoning  string  `json:"reasoning,omitempty"`
}

// FeedbackRequest represents user feedback
type FeedbackRequest struct {
	ArticleID    int64   `json:"article_id"`
	UserFeedback string  `json:"user_feedback"`
	UserScore    float64 `json:"user_score,omitempty"`
	Comments     string  `json:"comments,omitempty"`
}

// ManualScoreRequest represents a manual score request
type ManualScoreRequest struct {
	Score    float64 `json:"score"`
	Source   string  `json:"source,omitempty"`
	Comments string  `json:"comments,omitempty"`
}

// ProgressState represents the state of a long-running operation
type ProgressState struct {
	Status       string  `json:"status,omitempty"`
	Step         string  `json:"step,omitempty"`
	Percent      int     `json:"percent,omitempty"`
	Message      string  `json:"message,omitempty"`
	Error        string  `json:"error,omitempty"`
	ErrorDetails string  `json:"error_details,omitempty"`
	FinalScore   float64 `json:"final_score,omitempty"`
	LastUpdated  int64   `json:"last_updated,omitempty"`
}

// FeedHealth represents feed health status
type FeedHealth map[string]bool

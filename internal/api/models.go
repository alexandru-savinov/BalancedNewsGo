package api

import (
	"time"
)

// Article represents a news article with bias analysis
// @Description A news article with bias analysis information
type Article struct {
	ID             int64     `json:"id" example:"42"`                           // Unique identifier
	Source         string    `json:"source" example:"CNN"`                      // News source name
	URL            string    `json:"url" example:"https://example.com/article"` // URL to the original article
	Title          string    `json:"title" example:"Breaking News"`             // Article title
	Content        string    `json:"content" example:"Article content..."`      // Article content
	PubDate        time.Time `json:"pub_date" example:"2023-01-01T12:00:00Z"`   // Article publication date
	CreatedAt      time.Time `json:"created_at" example:"2023-01-02T00:00:00Z"` // When added to the system
	CompositeScore *float64  `json:"composite_score,omitempty" example:"0.25"`  // Political bias score (-1 to 1)
	Confidence     *float64  `json:"confidence,omitempty" example:"0.85"`       // Confidence in the score (0 to 1)
	ScoreSource    *string   `json:"score_source,omitempty" example:"llm"`      // Source of the score
}

// CreateArticleRequest represents the request payload for creating an article
// @Description Request body for creating a new article
type CreateArticleRequest struct {
	Source  string `json:"source" example:"CNN" binding:"required"`                      // News source name
	PubDate string `json:"pub_date" example:"2023-01-01T12:00:00Z" binding:"required"`   // Publication date in RFC3339 format
	URL     string `json:"url" example:"https://example.com/article" binding:"required"` // Article URL
	Title   string `json:"title" example:"Breaking News" binding:"required"`             // Article title
	Content string `json:"content" example:"Article content..." binding:"required"`      // Article content
}

// CreateArticleResponse represents the response for creating an article
// @Description Response from creating a new article
type CreateArticleResponse struct {
	Status    string `json:"status" example:"created"` // Status of the operation
	ArticleID int64  `json:"article_id" example:"42"`  // ID of the created article
}

// ScoreResponse represents the bias analysis result
// @Description Political bias score analysis result
type ScoreResponse struct {
	CompositeScore *float64                `json:"composite_score,omitempty" example:"0.25"`       // Overall bias score
	Results        []IndividualScoreResult `json:"results"`                                        // Individual model scores
	Status         string                  `json:"status,omitempty" example:"scoring_unavailable"` // Status message if applicable
}

// IndividualScoreResult represents an individual model's bias score
// @Description Individual model scoring result
type IndividualScoreResult struct {
	Model       string    `json:"model" example:"claude-3"`        // Model name
	Score       float64   `json:"score" example:"0.3"`             // Bias score
	Confidence  float64   `json:"confidence" example:"0.8"`        // Model confidence
	Explanation string    `json:"explanation" example:"Reasoning"` // Explanation for the score
	CreatedAt   time.Time `json:"created_at"`                      // When the score was generated
}

// FeedbackRequest represents the request payload for submitting feedback
// @Description Request body for submitting user feedback
type FeedbackRequest struct {
	ArticleID        int64  `json:"article_id" binding:"required"`    // Article ID
	UserID           string `json:"user_id,omitempty"`                // User ID (optional in single-user mode)
	FeedbackText     string `json:"feedback_text" binding:"required"` // Feedback content
	Category         string `json:"category" example:"agree"`         // Feedback category: agree, disagree, unclear, other
	EnsembleOutputID *int64 `json:"ensemble_output_id,omitempty"`     // ID of specific ensemble output
	Source           string `json:"source" example:"web"`             // Source of the feedback
}

// ErrorResponse represents an API error response
// @Description Standard API error response
type ErrorResponse struct {
	Success bool        `json:"success" example:"false"` // Always false for errors
	Error   ErrorDetail `json:"error"`                   // Error details
}

// ErrorDetail contains details about an error
// @Description Detailed error information
type ErrorDetail struct {
	Code    string `json:"code" example:"validation_error"`            // Error code
	Message string `json:"message" example:"Invalid input parameters"` // Human-readable error message
}

// StandardResponse represents a standard API success response
// @Description Standard API success response
type StandardResponse struct {
	Success bool        `json:"success" example:"true"` // Always true for success
	Data    interface{} `json:"data"`                   // Response data payload
}

// ManualScoreRequest represents a request to manually set an article score
// @Description Request body for manually setting an article's bias score
type ManualScoreRequest struct {
	Score float64 `json:"score" example:"0.5" binding:"required"` // Score value between -1.0 and 1.0
}

type ArticleResponse struct {
	ArticleID      int64    `json:"article_id"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	URL            string   `json:"url"`
	Source         string   `json:"source"`
	CompositeScore *float64 `json:"composite_score"`
	Confidence     *float64 `json:"confidence"`
	PubDate        string   `json:"pub_date"`
	CreatedAt      string   `json:"created_at"`
}

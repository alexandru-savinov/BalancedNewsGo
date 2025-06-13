package client

import (
	"fmt"
	"net/http"
	"strconv"
	"strings" // Ensure strings is imported
	"time"

	rawclient "github.com/alexandru-savinov/BalancedNewsGo/internal/api/client"
)

// Model definitions for the wrapper
type Article struct {
	ArticleID      int64     `json:"article_id,omitempty"`
	Title          string    `json:"title,omitempty"`
	Content        string    `json:"content,omitempty"`
	URL            string    `json:"url,omitempty"`
	Source         string    `json:"source,omitempty"`
	PubDate        time.Time `json:"pub_date,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	CompositeScore float64   `json:"composite_score,omitempty"`
	Confidence     float64   `json:"confidence,omitempty"`
	ScoreSource    string    `json:"score_source,omitempty"`
	BiasLabel      string    `json:"bias_label,omitempty"`
	AnalysisNotes  string    `json:"analysis_notes,omitempty"`
}

type ArticlesParams struct {
	Source  string `json:"source,omitempty"`
	Leaning string `json:"leaning,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

type CreateArticleRequest struct {
	Source  string `json:"source"`
	PubDate string `json:"pub_date"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type CreateArticleResponse struct {
	ArticleID int64  `json:"article_id,omitempty"`
	Status    string `json:"status,omitempty"`
}

type ScoreResponse struct {
	ArticleID int64                   `json:"article_id,omitempty"`
	Score     float64                 `json:"score,omitempty"`
	Details   []IndividualScoreResult `json:"details,omitempty"`
}

type IndividualScoreResult struct {
	Model      string  `json:"model,omitempty"`
	Score      float64 `json:"score,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Reasoning  string  `json:"reasoning,omitempty"`
}

type ManualScoreRequest struct {
	Score    float64 `json:"score"`
	Source   string  `json:"source,omitempty"`
	Comments string  `json:"comments,omitempty"`
}

type FeedHealth map[string]bool

// convertArticles converts raw client articles to wrapper articles
func convertArticles(rawArticles []rawclient.Article) []Article {
	articles := make([]Article, len(rawArticles))
	for i, raw := range rawArticles {
		articles[i] = *convertArticle(&raw)
	}
	return articles
}

// convertArticle converts a single raw client article to wrapper article
func convertArticle(raw *rawclient.Article) *Article {
	if raw == nil {
		return nil
	}

	return &Article{
		ArticleID:      raw.ArticleID,
		Title:          raw.Title,
		Content:        raw.Content,
		URL:            raw.URL,
		Source:         raw.Source,
		PubDate:        raw.PubDate,
		CreatedAt:      raw.CreatedAt,
		CompositeScore: raw.CompositeScore,
		Confidence:     raw.Confidence,
		ScoreSource:    raw.ScoreSource,
		BiasLabel:      raw.BiasLabel,
		AnalysisNotes:  raw.AnalysisNotes,
	}
}

// convertScoreResponse converts raw score response to wrapper score response
func convertScoreResponse(raw *rawclient.ScoreResponse) *ScoreResponse {
	if raw == nil {
		return nil
	}

	details := make([]IndividualScoreResult, len(raw.Details))
	for i, rawDetail := range raw.Details {
		details[i] = IndividualScoreResult{
			Model:      rawDetail.Model,
			Score:      rawDetail.Score,
			Confidence: rawDetail.Confidence,
			Reasoning:  rawDetail.Reasoning,
		}
	}

	return &ScoreResponse{
		ArticleID: raw.ArticleID,
		Score:     raw.Score,
		Details:   details,
	}
}

// translateError converts raw client errors to wrapper errors with better error classification
func (c *APIClient) translateError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific API errors defined in the schema first.
	// These are errors that the server returned in a structured format and were successfully decoded.
	if specificAPIErr, ok := err.(*rawclient.APIError); ok {
		return APIError{
			StatusCode: determineStatusCode(specificAPIErr.Code),
			Code:       specificAPIErr.Code,
			Message:    specificAPIErr.Message,
			Details:    fmt.Sprintf("%v", specificAPIErr.Details), // Ensure details are stringified
		}
	}

	// Check for JSON unmarshalling errors, which might indicate a mismatch
	// between the expected response structure and what the server sent,
	// or what the mock server sent in tests.
	// The actual error from client.go's json.Unmarshal will be a generic error if it's not APIError
	// For example, if GetArticles fails to unmarshal the body into []Article, the error comes from:
	//   if err := json.Unmarshal(articlesData, &articles); err != nil { return nil, err }
	// This 'err' is what we receive here.
	errStr := err.Error()
	if strings.Contains(errStr, "json: cannot unmarshal") || strings.Contains(errStr, "json: syntax error") {
		fmt.Printf("DEBUG: translateError: Encountered JSON unmarshal error. Error(): %s\\n", errStr)
		return APIError{
			StatusCode: http.StatusInternalServerError,
			Code:       "client_unmarshal_error",
			Message:    "Client failed to process API response due to an unmarshalling error.",
			Details:    errStr,
		}
	}

	// Handle common error patterns (fallback if not specific *rawclient.APIError or JSON error)
	// errStr is already defined above

	// Network errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "timeout") && !strings.Contains(errStr, "context deadline exceeded") { // Avoid double-handling context timeout
		return APIError{
			StatusCode: http.StatusServiceUnavailable,
			Code:       "network_error",
			Message:    "Unable to connect to API server",
			Details:    errStr,
		}
	}

	// Context errors
	if strings.Contains(errStr, "context deadline exceeded") {
		return APIError{
			StatusCode: http.StatusRequestTimeout,
			Code:       "timeout",
			Message:    "Request timed out",
			Details:    errStr,
		}
	}

	if strings.Contains(errStr, "context canceled") {
		return APIError{
			StatusCode: http.StatusRequestTimeout,
			Code:       "canceled",
			Message:    "Request was canceled",
			Details:    errStr,
		}
	}

	// Parse HTTP status from error string if possible
	if strings.Contains(errStr, "HTTP ") {
		parts := strings.Split(errStr, "HTTP ")
		if len(parts) > 1 {
			statusPart := strings.Split(parts[1], ":")[0]
			if statusCode, parseErr := strconv.Atoi(statusPart); parseErr == nil {
				return APIError{
					StatusCode: statusCode,
					Code:       getErrorCodeFromStatus(statusCode),
					Message:    getErrorMessageFromStatus(statusCode),
					Details:    errStr,
				}
			}
		}
	}

	// Default to internal error
	return APIError{
		StatusCode: http.StatusInternalServerError,
		Code:       "internal_error",
		Message:    "An unexpected error occurred",
		Details:    errStr,
	}
}

// determineStatusCode maps error codes to HTTP status codes
func determineStatusCode(code string) int {
	switch strings.ToLower(code) {
	case "not_found", "article_not_found":
		return http.StatusNotFound
	case "validation_error", "invalid_request", "bad_request":
		return http.StatusBadRequest
	case "unauthorized", "authentication_error":
		return http.StatusUnauthorized
	case "forbidden", "access_denied":
		return http.StatusForbidden
	case "rate_limit", "too_many_requests":
		return http.StatusTooManyRequests
	case "llm_service", "service_unavailable":
		return http.StatusServiceUnavailable
	case "timeout":
		return http.StatusRequestTimeout
	case "conflict", "duplicate_url":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// getErrorCodeFromStatus maps HTTP status codes to error codes
func getErrorCodeFromStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusRequestTimeout:
		return "timeout"
	case http.StatusTooManyRequests:
		return "rate_limit"
	case http.StatusInternalServerError:
		return "internal_error"
	case http.StatusBadGateway:
		return "bad_gateway"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	case http.StatusGatewayTimeout:
		return "gateway_timeout"
	default:
		return "unknown_error"
	}
}

// getErrorMessageFromStatus provides user-friendly error messages for HTTP status codes
func getErrorMessageFromStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "Invalid request format or parameters"
	case http.StatusUnauthorized:
		return "Authentication required"
	case http.StatusForbidden:
		return "Access denied"
	case http.StatusNotFound:
		return "Resource not found"
	case http.StatusConflict:
		return "Resource already exists or conflict detected"
	case http.StatusRequestTimeout:
		return "Request timed out"
	case http.StatusTooManyRequests:
		return "Too many requests - rate limit exceeded"
	case http.StatusInternalServerError:
		return "Internal server error"
	case http.StatusBadGateway:
		return "Bad gateway"
	case http.StatusServiceUnavailable:
		return "Service temporarily unavailable"
	case http.StatusGatewayTimeout:
		return "Gateway timeout"
	default:
		return fmt.Sprintf("HTTP error %d", statusCode)
	}
}

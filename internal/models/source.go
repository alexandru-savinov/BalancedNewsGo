package models

import "time"

// Source represents a news source with channel-specific configuration
// @Description A news source configuration for content ingestion
// swagger:model Source
type Source struct {
	ID            int64      `json:"id" example:"1"`                                         // Unique identifier
	Name          string     `json:"name" example:"CNN Politics"`                            // Display name
	ChannelType   string     `json:"channel_type" example:"rss"`                             // Channel type (rss, telegram, etc.)
	FeedURL       string     `json:"feed_url" example:"https://rss.cnn.com/rss/edition.rss"` // RSS URL or channel identifier
	Category      string     `json:"category" example:"center"`                              // Political category (left, center, right)
	Enabled       bool       `json:"enabled" example:"true"`                                 // Active/inactive toggle
	DefaultWeight float64    `json:"default_weight" example:"1.0"`                           // Scoring weight multiplier
	LastFetchedAt *time.Time `json:"last_fetched_at,omitempty"`                              // Last successful fetch
	ErrorStreak   int        `json:"error_streak" example:"0"`                               // Consecutive error count
	Metadata      *string    `json:"metadata,omitempty"`                                     // JSON for channel-specific config
	CreatedAt     time.Time  `json:"created_at" example:"2024-01-01T00:00:00Z"`              // Creation timestamp
	UpdatedAt     time.Time  `json:"updated_at" example:"2024-01-01T00:00:00Z"`              // Last update timestamp
}

// SourceStats represents aggregated statistics for a source
// @Description Statistical information about a news source
// swagger:model SourceStats
type SourceStats struct {
	SourceID      int64      `json:"source_id" example:"1"`                      // Source identifier
	ArticleCount  int64      `json:"article_count" example:"150"`                // Total articles from this source
	AvgScore      *float64   `json:"avg_score,omitempty" example:"0.05"`         // Average composite score
	LastArticleAt *time.Time `json:"last_article_at,omitempty"`                  // Most recent article timestamp
	ComputedAt    time.Time  `json:"computed_at" example:"2024-01-01T12:00:00Z"` // When stats were computed
}

// SourceWithStats combines source information with statistics
// @Description Source information with aggregated statistics
// swagger:model SourceWithStats
type SourceWithStats struct {
	Source
	Stats *SourceStats `json:"stats,omitempty"` // Optional statistics
}

// CreateSourceRequest represents a request to create a new source
// @Description Request body for creating a new news source
// swagger:model CreateSourceRequest
type CreateSourceRequest struct {
	Name          string  `json:"name" binding:"required" example:"CNN Politics"`                                // Display name (required)
	ChannelType   string  `json:"channel_type" binding:"required" example:"rss"`                                 // Channel type (required)
	FeedURL       string  `json:"feed_url" binding:"required,url" example:"https://rss.cnn.com/rss/edition.rss"` // Feed URL (required, must be valid URL)
	Category      string  `json:"category" binding:"required,oneof=left center right" example:"center"`          // Political category (required)
	DefaultWeight float64 `json:"default_weight,omitempty" example:"1.0"`                                        // Scoring weight (optional, defaults to 1.0)
	Metadata      *string `json:"metadata,omitempty"`                                                            // Channel-specific configuration (optional)
}

// UpdateSourceRequest represents a request to update an existing source
// @Description Request body for updating an existing news source
// swagger:model UpdateSourceRequest
type UpdateSourceRequest struct {
	Name          *string  `json:"name,omitempty" example:"Updated CNN Politics"`                     // Display name (optional)
	FeedURL       *string  `json:"feed_url,omitempty" example:"https://rss.cnn.com/rss/politics.rss"` // Feed URL (optional, must be valid URL if provided)
	Category      *string  `json:"category,omitempty" example:"center"`                               // Political category (optional)
	Enabled       *bool    `json:"enabled,omitempty" example:"true"`                                  // Active status (optional)
	DefaultWeight *float64 `json:"default_weight,omitempty" example:"1.5"`                            // Scoring weight (optional)
	Metadata      *string  `json:"metadata,omitempty"`                                                // Channel-specific configuration (optional)
}

// SourceListResponse represents a paginated list of sources
// @Description Paginated response containing sources
// swagger:model SourceListResponse
type SourceListResponse struct {
	Sources []SourceWithStats `json:"sources"`            // List of sources with optional stats
	Total   int64             `json:"total" example:"25"` // Total number of sources (for pagination)
	Limit   int               `json:"limit" example:"50"` // Page size limit
	Offset  int               `json:"offset" example:"0"` // Pagination offset
}

// SourceHealthStatus represents the health status of a source
// @Description Health check status for a news source
// swagger:model SourceHealthStatus
type SourceHealthStatus struct {
	SourceID     int64     `json:"source_id" example:"1"`                        // Source identifier
	SourceName   string    `json:"source_name" example:"CNN Politics"`           // Source display name
	IsHealthy    bool      `json:"is_healthy" example:"true"`                    // Overall health status
	LastCheckAt  time.Time `json:"last_check_at" example:"2024-01-01T12:00:00Z"` // Last health check timestamp
	ErrorMessage *string   `json:"error_message,omitempty"`                      // Error message if unhealthy
	ResponseTime *int64    `json:"response_time,omitempty" example:"250"`        // Response time in milliseconds
	StatusCode   *int      `json:"status_code,omitempty" example:"200"`          // HTTP status code
}

// Source channel types
const (
	ChannelTypeRSS      = "rss"      // RSS feed
	ChannelTypeTelegram = "telegram" // Telegram channel
	ChannelTypeTwitter  = "twitter"  // Twitter/X feed
	ChannelTypeReddit   = "reddit"   // Reddit subreddit
)

// Source categories
const (
	CategoryLeft   = "left"   // Left-leaning sources
	CategoryCenter = "center" // Centrist sources
	CategoryRight  = "right"  // Right-leaning sources
)

// ValidChannelTypes returns a list of valid channel types
func ValidChannelTypes() []string {
	return []string{ChannelTypeRSS, ChannelTypeTelegram, ChannelTypeTwitter, ChannelTypeReddit}
}

// ValidCategories returns a list of valid political categories
func ValidCategories() []string {
	return []string{CategoryLeft, CategoryCenter, CategoryRight}
}

// IsValidChannelType checks if a channel type is valid
func IsValidChannelType(channelType string) bool {
	for _, valid := range ValidChannelTypes() {
		if channelType == valid {
			return true
		}
	}
	return false
}

// IsValidCategory checks if a category is valid
func IsValidCategory(category string) bool {
	for _, valid := range ValidCategories() {
		if category == valid {
			return true
		}
	}
	return false
}

// ToUpdateMap converts UpdateSourceRequest to a map for database updates
func (r *UpdateSourceRequest) ToUpdateMap() map[string]interface{} {
	updates := make(map[string]interface{})

	if r.Name != nil {
		updates["name"] = *r.Name
	}
	if r.FeedURL != nil {
		updates["feed_url"] = *r.FeedURL
	}
	if r.Category != nil {
		updates["category"] = *r.Category
	}
	if r.Enabled != nil {
		updates["enabled"] = *r.Enabled
	}
	if r.DefaultWeight != nil {
		updates["default_weight"] = *r.DefaultWeight
	}
	if r.Metadata != nil {
		updates["metadata"] = *r.Metadata
	}

	return updates
}

// Validate validates the CreateSourceRequest
func (r *CreateSourceRequest) Validate() error {
	if r.Name == "" {
		return ErrSourceNameRequired
	}
	if r.ChannelType == "" {
		return ErrSourceChannelTypeRequired
	}
	if !IsValidChannelType(r.ChannelType) {
		return ErrSourceInvalidChannelType
	}
	if r.FeedURL == "" {
		return ErrSourceFeedURLRequired
	}
	if r.Category == "" {
		return ErrSourceCategoryRequired
	}
	if !IsValidCategory(r.Category) {
		return ErrSourceInvalidCategory
	}
	if r.DefaultWeight < 0 {
		return ErrSourceInvalidWeight
	}
	return nil
}

// Validate validates the UpdateSourceRequest
func (r *UpdateSourceRequest) Validate() error {
	if r.Category != nil && !IsValidCategory(*r.Category) {
		return ErrSourceInvalidCategory
	}
	if r.DefaultWeight != nil && *r.DefaultWeight < 0 {
		return ErrSourceInvalidWeight
	}
	return nil
}

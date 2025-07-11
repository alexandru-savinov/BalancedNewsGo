package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFeedURL    = "https://example.com/feed.xml"
	testSourceName = "Test Source"
)

func TestCreateSourceRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateSourceRequest
		wantErr bool
		errType error
	}{
		{
			name: "valid RSS source",
			req: CreateSourceRequest{
				Name:          "Test RSS Source",
				ChannelType:   "rss",
				FeedURL:       testFeedURL,
				Category:      "center",
				DefaultWeight: 1.0,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: CreateSourceRequest{
				ChannelType:   "rss",
				FeedURL:       "https://example.com/feed.xml",
				Category:      "center",
				DefaultWeight: 1.0,
			},
			wantErr: true,
			errType: ErrSourceNameRequired,
		},
		{
			name: "missing channel type",
			req: CreateSourceRequest{
				Name:          "Test Source",
				FeedURL:       "https://example.com/feed.xml",
				Category:      "center",
				DefaultWeight: 1.0,
			},
			wantErr: true,
			errType: ErrSourceChannelTypeRequired,
		},
		{
			name: "invalid channel type",
			req: CreateSourceRequest{
				Name:          "Test Source",
				ChannelType:   "invalid",
				FeedURL:       "https://example.com/feed.xml",
				Category:      "center",
				DefaultWeight: 1.0,
			},
			wantErr: true,
			errType: ErrSourceInvalidChannelType,
		},
		{
			name: "RSS source missing feed URL",
			req: CreateSourceRequest{
				Name:          "Test RSS Source",
				ChannelType:   "rss",
				Category:      "center",
				DefaultWeight: 1.0,
			},
			wantErr: true,
			errType: ErrSourceFeedURLRequired,
		},
		{
			name: "invalid category",
			req: CreateSourceRequest{
				Name:          "Test Source",
				ChannelType:   "rss",
				FeedURL:       "https://example.com/feed.xml",
				Category:      "invalid",
				DefaultWeight: 1.0,
			},
			wantErr: true,
			errType: ErrSourceInvalidCategory,
		},
		{
			name: "weight negative",
			req: CreateSourceRequest{
				Name:          testSourceName,
				ChannelType:   "rss",
				FeedURL:       testFeedURL,
				Category:      "center",
				DefaultWeight: -1.0,
			},
			wantErr: true,
			errType: ErrSourceInvalidWeight,
		},
		{
			name: "missing category",
			req: CreateSourceRequest{
				Name:          testSourceName,
				ChannelType:   "rss",
				FeedURL:       testFeedURL,
				DefaultWeight: 1.0,
			},
			wantErr: true,
			errType: ErrSourceCategoryRequired,
		},
		{
			name: "valid telegram source",
			req: CreateSourceRequest{
				Name:          "Test Telegram Source",
				ChannelType:   "telegram",
				FeedURL:       "@testchannel",
				Category:      "left",
				DefaultWeight: 2.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateSourceRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateSourceRequest
		wantErr bool
		errType error
	}{
		{
			name: "valid update",
			req: UpdateSourceRequest{
				Name:          stringPtr("Updated Source"),
				FeedURL:       stringPtr("https://example.com/updated-feed.xml"),
				Category:      stringPtr("right"),
				Enabled:       boolPtr(false),
				DefaultWeight: float64Ptr(1.5),
			},
			wantErr: false,
		},

		{
			name: "invalid category",
			req: UpdateSourceRequest{
				Category: stringPtr("invalid"),
			},
			wantErr: true,
			errType: ErrSourceInvalidCategory,
		},
		{
			name: "weight too low",
			req: UpdateSourceRequest{
				DefaultWeight: float64Ptr(-1.0),
			},
			wantErr: true,
			errType: ErrSourceInvalidWeight,
		},
		{
			name: "partial update - only name",
			req: UpdateSourceRequest{
				Name: stringPtr("New Name"),
			},
			wantErr: false,
		},
		{
			name: "partial update - only enabled",
			req: UpdateSourceRequest{
				Enabled: boolPtr(true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateSourceRequest_ToUpdateMap(t *testing.T) {
	name := "Updated Source"
	enabled := true
	weight := 1.5
	category := "right"
	feedURL := "https://example.com/feed.xml"

	req := UpdateSourceRequest{
		Name:          &name,
		FeedURL:       &feedURL,
		Category:      &category,
		Enabled:       &enabled,
		DefaultWeight: &weight,
	}

	updateMap := req.ToUpdateMap()

	assert.Equal(t, "Updated Source", updateMap["name"])
	assert.Equal(t, "https://example.com/feed.xml", updateMap["feed_url"])
	assert.Equal(t, "right", updateMap["category"])
	assert.Equal(t, true, updateMap["enabled"])
	assert.Equal(t, 1.5, updateMap["default_weight"])
}

func TestUpdateSourceRequest_ToUpdateMap_PartialUpdate(t *testing.T) {
	name := "Only Name Updated"

	req := UpdateSourceRequest{
		Name: &name,
	}

	updateMap := req.ToUpdateMap()

	assert.Equal(t, "Only Name Updated", updateMap["name"])
	assert.NotContains(t, updateMap, "feed_url")
	assert.NotContains(t, updateMap, "category")
	assert.NotContains(t, updateMap, "enabled")
	assert.NotContains(t, updateMap, "default_weight")
}

func TestValidChannelTypes(t *testing.T) {
	types := ValidChannelTypes()
	expected := []string{"rss", "telegram", "twitter", "reddit"}
	assert.Equal(t, expected, types)
}

func TestValidCategories(t *testing.T) {
	categories := ValidCategories()
	expected := []string{"left", "center", "right"}
	assert.Equal(t, expected, categories)
}

func TestIsValidChannelType(t *testing.T) {
	tests := []struct {
		channelType string
		expected    bool
	}{
		{"rss", true},
		{"telegram", true},
		{"twitter", true},
		{"reddit", true},
		{"invalid", false},
		{"", false},
		{"RSS", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.channelType, func(t *testing.T) {
			result := IsValidChannelType(tt.channelType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		category string
		expected bool
	}{
		{"left", true},
		{"center", true},
		{"right", true},
		{"invalid", false},
		{"", false},
		{"LEFT", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := IsValidCategory(tt.category)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSource_Basic(t *testing.T) {
	now := time.Now()
	source := Source{
		ID:            1,
		Name:          "Test Source",
		ChannelType:   "rss",
		FeedURL:       "https://example.com/feed.xml",
		Category:      "center",
		Enabled:       true,
		DefaultWeight: 1.0,
		ErrorStreak:   0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, int64(1), source.ID)
	assert.Equal(t, "Test Source", source.Name)
	assert.Equal(t, "rss", source.ChannelType)
	assert.Equal(t, "https://example.com/feed.xml", source.FeedURL)
	assert.Equal(t, "center", source.Category)
	assert.True(t, source.Enabled)
	assert.Equal(t, 1.0, source.DefaultWeight)
	assert.Equal(t, 0, source.ErrorStreak)
	assert.Equal(t, now, source.CreatedAt)
	assert.Equal(t, now, source.UpdatedAt)
}

func TestSourceStats_Basic(t *testing.T) {
	now := time.Now()
	avgScore := 0.75

	stats := SourceStats{
		SourceID:      1,
		ArticleCount:  150,
		AvgScore:      &avgScore,
		LastArticleAt: &now,
		ComputedAt:    now,
	}

	assert.Equal(t, int64(1), stats.SourceID)
	assert.Equal(t, int64(150), stats.ArticleCount)
	require.NotNil(t, stats.AvgScore)
	assert.Equal(t, 0.75, *stats.AvgScore)
	require.NotNil(t, stats.LastArticleAt)
	assert.Equal(t, now, *stats.LastArticleAt)
	assert.Equal(t, now, stats.ComputedAt)
}

func TestSourceWithStats_Basic(t *testing.T) {
	now := time.Now()
	avgScore := 0.75

	source := Source{
		ID:            1,
		Name:          "Test Source",
		ChannelType:   "rss",
		FeedURL:       "https://example.com/feed.xml",
		Category:      "center",
		Enabled:       true,
		DefaultWeight: 1.0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	stats := &SourceStats{
		SourceID:      1,
		ArticleCount:  150,
		AvgScore:      &avgScore,
		LastArticleAt: &now,
		ComputedAt:    now,
	}

	sourceWithStats := SourceWithStats{
		Source: source,
		Stats:  stats,
	}

	assert.Equal(t, int64(1), sourceWithStats.ID)
	assert.Equal(t, "Test Source", sourceWithStats.Name)
	require.NotNil(t, sourceWithStats.Stats)
	assert.Equal(t, int64(150), sourceWithStats.Stats.ArticleCount)
}

func TestSourceListResponse_Basic(t *testing.T) {
	sources := []SourceWithStats{
		{
			Source: Source{
				ID:   1,
				Name: "Source 1",
			},
		},
		{
			Source: Source{
				ID:   2,
				Name: "Source 2",
			},
		},
	}

	response := SourceListResponse{
		Sources: sources,
		Total:   2,
		Limit:   50,
		Offset:  0,
	}

	assert.Len(t, response.Sources, 2)
	assert.Equal(t, int64(2), response.Total)
	assert.Equal(t, 50, response.Limit)
	assert.Equal(t, 0, response.Offset)
}

func TestSourceHealthStatus_Basic(t *testing.T) {
	now := time.Now()
	errorMsg := "Connection timeout"
	responseTime := int64(250)
	statusCode := 200

	health := SourceHealthStatus{
		SourceID:     1,
		SourceName:   "Test Source",
		IsHealthy:    false,
		LastCheckAt:  now,
		ErrorMessage: &errorMsg,
		ResponseTime: &responseTime,
		StatusCode:   &statusCode,
	}

	assert.Equal(t, int64(1), health.SourceID)
	assert.Equal(t, "Test Source", health.SourceName)
	assert.False(t, health.IsHealthy)
	assert.Equal(t, now, health.LastCheckAt)
	require.NotNil(t, health.ErrorMessage)
	assert.Equal(t, "Connection timeout", *health.ErrorMessage)
	require.NotNil(t, health.ResponseTime)
	assert.Equal(t, int64(250), *health.ResponseTime)
	require.NotNil(t, health.StatusCode)
	assert.Equal(t, 200, *health.StatusCode)
}

func TestSourceConstants(t *testing.T) {
	// Test channel type constants
	assert.Equal(t, "rss", ChannelTypeRSS)
	assert.Equal(t, "telegram", ChannelTypeTelegram)
	assert.Equal(t, "twitter", ChannelTypeTwitter)
	assert.Equal(t, "reddit", ChannelTypeReddit)

	// Test category constants
	assert.Equal(t, "left", CategoryLeft)
	assert.Equal(t, "center", CategoryCenter)
	assert.Equal(t, "right", CategoryRight)
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}

package main

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestExtractFilterParamsWithAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/?source=x&leaning=Left&query=foo&page=3", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	params := extractFilterParams(c)
	assert.Equal(t, "x", params.Source)
	assert.Equal(t, "Left", params.Leaning)
	assert.Equal(t, "foo", params.Query)
	assert.Equal(t, 20, params.Limit)
	assert.Equal(t, 40, params.Offset) // (3-1)*20
	assert.Equal(t, 3, params.CurrentPage)
}

func TestBuildSearchQueryWithFilters(t *testing.T) {
	params := FilterParams{Source: "s", Leaning: "Right", Query: "bar", Limit: 5, Offset: 0}
	q, args := buildSearchQuery(params)
	assert.Contains(t, q, "AND source = ?")
	assert.Contains(t, q, "composite_score > 0.1")
	assert.Contains(t, q, "title LIKE")
	assert.Len(t, args, 5) // Source (1) + Query (2) + Limit (1) + Offset (1) = 5
}

func TestConvertToTemplateDataBiasAndExcerpt(t *testing.T) {
	now := time.Now()
	score := 0.5
	conf := 0.9
	source := "src"

	// Create a test db.Article and convert it to template data
	td := convertToTemplateData(&db.Article{
		ID:             1,
		Title:          "T",
		Content:        string(make([]byte, 250)), // content longer for excerpt
		URL:            "u",
		Source:         source,
		PubDate:        now,
		CreatedAt:      now,
		CompositeScore: &score,
		Confidence:     &conf,
		ScoreSource:    &source,
	})

	assert.Equal(t, "Right Leaning", td.BiasLabel)
	assert.True(t, len(td.Excerpt) < 250) // excerpt should be shorter than original content
	assert.Equal(t, score, td.CompositeScore)
	assert.Equal(t, conf, td.Confidence)
	assert.Equal(t, source, td.ScoreSource)
}

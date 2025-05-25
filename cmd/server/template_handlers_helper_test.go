package main

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestExtractFilterParams_WithAll(t *testing.T) {
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

func TestBuildSearchQuery_WithFilters(t *testing.T) {
	params := FilterParams{Source: "s", Leaning: "Right", Query: "bar", Limit: 5, Offset: 0}
	q, args := buildSearchQuery(params)
	assert.Contains(t, q, "AND source = ?")
	assert.Contains(t, q, "composite_score > 0.1")
	assert.Contains(t, q, "title LIKE")
	assert.Len(t, args, 5) // Source (1) + Query (2) + Limit (1) + Offset (1) = 5
}

func TestConvertToTemplateData_BiasAndExcerpt(t *testing.T) {
	now := time.Now()
	score := 0.5
	conf := 0.9
	source := "src"
	art := &ArticleTemplateData{ // content longer for excerpt
		ID:             1,
		Title:          "T",
		Content:        string(make([]byte, 250)),
		URL:            "u",
		Source:         source,
		PubDate:        now,
		CreatedAt:      now,
		CompositeScore: score,
		Confidence:     conf,
		ScoreSource:    source,
	}
	td := convertToTemplateData(&db.Article{ // convert from db.Article
		ID:             art.ID,
		Title:          art.Title,
		Content:        art.Content,
		URL:            art.URL,
		Source:         art.Source,
		PubDate:        art.PubDate,
		CreatedAt:      art.CreatedAt,
		CompositeScore: &score,
		Confidence:     &conf,
		ScoreSource:    &source,
	})

	assert.Equal(t, "Right Leaning", td.BiasLabel)
	assert.True(t, len(td.Excerpt) < len(art.Content))
}

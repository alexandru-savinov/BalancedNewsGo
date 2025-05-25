package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestExtractFilterParams_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	params := extractFilterParams(c)
	assert.Equal(t, "", params.Source)
	assert.Equal(t, "", params.Leaning)
	assert.Equal(t, "", params.Query)
	assert.Equal(t, 20, params.Limit)
	assert.Equal(t, 0, params.Offset)
	assert.Equal(t, 1, params.CurrentPage)
}

func TestExtractFilterParams_PageAndBias(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/?bias=Right&page=2", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	params := extractFilterParams(c)
	assert.Equal(t, "Right", params.Leaning)
	assert.Equal(t, 20, params.Limit)
	assert.Equal(t, 20, params.Offset)
	assert.Equal(t, 2, params.CurrentPage)
}

func TestBuildSearchQuery_NoFilters(t *testing.T) {
	q, args := buildSearchQuery(FilterParams{Limit: 10, Offset: 5})
	assert.Contains(t, q, "SELECT * FROM articles WHERE 1=1")
	assert.Contains(t, q, "LIMIT ? OFFSET ?")
	assert.Len(t, args, 2)
}

func TestBuildSearchQuery_WithAll(t *testing.T) {
	params := FilterParams{Source: "src", Leaning: "Left", Query: "foo", Limit: 3, Offset: 1}
	q, args := buildSearchQuery(params)
	assert.Contains(t, q, "source = ?")
	assert.Contains(t, q, "composite_score < -0.1")
	assert.Contains(t, q, "title LIKE")
	// args: source, term, term, limit+1, offset
	assert.Len(t, args, 5)
	assert.Equal(t, params.Source, args[0])
}

func TestGetBiasLabel(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{-1, "Left Leaning"},
		{0, "Center"},
		{1, "Right Leaning"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, getBiasLabel(c.score))
	}
}

func TestGetStringValue(t *testing.T) {
	var s *string
	r := getStringValue(s)
	assert.Equal(t, "", r)
	val := "ok"
	r2 := getStringValue(&val)
	assert.Equal(t, "ok", r2)
}

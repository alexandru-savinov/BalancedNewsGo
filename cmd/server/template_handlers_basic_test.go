package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestExtractFilterParamsDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	params := extractFilterParams(c)
	assert.Equal(t, "", params.Source)
	assert.Equal(t, "", params.Leaning)
	assert.Equal(t, "", params.Query)
}

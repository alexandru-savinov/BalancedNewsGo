package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func setupTestDB() *sqlx.DB {
	dbConn, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic("failed to connect to in-memory sqlite: " + err.Error())
	}
	schema := `
	CREATE TABLE llm_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER,
		model TEXT,
		score REAL,
		metadata TEXT,
		created_at DATETIME
	);`
	dbConn.MustExec(schema)
	return dbConn
}

func insertMockScore(dbConn *sqlx.DB, articleID int64, model string, score float64, metadata string) {
	dbConn.MustExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at) VALUES (?, ?, ?, ?, ?)`,
		articleID, model, score, metadata, time.Now())
}

func TestBiasHandlerFilterSort(t *testing.T) {
	dbConn := setupTestDB()
	router := gin.Default()
	router.GET("/api/articles/:id/bias", biasHandler(dbConn))

	meta1, _ := json.Marshal(map[string]interface{}{
		"aggregation": map[string]interface{}{"weighted_mean": 0.2},
	})
	meta2, _ := json.Marshal(map[string]interface{}{
		"aggregation": map[string]interface{}{"weighted_mean": -0.5},
	})
	insertMockScore(dbConn, 1, "ensemble", 0.2, string(meta1))
	insertMockScore(dbConn, 1, "ensemble", -0.5, string(meta2))

	req, _ := http.NewRequest("GET", "/api/articles/1/bias?min_score=-1&max_score=1&sort=asc", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	var body map[string][]map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	assert.NoError(t, err)
	results := body["results"]
	assert.Len(t, results, 2)
	assert.True(t, results[0]["score"].(float64) < results[1]["score"].(float64))
}

func TestEnsembleDetailsHandler(t *testing.T) {
	dbConn := setupTestDB()
	router := gin.Default()
	router.GET("/api/articles/:id/ensemble", ensembleDetailsHandler(dbConn))

	subResults := []map[string]interface{}{
		{"model": "gpt-4", "score": 0.3, "confidence": 0.9},
		{"model": "claude", "score": -0.2, "confidence": 0.8},
	}
	meta, _ := json.Marshal(map[string]interface{}{
		"sub_results": subResults,
		"aggregation": map[string]interface{}{
			"weighted_mean":    0.1,
			"variance":         0.05,
			"uncertainty_flag": false,
		},
	})
	insertMockScore(dbConn, 2, "ensemble", 0.1, string(meta))

	req, _ := http.NewRequest("GET", "/api/articles/2/ensemble", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	var body map[string][]map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	assert.NoError(t, err)
	ensembles := body["ensembles"]
	assert.Len(t, ensembles, 1)
	agg := ensembles[0]["aggregation"].(map[string]interface{})
	assert.Equal(t, 0.1, agg["weighted_mean"])
}

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
	var raw map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &raw)
	assert.NoError(t, err)

	resultsRaw, ok := raw["results"]
	assert.True(t, ok, "response missing 'results' key")

	resultsSlice, ok := resultsRaw.([]interface{})
	assert.True(t, ok, "'results' is not an array")

	assert.Len(t, resultsSlice, 2)

	score0 := resultsSlice[0].(map[string]interface{})["score"].(float64)
	score1 := resultsSlice[1].(map[string]interface{})["score"].(float64)
	assert.True(t, score0 < score1)
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

// Automated E2E check for API/DB validation of new articles and CompositeScores
func TestAPIAndDBValidationForCompositeScores(t *testing.T) {
	dbConn := setupTestDB()
	router := gin.Default()
	router.GET("/api/articles/:id/ensemble", ensembleDetailsHandler(dbConn))

	// Insert a mock CompositeScore for a new article
	articleID := int64(42)
	expectedScore := 0.75
	expectedModel := "ensemble"
	meta, _ := json.Marshal(map[string]interface{}{
		"aggregation": map[string]interface{}{
			"weighted_mean":    expectedScore,
			"variance":         0.02,
			"uncertainty_flag": false,
		},
	})
	insertMockScore(dbConn, articleID, expectedModel, expectedScore, string(meta))

	// Query DB directly
	var dbScore float64
	var dbModel string
	var dbMetadata string
	err := dbConn.QueryRowx(
		"SELECT score, model, metadata FROM llm_scores WHERE article_id = ? AND model = ?",
		articleID, expectedModel,
	).Scan(&dbScore, &dbModel, &dbMetadata)
	if err != nil {
		t.Errorf("DB validation failed: could not find CompositeScore for article_id=%d: %v", articleID, err)
	} else {
		t.Logf("DB validation: found CompositeScore for article_id=%d: model=%s, score=%v", articleID, dbModel, dbScore)
	}

	// Call API to fetch CompositeScore
	req, _ := http.NewRequest("GET", "/api/articles/42/ensemble", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Errorf("API validation failed: expected 200, got %d", resp.Code)
		return
	}

	var apiBody map[string][]map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &apiBody)
	if err != nil {
		t.Errorf("API validation failed: could not parse response: %v", err)
		return
	}
	ensembles, ok := apiBody["ensembles"]
	if !ok || len(ensembles) == 0 {
		t.Errorf("API validation failed: no ensembles found in response for article_id=%d", articleID)
		return
	}
	apiAgg, ok := ensembles[0]["aggregation"].(map[string]interface{})
	if !ok {
		t.Errorf("API validation failed: aggregation missing or wrong type in API response")
		return
	}
	apiScore, ok := apiAgg["weighted_mean"].(float64)
	if !ok {
		t.Errorf("API validation failed: weighted_mean missing or wrong type in API response")
		return
	}

	// Compare API and DB results
	if apiScore != dbScore {
		t.Errorf("Discrepancy: API score (%v) does not match DB score (%v) for article_id=%d", apiScore, dbScore, articleID)
	} else {
		t.Logf("Validation successful: API and DB scores match for article_id=%d: %v", articleID, apiScore)
	}
}

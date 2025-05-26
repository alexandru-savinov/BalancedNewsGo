package db_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// openFilterTestDB sets up a new in-memory DB
func openFilterTestDB(t *testing.T) *sqlx.DB {
	dbInstance, err := db.New(":memory:")
	assert.NoError(t, err)

	// Create necessary tables for testing
	_, err = dbInstance.DB.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			pub_date TIMESTAMP NOT NULL,
			url TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			composite_score REAL,
			confidence REAL,
			score_source TEXT,
			status TEXT, -- Added missing column
			fail_count INTEGER,
			last_attempt TIMESTAMP,
			escalated BOOLEAN
		);

		CREATE TABLE IF NOT EXISTS llm_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			article_id INTEGER NOT NULL,
			model TEXT NOT NULL,
			score REAL NOT NULL,
			metadata TEXT,
			version TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (article_id) REFERENCES articles (id)
		);

		CREATE TABLE IF NOT EXISTS feedback (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			article_id INTEGER NOT NULL,
			user_id TEXT,
			feedback_text TEXT,
			category TEXT,
			ensemble_output_id INTEGER,
			source TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (article_id) REFERENCES articles (id)
		);

		CREATE TABLE IF NOT EXISTS labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data TEXT NOT NULL,
			label TEXT NOT NULL,
			source TEXT NOT NULL,
			date_labeled TIMESTAMP NOT NULL,
			labeler TEXT NOT NULL,
			confidence REAL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	assert.NoError(t, err)

	return dbInstance.DB
}

func TestFetchArticlesFiltering(t *testing.T) {
	dbConn := openFilterTestDB(t)
	// insert articles with different source and composite_score
	articles := []struct {
		source string
		score  float64
	}{{"A", -0.5}, {"B", 0.0}, {"A", 0.5}}
	for i, a := range articles {
		id, err := db.InsertArticle(dbConn, &db.Article{
			Source:    a.source,
			PubDate:   time.Now(),
			URL:       "url" + strconv.Itoa(i),
			Title:     "t",
			Content:   "c",
			CreatedAt: time.Now(),
		})
		assert.NoError(t, err)
		// set composite_score and confidence via UpdateArticleScore
		err = db.UpdateArticleScore(dbConn, id, a.score, 1.0)
		assert.NoError(t, err)
	}

	// no filter: expect 3
	all, err := db.FetchArticles(dbConn, "", "", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, all, 3)

	// source filter A: expect 2
	aA, err := db.FetchArticles(dbConn, "A", "", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, aA, 2)

	// leaning left (< -0.1): expect one
	left, err := db.FetchArticles(dbConn, "", "left", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, left, 1)

	// leaning center (-0.1 <= score <= 0.1): expect one
	center, err := db.FetchArticles(dbConn, "", "center", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, center, 1)

	// leaning right (> 0.1): expect one
	right, err := db.FetchArticles(dbConn, "", "right", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, right, 1)
}

func TestMigrateSchemaIdempotent(t *testing.T) {
	// calling migrateSchema multiple times should not error
	_, err := db.New(":memory:")
	assert.NoError(t, err)
	_, err = db.New(":memory:")
	assert.NoError(t, err)
}

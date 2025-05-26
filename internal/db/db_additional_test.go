package db_test

import (
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// openTestDB returns a fresh in-memory DB with schema applied
func openTestDB(t *testing.T) *sqlx.DB {
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

	// Register cleanup to ensure connection is closed after test
	t.Cleanup(func() {
		if err := dbInstance.Close(); err != nil {
			t.Logf("Error closing database connection: %v", err)
		} else {
			t.Logf("Database connection closed successfully")
		}
	})

	return dbInstance.DB
}

func TestFetchArticleByID_NotFound(t *testing.T) {
	dbConn := openTestDB(t)
	_, err := db.FetchArticleByID(dbConn, 999)
	assert.ErrorIs(t, err, db.ErrArticleNotFound)
}

func TestArticleExistsBySimilarTitle(t *testing.T) {
	dbConn := openTestDB(t)
	// Insert articles with varying titles
	titles := []string{
		"Hello, World!",
		"Go-Programming",
		"Test Case",
	}
	for _, title := range titles {
		_, err := db.InsertArticle(dbConn, &db.Article{
			Source:    "s",
			PubDate:   time.Now(),
			URL:       "u" + title,
			Title:     title,
			Content:   "c",
			CreatedAt: time.Now(),
		})
		assert.NoError(t, err)
	}
	// Exact match
	exact, err := db.ArticleExistsBySimilarTitle(dbConn, "hello world")
	assert.NoError(t, err)
	assert.True(t, exact)
	// Partial match
	part, err := db.ArticleExistsBySimilarTitle(dbConn, "programming")
	assert.NoError(t, err)
	assert.True(t, part)
	// No match
	none, err := db.ArticleExistsBySimilarTitle(dbConn, "unknown")
	assert.NoError(t, err)
	assert.False(t, none)
}

func TestInsertLabelAndFeedback(t *testing.T) {
	dbConn := openTestDB(t)
	// Test InsertLabel
	label := &db.Label{
		Data:        "data1",
		Label:       "lbl",
		Source:      "src",
		DateLabeled: time.Now(),
		Labeler:     "u1",
		Confidence:  0.75,
		CreatedAt:   time.Now(),
	}
	err := db.InsertLabel(dbConn, label)
	assert.NoError(t, err)
	assert.Greater(t, label.ID, int64(0))

	// Test InsertFeedback
	feedback := &db.Feedback{
		ArticleID:        0, // allow zero for test
		UserID:           "user1",
		FeedbackText:     "fb",
		Category:         "agree",
		EnsembleOutputID: nil,
		Source:           "src",
		CreatedAt:        time.Now(),
	}
	err = db.InsertFeedback(dbConn, feedback)
	assert.NoError(t, err)
	assert.Greater(t, feedback.ID, int64(0))
}

func TestUpdateArticleScoreLLM(t *testing.T) {
	dbConn := openTestDB(t)

	// Insert article
	article := &db.Article{
		Source:    "x",
		PubDate:   time.Now(),
		URL:       "u5",
		Title:     "t5",
		Content:   "c5",
		CreatedAt: time.Now(),
	}
	id, err := db.InsertArticle(dbConn, article)
	assert.NoError(t, err)

	// Debug log for inserted article ID
	t.Logf("Inserted article ID: %d", id)

	// Use ExecContext on UpdateArticleScoreLLM
	exec := dbConn
	err = db.UpdateArticleScoreLLM(exec, id, 0.33, 0.66)
	assert.NoError(t, err)

	// Debug log for updated article ID
	t.Logf("Updated article ID: %d", id)

	// Verify via FetchArticleByID
	fetched, err := db.FetchArticleByID(dbConn, id)
	assert.NoError(t, err)
	assert.NotNil(t, fetched.CompositeScore)
	assert.Equal(t, 0.33, *fetched.CompositeScore)
	assert.NotNil(t, fetched.Confidence)
	assert.Equal(t, 0.66, *fetched.Confidence)
}

package db

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	testDBFile := filepath.Join(t.TempDir(), "test.db")
	dbInstance, err := New(testDBFile)
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
			score_source TEXT
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

func TestInsertDuplicateArticle(t *testing.T) {
	dbConn := setupTestDB(t)
	t.Cleanup(func() {
		dbConn.Close()
	})

	url := "https://example.com/test-article-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	article1 := &Article{
		Source:  "test",
		PubDate: time.Now().UTC().Truncate(time.Second),
		URL:     url,
		Title:   "Test Article",
		Content: "This is a test article.",
	}

	_, err := InsertArticle(dbConn, article1)
	if err != nil {
		t.Fatalf("Failed to insert first article: %v", err)
	}

	article2 := &Article{
		Source:  "test",
		PubDate: time.Now().UTC().Truncate(time.Second),
		URL:     url,
		Title:   "Another Test Article",
		Content: "This is another test article.",
	}

	_, err = InsertArticle(dbConn, article2)
	if err == nil {
		t.Error("Expected error when inserting duplicate URL, got nil")
	}
}

func TestArticlePagination(t *testing.T) {
	dbConn := setupTestDB(t)
	t.Cleanup(func() {
		dbConn.Close()
	})

	_, err := FetchArticles(dbConn, "test", "", 10, 0)
	if err != nil {
		t.Errorf("FetchArticles with basic filter failed: %v", err)
	}
}

func TestInsertAndFetchLLMScore(t *testing.T) {
	dbConn := setupTestDB(t)
	// Ensure the database connection is closed after the test
	defer dbConn.Close()

	article := &Article{
		Source:  "Test Source",
		PubDate: time.Now(),
		URL:     "http://example.com/test2",
		Title:   "Test Title 2",
		Content: "Test Content 2",
	}

	articleID, err := InsertArticle(dbConn, article)
	if err != nil {
		t.Fatalf("InsertArticle failed: %v", err)
	}

	score := &LLMScore{
		ArticleID: articleID,
		Model:     "left",
		Score:     0.5,
		Metadata:  "{}",
		CreatedAt: time.Now(),
	}

	_, err = InsertLLMScore(dbConn, score)
	if err != nil {
		t.Fatalf("InsertLLMScore failed: %v", err)
	}

	scores, err := FetchLLMScores(dbConn, articleID)
	if err != nil {
		t.Fatalf("FetchLLMScores failed: %v", err)
	}

	if len(scores) == 0 || scores[0].ArticleID != articleID {
		t.Errorf("Inserted LLM score not found")
	}
}

func TestArticleWithNullFields(t *testing.T) {
	dbConn := setupTestDB(t)
	t.Cleanup(func() {
		dbConn.Close()
	})

	// Create an article with null score_source
	article := &Article{
		Source:  "Test Source",
		PubDate: time.Now(),
		URL:     "http://example.com/test-null",
		Title:   "Test Title Null",
		Content: "Test Content Null",
		// Explicitly not setting ScoreSource to test NULL handling
	}

	articleID, err := InsertArticle(dbConn, article)
	if err != nil {
		t.Fatalf("InsertArticle failed: %v", err)
	}

	// Fetch the article back
	articles, err := FetchArticles(dbConn, "", "", 10, 0)
	if err != nil {
		t.Fatalf("FetchArticles failed: %v", err)
	}

	if len(articles) == 0 {
		t.Fatal("No articles returned")
	}

	// Check if we can read the article with NULL score_source
	found := false
	for _, a := range articles {
		if a.ID == articleID {
			found = true
			// score_source should be nil when NULL
			if a.ScoreSource != nil {
				t.Errorf("Expected nil score_source for NULL value, got %q", *a.ScoreSource)
			}
		}
	}

	if !found {
		t.Error("Inserted article not found in results")
	}
}

func TestTransactionRollback(t *testing.T) {
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	tx, err := dbConn.Beginx()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	article := &Article{
		Source:  "Rollback Source",
		PubDate: time.Now(),
		URL:     "http://example.com/rollback",
		Title:   "Rollback Title",
		Content: "Rollback Content",
	}

	_, err = tx.NamedExec(`INSERT INTO articles (source, pub_date, url, title, content) VALUES (:source, :pub_date, :url, :title, :content)`, article)
	if err != nil {
		t.Fatalf("Failed to insert article in tx: %v", err)
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Should not find the article after rollback
	articles, err := FetchArticles(dbConn, "Rollback Source", "", 10, 0)
	if err != nil {
		t.Fatalf("FetchArticles failed: %v", err)
	}
	for _, a := range articles {
		if a.URL == "http://example.com/rollback" {
			t.Error("Article should not exist after rollback")
		}
	}
}

func TestConcurrentInserts(t *testing.T) {
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	n := 5
	done := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			article := &Article{
				Source:  "Concurrent",
				PubDate: time.Now(),
				URL:     "http://example.com/concurrent-" + strconv.Itoa(idx),
				Title:   "Concurrent Title",
				Content: "Concurrent Content",
			}
			_, err := InsertArticle(dbConn, article)
			done <- err == nil
		}(i)
	}
	count := 0
	for i := 0; i < n; i++ {
		if <-done {
			count++
		}
	}
	if count != n {
		t.Errorf("Expected %d successful inserts, got %d", n, count)
	}
}

func TestInsertAndFetchArticle(t *testing.T) {
	dbConn := setupTestDB(t)
	article := &Article{
		Source:    "test",
		PubDate:   time.Now().UTC(),
		URL:       "http://example.com/1",
		Title:     "Title 1",
		Content:   "Content",
		CreatedAt: time.Now().UTC(),
	}
	id, err := InsertArticle(dbConn, article)
	assert.NoError(t, err)
	assert.Greater(t, id, int64(0))

	fetched, err := FetchArticleByID(dbConn, id)
	assert.NoError(t, err)
	assert.Equal(t, article.URL, fetched.URL)
	assert.Equal(t, article.Title, fetched.Title)
}

func TestArticleExistsByURL(t *testing.T) {
	dbConn := setupTestDB(t)
	exists, err := ArticleExistsByURL(dbConn, "http://nope")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Insert one and test exists
	article := &Article{Source: "test", PubDate: time.Now(), URL: "http://test", Title: "t", Content: "c", CreatedAt: time.Now()}
	_, err = InsertArticle(dbConn, article)
	assert.NoError(t, err)
	exists, err = ArticleExistsByURL(dbConn, "http://test")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestInsertAndFetchLLMScores(t *testing.T) {
	dbConn := setupTestDB(t)
	// insert article first
	article := &Article{Source: "src", PubDate: time.Now(), URL: "u2", Title: "t2", Content: "c2", CreatedAt: time.Now()}
	artID, err := InsertArticle(dbConn, article)
	assert.NoError(t, err)

	score := &LLMScore{ArticleID: artID, Model: "left", Score: 0.5, Metadata: "meta", CreatedAt: time.Now()}
	sid, err := InsertLLMScore(dbConn, score)
	assert.NoError(t, err)
	assert.Greater(t, sid, int64(0))

	scores, err := FetchLLMScores(dbConn, artID)
	assert.NoError(t, err)
	assert.Len(t, scores, 1)
	assert.Equal(t, "left", scores[0].Model)
}

func TestUpdateArticleScoreAndFetchConfidence(t *testing.T) {
	dbConn := setupTestDB(t)
	// insert article
	article := &Article{Source: "src", PubDate: time.Now(), URL: "u3", Title: "t3", Content: "c3", CreatedAt: time.Now()}
	artID, err := InsertArticle(dbConn, article)
	assert.NoError(t, err)

	err = UpdateArticleScore(dbConn, artID, 1.23, 0.45)
	assert.NoError(t, err)

	conf, err := FetchLatestConfidence(dbConn, artID)
	assert.NoError(t, err)
	assert.InDelta(t, 0.45, conf, 1e-6)
}

func TestFetchLatestEnsembleScore(t *testing.T) {
	dbConn := setupTestDB(t)
	// insert article
	article := &Article{Source: "s", PubDate: time.Now(), URL: "u4", Title: "t4", Content: "c4", CreatedAt: time.Now()}
	artID, err := InsertArticle(dbConn, article)
	assert.NoError(t, err)

	// no ensemble score yet
	s, err := FetchLatestEnsembleScore(dbConn, artID)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, s)

	// insert ensemble
	es := &LLMScore{ArticleID: artID, Model: "ensemble", Score: 2.5, Metadata: "{}", CreatedAt: time.Now()}
	_, err = InsertLLMScore(dbConn, es)
	assert.NoError(t, err)

	s2, err := FetchLatestEnsembleScore(dbConn, artID)
	assert.NoError(t, err)
	assert.Equal(t, 2.5, s2)
}

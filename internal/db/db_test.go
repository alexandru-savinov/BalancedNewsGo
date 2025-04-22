package db

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

const testDBFile = "test.db"

func setupTestDB(t *testing.T) *sqlx.DB {
	if err := os.Remove(testDBFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to remove test DB file: %v", err)
	}

	dbConn, err := InitDB(testDBFile)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	return dbConn
}

func TestInsertDuplicateArticle(t *testing.T) {
	dbConn := setupTestDB(t)
	t.Cleanup(func() {
		dbConn.Close()
		if err := os.Remove(testDBFile); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: failed to remove test DB file: %v", err)
		}
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
		if err := os.Remove(testDBFile); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: failed to remove test DB file: %v", err)
		}
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

	// Defer the removal of the test database file
	defer func() {
		if err := os.Remove(testDBFile); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: failed to remove test DB file: %v", err)
		}
	}()

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
		if err := os.Remove(testDBFile); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: failed to remove test DB file: %v", err)
		}
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
	defer func() {
		_ = os.Remove(testDBFile)
	}()

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
	defer func() {
		_ = os.Remove(testDBFile)
	}()

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

package db

import (
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

const testDBFile = "test.db"

func setupTestDB(t *testing.T) *sqlx.DB {
	os.Remove(testDBFile)
	dbConn, err := InitDB(testDBFile)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}
	return dbConn
}

func TestInsertAndFetchArticle(t *testing.T) {
	dbConn := setupTestDB(t)
	defer os.Remove(testDBFile)

	article := &Article{
		Source:  "Test Source",
		PubDate: time.Now(),
		URL:     "http://example.com/test",
		Title:   "Test Title",
		Content: "Test Content",
	}

	id, err := InsertArticle(dbConn, article)
	if err != nil {
		t.Fatalf("InsertArticle failed: %v", err)
	}

	articles, err := FetchArticles(dbConn, "", "", 10, 0)
	if err != nil {
		t.Fatalf("FetchArticles failed: %v", err)
	}
	if len(articles) == 0 || articles[0].ID != id {
		t.Errorf("Inserted article not found")
	}
}

func TestInsertAndFetchLLMScore(t *testing.T) {
	dbConn := setupTestDB(t)
	defer os.Remove(testDBFile)

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

package db

import (
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// helper to open a minimal in-memory DB
func openStatusTestDB(t *testing.T) *sqlx.DB {
	db, err := InitDB(":memory:")
	assert.NoError(t, err)
	return db
}

func TestUpdateArticleStatus(t *testing.T) {
	db := openStatusTestDB(t)
	defer db.Close()

	article := &Article{
		Source:  "src",
		PubDate: time.Now(),
		URL:     "u1",
		Title:   "t1",
		Content: "c1",
	}
	id, err := InsertArticle(db, article)
	assert.NoError(t, err)

	// update status and verify
	err = UpdateArticleStatus(db, id, "processed")
	assert.NoError(t, err)

	fetched, err := FetchArticleByID(db, id)
	assert.NoError(t, err)
	assert.NotNil(t, fetched.Status)
	assert.Equal(t, "processed", *fetched.Status)

	// call with non existing id should not error
	err = UpdateArticleStatus(db, id+9999, "none")
	assert.NoError(t, err)
}

func TestWithRetry(t *testing.T) {
	attempts := 0
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond}

	err := WithRetry(cfg, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("database is locked")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)

	attempts = 0
	err = WithRetry(cfg, func() error {
		attempts++
		return errors.New("database is locked")
	})

	assert.Error(t, err)
	assert.Equal(t, cfg.MaxAttempts, attempts)
}

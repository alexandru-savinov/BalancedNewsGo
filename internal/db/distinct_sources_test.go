package db_test

import (
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestFetchDistinctSources(t *testing.T) {
	dbConn := openTestDB(t)

	// Empty DB should return empty slice
	sources, err := db.FetchDistinctSources(dbConn)
	assert.NoError(t, err)
	assert.Empty(t, sources)

	// Insert some articles with duplicate and empty sources
	articles := []db.Article{
		{Source: "CNN", PubDate: time.Now(), URL: "http://cnn.com/1", Title: "t1", Content: "c1", CreatedAt: time.Now()},
		{Source: "BBC", PubDate: time.Now(), URL: "http://bbc.com/1", Title: "t2", Content: "c2", CreatedAt: time.Now()},
		{Source: "CNN", PubDate: time.Now(), URL: "http://cnn.com/2", Title: "t3", Content: "c3", CreatedAt: time.Now()},
		{Source: "", PubDate: time.Now(), URL: "http://empty.com", Title: "t4", Content: "c4", CreatedAt: time.Now()},
	}
	for i := range articles {
		_, err := db.InsertArticle(dbConn, &articles[i])
		assert.NoError(t, err)
	}

	sources, err = db.FetchDistinctSources(dbConn)
	assert.NoError(t, err)
	assert.Equal(t, []string{"BBC", "CNN"}, sources)
}

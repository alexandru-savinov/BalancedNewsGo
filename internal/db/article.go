package db

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// Article represents an article in the database
type Article struct {
	ID             int64          `db:"id" json:"id"`
	Source         string         `db:"source" json:"source"`
	PubDate        time.Time      `db:"pub_date" json:"pub_date"`
	URL            string         `db:"url" json:"url"`
	Title          string         `db:"title" json:"title"`
	Content        string         `db:"content" json:"content"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	CompositeScore *float64       `db:"composite_score" json:"composite_score"`
	Confidence     *float64       `db:"confidence" json:"confidence"`
	ScoreSource    sql.NullString `db:"score_source" json:"score_source"`
	Status         string         `db:"status" json:"status"`
	FailCount      int            `db:"fail_count" json:"fail_count"`
	LastAttempt    *time.Time     `db:"last_attempt" json:"last_attempt"`
	Escalated      int            `db:"escalated" json:"escalated"`
}

// FetchArticleByID retrieves an article by its ID
func FetchArticleByID(db *sqlx.DB, id int64) (*Article, error) {
	var article Article
	err := db.Get(&article, "SELECT * FROM articles WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrArticleNotFound
		}
		return nil, handleError(err, "failed to fetch article")
	}
	return &article, nil
}

// FetchArticles retrieves articles with optional filters
func FetchArticles(db *sqlx.DB, source string, leaning string, limit int, offset int) ([]Article, error) {
	query := "SELECT * FROM articles"
	var conditions []string
	var args []interface{}

	if source != "" && source != "all" {
		conditions = append(conditions, "source = ?")
		args = append(args, source)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, cond := range conditions[1:] {
			query += " AND " + cond
		}
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	var articles []Article
	err := db.Select(&articles, query, args...)
	if err != nil {
		return nil, handleError(err, "failed to fetch articles")
	}
	return articles, nil
}

// InsertArticle creates a new article in the database
func InsertArticle(db *sqlx.DB, article *Article) (int64, error) {
	// Check if article with URL already exists
	exists, err := ArticleExistsByURL(db, article.URL)
	if err != nil {
		return 0, handleError(err, "failed to check article existence")
	}
	if exists {
		return 0, ErrDuplicateArticle
	}

	result, err := db.NamedExec(`
		INSERT INTO articles (
			source, pub_date, url, title, content, created_at,
			composite_score, confidence, score_source, status,
			fail_count, last_attempt, escalated
		) VALUES (
			:source, :pub_date, :url, :title, :content, :created_at,
			:composite_score, :confidence, :score_source, :status,
			:fail_count, :last_attempt, :escalated
		)`,
		article)
	if err != nil {
		return 0, handleError(err, "failed to insert article")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, handleError(err, "failed to get inserted article ID")
	}

	return id, nil
}

// UpdateArticleScore updates an article's composite score and confidence
func UpdateArticleScore(db *sqlx.DB, articleID int64, score float64, confidence float64) error {
	result, err := db.Exec(`
		UPDATE articles 
		SET composite_score = ?, confidence = ?, score_source = 'llm'
		WHERE id = ?`,
		score, confidence, articleID)
	if err != nil {
		return handleError(err, "failed to update article score")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return ErrArticleNotFound
	}

	return nil
}

// ArticleExistsByURL checks if an article with the given URL exists
func ArticleExistsByURL(db *sqlx.DB, url string) (bool, error) {
	var exists bool
	err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM articles WHERE url = ?)", url)
	if err != nil {
		return false, handleError(err, "failed to check article URL existence")
	}
	return exists, nil
}

// DeleteArticle removes an article and its associated data
func DeleteArticle(db *sqlx.DB, id int64) error {
	tx, err := db.Beginx()
	if err != nil {
		return handleError(err, "failed to begin transaction")
	}

	// Delete associated LLM scores first
	_, err = tx.Exec("DELETE FROM llm_scores WHERE article_id = ?", id)
	if err != nil {
		tx.Rollback()
		return handleError(err, "failed to delete article scores")
	}

	// Delete the article
	result, err := tx.Exec("DELETE FROM articles WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return handleError(err, "failed to delete article")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return handleError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		tx.Rollback()
		return ErrArticleNotFound
	}

	if err := tx.Commit(); err != nil {
		return handleError(err, "failed to commit transaction")
	}

	return nil
}

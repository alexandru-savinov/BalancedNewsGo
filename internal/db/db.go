package db

import (
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Article struct {
	ID        int64     `db:"id"`
	Source    string    `db:"source"`
	PubDate   time.Time `db:"pub_date"`
	URL       string    `db:"url"`
	Title     string    `db:"title"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
}

type LLMScore struct {
	ID        int64     `db:"id"`
	ArticleID int64     `db:"article_id"`
	Model     string    `db:"model"`
	Score     float64   `db:"score"`
	Metadata  string    `db:"metadata"`
	CreatedAt time.Time `db:"created_at"`
}
type Feedback struct {
	ID           int64     `db:"id"`
	ArticleID    int64     `db:"article_id"`
	UserID       string    `db:"user_id"`
	FeedbackText string    `db:"feedback_text"`
	CreatedAt    time.Time `db:"created_at"`
}

func InitDB(dbPath string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	schema := `
CREATE TABLE IF NOT EXISTS articles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source TEXT,
	pub_date DATETIME,
	url TEXT UNIQUE,
	title TEXT,
	content TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS llm_scores (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	article_id INTEGER,
	model TEXT,
	score REAL,
	metadata TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(article_id) REFERENCES articles(id)
);

CREATE TABLE IF NOT EXISTS feedback (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	article_id INTEGER,
	user_id TEXT,
	feedback_text TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(article_id) REFERENCES articles(id)
);
`
	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertArticle(db *sqlx.DB, article *Article) (int64, error) {
	res, err := db.NamedExec(`INSERT INTO articles (source, pub_date, url, title, content) 
	VALUES (:source, :pub_date, :url, :title, :content)`, article)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func InsertFeedback(db *sqlx.DB, feedback *Feedback) (int64, error) {
	res, err := db.NamedExec(`INSERT INTO feedback (article_id, user_id, feedback_text)
		VALUES (:article_id, :user_id, :feedback_text)`, feedback)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ArticleExistsByURL(db *sqlx.DB, url string) (bool, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(1) FROM articles WHERE url = ?", url)
	return count > 0, err
}

func InsertLLMScore(db *sqlx.DB, score *LLMScore) (int64, error) {
	res, err := db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata) 
	VALUES (:article_id, :model, :score, :metadata)`, score)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FetchArticles with optional filters and pagination
func FetchArticles(db *sqlx.DB, source, leaning string, limit, offset int) ([]Article, error) {
	query := "SELECT * FROM articles WHERE 1=1"
	args := []interface{}{}

	if source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}

	// For MVP, leaning filter is ignored or can be implemented via join with llm_scores
	query += " ORDER BY pub_date DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	var articles []Article
	err := db.Select(&articles, query, args...)
	return articles, err
}

// FetchArticleByID returns a single article by ID
func FetchArticleByID(db *sqlx.DB, id int64) (Article, error) {
	var article Article
	err := db.Get(&article, "SELECT * FROM articles WHERE id = ?", id)
	return article, err
}

// FetchLLMScores returns all LLM scores for an article
func FetchLLMScores(db *sqlx.DB, articleID int64) ([]LLMScore, error) {
	var scores []LLMScore
	err := db.Select(&scores, "SELECT * FROM llm_scores WHERE article_id = ?", articleID)
	return scores, err
}

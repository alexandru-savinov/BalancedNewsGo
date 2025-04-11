package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Article struct {
	ID             int64     `db:"id"`
	Source         string    `db:"source"`
	PubDate        time.Time `db:"pub_date"`
	URL            string    `db:"url"`
	Title          string    `db:"title"`
	Content        string    `db:"content"`
	CreatedAt      time.Time `db:"created_at"`
	CompositeScore float64   `db:"-"`

	Status      string     `db:"status"`
	FailCount   int        `db:"fail_count"`
	LastAttempt *time.Time `db:"last_attempt"`
	Escalated   bool       `db:"escalated"`
}

type LLMScore struct {
	ID        int64     `db:"id"`
	ArticleID int64     `db:"article_id"`
	Model     string    `db:"model"`
	Score     float64   `db:"score"`
	Metadata  string    `db:"metadata"`
	Version   int       `db:"version"`
	CreatedAt time.Time `db:"created_at"`
}
type Feedback struct {
	ID               int64     `db:"id"`
	ArticleID        int64     `db:"article_id"`
	UserID           string    `db:"user_id"`
	FeedbackText     string    `db:"feedback_text"`
	Category         string    `db:"category"`           // agree, disagree, unclear, other
	EnsembleOutputID *int64    `db:"ensemble_output_id"` // optional, nullable
	Source           string    `db:"source"`             // form, email, api
	CreatedAt        time.Time `db:"created_at"`
}

type Label struct {
	ID          int64     `db:"id"`
	Data        string    `db:"data"`
	Label       string    `db:"label"`
	Source      string    `db:"source"`
	DateLabeled time.Time `db:"date_labeled"`
	Labeler     string    `db:"labeler"`
	Confidence  float64   `db:"confidence"`
	CreatedAt   time.Time `db:"created_at"`
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
	version INTEGER DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(article_id) REFERENCES articles(id)
);

CREATE INDEX IF NOT EXISTS idx_llm_scores_article_version ON llm_scores(article_id, version);

CREATE TABLE IF NOT EXISTS feedback (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	article_id INTEGER,
	user_id TEXT,
	feedback_text TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(article_id) REFERENCES articles(id)
);
CREATE TABLE IF NOT EXISTS labels (
   id INTEGER PRIMARY KEY AUTOINCREMENT,
   data TEXT,
   label TEXT,
   source TEXT,
   date_labeled DATETIME,
   labeler TEXT,
   confidence REAL,
   created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	// Migrate: add columns for failure tracking if they don't exist
	alterStatements := []string{
		"ALTER TABLE articles ADD COLUMN status TEXT DEFAULT 'pending';",
		"ALTER TABLE articles ADD COLUMN fail_count INTEGER DEFAULT 0;",
		"ALTER TABLE articles ADD COLUMN last_attempt DATETIME;",
		"ALTER TABLE articles ADD COLUMN escalated BOOLEAN DEFAULT 0;",
	}

	for _, stmt := range alterStatements {
		_, err := db.Exec(stmt)
		if err != nil && !isDuplicateColumnError(err) {
			fmt.Printf("DB migration error: %v\n", err)
		}
	}

	return db, nil
}

// isDuplicateColumnError returns true if the error is due to an existing column
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate column name") || strings.Contains(msg, "already exists")
}

func InsertArticle(db *sqlx.DB, article *Article) (int64, error) {
	res, err := db.NamedExec(`INSERT INTO articles (source, pub_date, url, title, content) 
	VALUES (:source, :pub_date, :url, :title, :content)`, article)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func InsertLabel(db *sqlx.DB, label *Label) (int64, error) {
	res, err := db.NamedExec(`INSERT INTO labels (data, label, source, date_labeled, labeler, confidence)
		VALUES (:data, :label, :source, :date_labeled, :labeler, :confidence)`, label)
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

func ArticleExistsBySimilarTitle(db *sqlx.DB, title string, days int) (bool, error) {
	var count int
	query := `
		SELECT COUNT(1) FROM articles
		WHERE pub_date >= datetime('now', ?)
		  AND LOWER(title) LIKE '%' || LOWER(?) || '%'
	`
	interval := fmt.Sprintf("-%d days", days)
	err := db.Get(&count, query, interval, title)
	return count > 0, err
}

func InsertLLMScore(db *sqlx.DB, score *LLMScore) (int64, error) {
	res, err := db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, version)
	VALUES (:article_id, :model, :score, :metadata, :version)`, score)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func UpdateArticleScore(db *sqlx.DB, articleID int64, score float64, confidence float64) error {
	_, err := db.Exec(`UPDATE articles SET score = ?, confidence = ? WHERE id = ?`, score, confidence, articleID)
	return err
}

// FetchArticles with optional filters and pagination.
func FetchArticles(db *sqlx.DB, source, leaning string, limit, offset int) ([]Article, error) {
	query := "SELECT * FROM articles WHERE 1=1"
	args := make([]interface{}, 0, 3)

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

// FetchArticleByID returns a single article by ID.
func FetchArticleByID(db *sqlx.DB, id int64) (Article, error) {
	var article Article
	err := db.Get(&article, "SELECT * FROM articles WHERE id = ?", id)

	return article, err
}

// FetchLLMScores returns all LLM scores for an article.
func FetchLLMScores(db *sqlx.DB, articleID int64) ([]LLMScore, error) {
	var scores []LLMScore
	err := db.Select(&scores, "SELECT * FROM llm_scores WHERE article_id = ? ORDER BY version DESC", articleID)

	return scores, err
}

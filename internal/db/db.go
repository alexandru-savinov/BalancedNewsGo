package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var ErrArticleNotFound = errors.New("db: article not found")

type Article struct {
	ID             int64          `db:"id" json:"id"`
	Source         string         `db:"source" json:"source"`
	PubDate        time.Time      `db:"pub_date" json:"pub_date"`
	URL            string         `db:"url" json:"url"`
	Title          string         `db:"title" json:"title"`
	Content        string         `db:"content" json:"content"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	CompositeScore *float64       `db:"composite_score" json:"composite_score"` // Use pointer for nullable
	Confidence     *float64       `db:"confidence" json:"confidence"`           // Use pointer for nullable
	ScoreSource    sql.NullString `db:"score_source" json:"score_source"`       // "llm" or "manual", now nullable

	Status      string     `db:"status" json:"status"`
	FailCount   int        `db:"fail_count" json:"fail_count"`
	LastAttempt *time.Time `db:"last_attempt" json:"last_attempt"`
	Escalated   bool       `db:"escalated" json:"escalated"`
}

// Custom JSON marshaling for Article to handle sql.NullString for ScoreSource
func (a Article) MarshalJSON() ([]byte, error) {
	type Alias Article
	return json.Marshal(&struct {
		ScoreSource string `json:"score_source"`
		*Alias
	}{
		ScoreSource: func() string {
			if a.ScoreSource.Valid {
				return a.ScoreSource.String
			}
			return ""
		}(),
		Alias: (*Alias)(&a),
	})
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
	composite_score REAL, -- ADDED
	confidence REAL,      -- ADDED
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
	category TEXT,
	ensemble_output_id INTEGER,
	source TEXT,
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
		"ALTER TABLE articles ADD COLUMN composite_score REAL;",       // ADDED Migration
		"ALTER TABLE articles ADD COLUMN confidence REAL;",            // ADDED Migration
		"ALTER TABLE articles ADD COLUMN score_source TEXT;",          // ADDED Migration for score_source
		"ALTER TABLE feedback ADD COLUMN category TEXT;",              // ADDED Migration for feedback category
		"ALTER TABLE feedback ADD COLUMN source TEXT;",                // ADDED Migration for feedback source
		"ALTER TABLE feedback ADD COLUMN ensemble_output_id INTEGER;", // ADDED Migration for feedback ensemble_output_id
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
	res, err := db.NamedExec(`INSERT INTO articles (source, pub_date, url, title, content, composite_score, confidence, score_source)
	VALUES (:source, :pub_date, :url, :title, :content, :composite_score, :confidence, :score_source)`, article)
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
	res, err := db.NamedExec(`INSERT INTO feedback (
		article_id, 
		user_id, 
		feedback_text, 
		category, 
		ensemble_output_id, 
		source,
		created_at
	) VALUES (
		:article_id, 
		:user_id, 
		:feedback_text, 
		:category, 
		:ensemble_output_id, 
		:source,
		CURRENT_TIMESTAMP
	)`, feedback)
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
	_, err := db.Exec(`UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'manual' WHERE id = ?`, score, confidence, articleID)
	return err
}

// UpdateArticleScoreLLM sets the score and score_source to 'llm'
func UpdateArticleScoreLLM(db *sqlx.DB, articleID int64, score float64, confidence float64) error {
	_, err := db.Exec(`UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'llm' WHERE id = ?`, score, confidence, articleID)
	return err
}

// FetchArticles with optional filters and pagination.
func FetchArticles(db *sqlx.DB, source, leaning string, limit, offset int) ([]Article, error) {
	query := "SELECT id, source, pub_date, url, title, content, created_at, composite_score, confidence, COALESCE(score_source, '') AS score_source, status, fail_count, last_attempt, escalated FROM articles WHERE 1=1"
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
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Article{}, ErrArticleNotFound
		}
		return Article{}, err
	}
	return article, nil
}

// FetchLLMScores returns all LLM scores for an article.
func FetchLLMScores(db *sqlx.DB, articleID int64) ([]LLMScore, error) {
	var scores []LLMScore
	err := db.Select(&scores, "SELECT * FROM llm_scores WHERE article_id = ? ORDER BY version DESC", articleID)

	return scores, err
}

// FetchLatestEnsembleScore returns the score value of the most recent 'ensemble' score record for an article.
// Returns 0.0 and nil error if no ensemble score is found.
func FetchLatestEnsembleScore(db *sqlx.DB, articleID int64) (float64, error) {
	var score float64
	// Query for the score field of the latest record matching article_id and model='ensemble'
	err := db.Get(&score, "SELECT score FROM llm_scores WHERE article_id = ? AND model = 'ensemble' ORDER BY created_at DESC LIMIT 1", articleID)
	if err != nil {
		// If no rows are found, it's not a fatal error, just means no ensemble score exists yet.
		if err.Error() == "sql: no rows in result set" { // Check specifically for no rows error
			return 0.0, nil // Return 0.0 score, no error
		}
		// For other potential errors (DB connection issues, etc.), return the error.
		return 0.0, err
	}
	return score, nil
}

// FetchLatestConfidence returns the confidence value from metadata of the most recent 'ensemble' score record for an article.
// Returns 0.0 and nil error if no ensemble score is found or if metadata doesn't contain confidence.
func FetchLatestConfidence(db *sqlx.DB, articleID int64) (float64, error) {
	var metadata string
	// Query for the metadata field of the latest record matching article_id and model='ensemble'
	err := db.Get(&metadata, "SELECT metadata FROM llm_scores WHERE article_id = ? AND model = 'ensemble' ORDER BY created_at DESC LIMIT 1", articleID)
	if err != nil {
		// If no rows are found, it's not a fatal error, just means no ensemble score exists yet.
		if err.Error() == "sql: no rows in result set" { // Check specifically for no rows error
			return 0.0, nil // Return 0.0 confidence, no error
		}
		// For other potential errors (DB connection issues, etc.), return the error.
		return 0.0, err
	}

	// Parse metadata JSON to extract confidence
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
		// If metadata can't be parsed, return a default confidence but no error
		return 0.0, nil
	}

	// Try to extract confidence from metadata
	if conf, ok := meta["confidence"].(float64); ok {
		return conf, nil
	}

	// If confidence not found in metadata, return default
	return 0.0, nil
}

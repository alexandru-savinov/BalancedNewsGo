package db

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"context"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Error codes
const (
	ErrDBConnection = "db_connection_error"
	ErrDBQuery      = "db_query_error"
	ErrDBConstraint = "db_constraint_error"
	ErrDBMigration  = "db_migration_error"
	ErrInvalidInput = "db_invalid_input"
)

// Pre-defined database errors
var (
	ErrDuplicateURL    = apperrors.New("Article with this URL already exists", "conflict")
	ErrInvalidScore    = apperrors.New("Invalid score value", "validation_error")
	ErrArticleNotFound = errors.New("not found")
)

type Article struct {
	ID             int64     `db:"id"`
	Source         string    `db:"source"`
	PubDate        time.Time `db:"pub_date"`
	URL            string    `db:"url"`
	Title          string    `db:"title"`
	Content        string    `db:"content"`
	CompositeScore *float64  `db:"composite_score"`
	Confidence     *float64  `db:"confidence"`
	CreatedAt      time.Time `db:"created_at"`
	Status         string    `db:"status"`
	FailCount      int       `db:"fail_count"`
	LastAttempt    time.Time `db:"last_attempt"`
	Escalated      bool      `db:"escalated"`
	ScoreSource    string    `db:"score_source"`
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
	Category         string    `db:"category"`
	EnsembleOutputID *int64    `db:"ensemble_output_id"`
	Source           string    `db:"source"`
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

// InitDB initializes the database with all required tables
func InitDB(dbPath string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, apperrors.New(ErrDBConnection, "Failed to open database")
	}

	if err := db.Ping(); err != nil {
		return nil, apperrors.New(ErrDBConnection, "Failed to connect to database")
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	if err := migrateSchema(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sqlx.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS articles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source TEXT,
	pub_date DATETIME,
	url TEXT UNIQUE,
	title TEXT,
	content TEXT,
	composite_score REAL,
	confidence REAL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	status TEXT DEFAULT 'pending',
	fail_count INTEGER DEFAULT 0,
	last_attempt DATETIME,
	escalated BOOLEAN DEFAULT 0,
	score_source TEXT
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
	_, err := db.Exec(schema)
	if err != nil {
		return apperrors.New(ErrDBMigration, "Failed to create tables")
	}
	return nil
}

func migrateSchema(db *sqlx.DB) error {
	alterStatements := []string{
		"ALTER TABLE articles ADD COLUMN status TEXT DEFAULT 'pending';",
		"ALTER TABLE articles ADD COLUMN fail_count INTEGER DEFAULT 0;",
		"ALTER TABLE articles ADD COLUMN last_attempt DATETIME;",
		"ALTER TABLE articles ADD COLUMN escalated BOOLEAN DEFAULT 0;",
		"ALTER TABLE articles ADD COLUMN composite_score REAL;",
		"ALTER TABLE articles ADD COLUMN confidence REAL;",
		"ALTER TABLE articles ADD COLUMN score_source TEXT;",
		"ALTER TABLE feedback ADD COLUMN category TEXT;",
		"ALTER TABLE feedback ADD COLUMN source TEXT;",
		"ALTER TABLE feedback ADD COLUMN ensemble_output_id INTEGER;",
	}

	for _, stmt := range alterStatements {
		_, err := db.Exec(stmt)
		if err != nil && !isDuplicateColumnError(err) {
			return apperrors.New(ErrDBMigration, "Failed to migrate schema")
		}
	}
	return nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate column name") || strings.Contains(msg, "already exists")
}

// handleError is a helper to wrap database errors with appropriate context
func handleError(err error, msg string) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	switch {
	case err == sql.ErrNoRows:
		return apperrors.New("not_found", msg)
	case strings.Contains(errMsg, "UNIQUE constraint"):
		return apperrors.New("conflict", msg)
	case strings.Contains(errMsg, "FOREIGN KEY constraint"):
		return apperrors.New("foreign_key_violation", msg)
	default:
		return apperrors.New("internal", msg)
	}
}

// InsertLabel inserts a new label record
func InsertLabel(db *sqlx.DB, label *Label) error {
	result, err := db.NamedExec(`
        INSERT INTO labels (data, label, source, date_labeled, labeler, confidence, created_at)
        VALUES (:data, :label, :source, :date_labeled, :labeler, :confidence, :created_at)`,
		label)
	if err != nil {
		return handleError(err, "failed to insert label")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return handleError(err, "failed to get inserted label ID")
	}
	label.ID = id
	return nil
}

// InsertFeedback stores user feedback for an article
func InsertFeedback(db *sqlx.DB, feedback *Feedback) error {
	result, err := db.NamedExec(`
        INSERT INTO feedback (article_id, user_id, feedback_text, category, ensemble_output_id, source, created_at)
        VALUES (:article_id, :user_id, :feedback_text, :category, :ensemble_output_id, :source, :created_at)`,
		feedback)
	if err != nil {
		return handleError(err, "failed to insert feedback")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return handleError(err, "failed to get inserted feedback ID")
	}
	feedback.ID = id
	return nil
}

// FetchLatestEnsembleScore gets the most recent ensemble score for an article
func FetchLatestEnsembleScore(db *sqlx.DB, articleID int64) (float64, error) {
	var score float64
	err := db.Get(&score, `
        SELECT score FROM llm_scores 
        WHERE article_id = ? AND model = 'ensemble'
        ORDER BY created_at DESC LIMIT 1`,
		articleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0.0, nil // Return 0 if no score exists
		}
		return 0.0, handleError(err, "failed to fetch latest ensemble score")
	}
	return score, nil
}

// FetchLatestConfidence gets the most recent confidence score for an article
func FetchLatestConfidence(db *sqlx.DB, articleID int64) (float64, error) {
	var confidence float64
	err := db.Get(&confidence, `
        SELECT confidence FROM articles WHERE id = ?`,
		articleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0.0, nil // Return 0 if no confidence exists
		}
		return 0.0, handleError(err, "failed to fetch latest confidence score")
	}
	return confidence, nil
}

// ArticleExistsBySimilarTitle checks if an article with a similar title exists
func ArticleExistsBySimilarTitle(db *sqlx.DB, title string) (bool, error) {
	// Use SQLite's LIKE operator with wildcards to find similar titles
	// Remove common punctuation and spaces for comparison
	cleanTitle := strings.TrimSpace(strings.ToLower(title))
	cleanTitle = strings.ReplaceAll(cleanTitle, "'", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "\"", "")

	var exists bool
	err := db.Get(&exists, `
        SELECT EXISTS(
            SELECT 1 FROM articles 
            WHERE LOWER(REPLACE(REPLACE(REPLACE(title, "'", ""), '"', ""), ' ', '')) 
            LIKE '%' || ? || '%'
        )`,
		strings.ReplaceAll(cleanTitle, " ", ""))

	if err != nil {
		return false, handleError(err, "failed to check for similar title")
	}
	return exists, nil
}

// InsertArticle creates a new article record
func InsertArticle(db *sqlx.DB, article *Article) (int64, error) {
	result, err := db.NamedExec(`
        INSERT INTO articles (source, pub_date, url, title, content, created_at, composite_score, confidence, score_source)
        VALUES (:source, :pub_date, :url, :title, :content, :created_at, :composite_score, :confidence, :score_source)`,
		article)
	if err != nil {
		return 0, handleError(err, "failed to insert article")
	}
	return result.LastInsertId()
}

// InsertLLMScore creates a new LLM score record
func InsertLLMScore(exec sqlx.ExtContext, score *LLMScore) (int64, error) {
	result, err := sqlx.NamedExecContext(context.Background(), exec, `
        INSERT INTO llm_scores (article_id, model, score, metadata, version, created_at)
        VALUES (:article_id, :model, :score, :metadata, :version, :created_at)`,
		score)
	if err != nil {
		return 0, handleError(err, "failed to insert LLM score")
	}
	return result.LastInsertId()
}

// FetchArticles retrieves articles with optional filters
func FetchArticles(db *sqlx.DB, source string, leaning string, limit int, offset int) ([]Article, error) {
	query := `SELECT * FROM articles WHERE 1=1`
	var args []interface{}

	if source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}
	if leaning != "" {
		switch leaning {
		case "left":
			query += " AND composite_score < -0.1"
		case "right":
			query += " AND composite_score > 0.1"
		case "center":
			query += " AND composite_score BETWEEN -0.1 AND 0.1"
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

// FetchArticleByID retrieves a single article by ID
func FetchArticleByID(db *sqlx.DB, id int64) (*Article, error) {
	var article Article

	// Add retry logic with backoff for recently created articles
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.Get(&article, "SELECT * FROM articles WHERE id = ?", id)
		if err == nil {
			// Article found, return it
			return &article, nil
		}

		if err != sql.ErrNoRows {
			// For errors other than "no rows", log the specific error
			log.Printf("[ERROR] FetchArticleByID %d failed (attempt %d): %v", id, attempt+1, err)
			// Don't retry for database errors
			break
		}

		log.Printf("[INFO] FetchArticleByID %d: article not found, retrying after %v (attempt %d of %d)", id, retryDelay, attempt+1, maxRetries)
		// Only for "no rows" error, wait and retry
		// This helps with timing issues when an article was just created
		// but the transaction hasn't fully committed yet
		if attempt < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	// Handle the final error
	if err == sql.ErrNoRows {
		log.Printf("[WARN] FetchArticleByID %d: article not found after %d attempts", id, maxRetries)
		return nil, ErrArticleNotFound
	}
	log.Printf("[ERROR] FetchArticleByID %d failed with database error: %v", id, err)
	return nil, handleError(err, "failed to fetch article")
}

// FetchLLMScores retrieves all LLM scores for an article
func FetchLLMScores(db *sqlx.DB, articleID int64) ([]LLMScore, error) {
	var scores []LLMScore
	err := db.Select(&scores, "SELECT * FROM llm_scores WHERE article_id = ? ORDER BY created_at DESC", articleID)
	if err != nil {
		return nil, handleError(err, "failed to fetch LLM scores")
	}
	return scores, nil
}

// UpdateArticleScore updates the composite score for an article
func UpdateArticleScore(db *sqlx.DB, articleID int64, score float64, confidence float64) error {
	_, err := db.Exec(`
        UPDATE articles 
        SET composite_score = ?, confidence = ?, score_source = 'llm'
        WHERE id = ?`,
		score, confidence, articleID)
	if err != nil {
		return handleError(err, "failed to update article score")
	}
	return nil
}

// UpdateArticleScoreLLM updates the composite score for an article, specifically from LLM rescoring
func UpdateArticleScoreLLM(exec sqlx.ExtContext, articleID int64, score float64, confidence float64) error {
	_, err := exec.ExecContext(context.Background(), `
        UPDATE articles 
        SET composite_score = ?, confidence = ?, score_source = 'llm'
        WHERE id = ?`,
		score, confidence, articleID)
	if err != nil {
		return handleError(err, "failed to update article score (LLM)")
	}
	return nil
}

// ArticleExistsByURL checks if an article exists with the given URL
func ArticleExistsByURL(db *sqlx.DB, url string) (bool, error) {
	var exists bool
	err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM articles WHERE url = ?)", url)
	if err != nil {
		return false, handleError(err, "failed to check article URL existence")
	}
	return exists, nil
}

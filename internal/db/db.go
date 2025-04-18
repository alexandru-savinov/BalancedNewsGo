package db

import (
	"database/sql"
	"strings"
	"time"

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
	ErrDuplicateURL = apperrors.New("Article with this URL already exists", "conflict")
	ErrInvalidScore = apperrors.New("Invalid score value", "validation_error")
)

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

// InsertLabel creates a new label entry
func InsertLabel(db *sqlx.DB, label *Label) error {
	_, err := db.NamedExec(`
		INSERT INTO labels (data, label, source, date_labeled, labeler, confidence, created_at)
		VALUES (:data, :label, :source, :date_labeled, :labeler, :confidence, :created_at)`,
		label)
	if err != nil {
		return handleError(err, "failed to insert label")
	}
	return nil
}

// InsertFeedback creates a new feedback entry
func InsertFeedback(db *sqlx.DB, feedback *Feedback) (int64, error) {
	result, err := db.NamedExec(`
		INSERT INTO feedback (article_id, user_id, feedback_text, category, ensemble_output_id, source, created_at)
		VALUES (:article_id, :user_id, :feedback_text, :category, :ensemble_output_id, :source, :created_at)`,
		feedback)
	if err != nil {
		return 0, handleError(err, "failed to insert feedback")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, handleError(err, "failed to get inserted feedback ID")
	}
	return id, nil
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
			return 0, nil // Return 0 if no score found
		}
		return 0, handleError(err, "failed to fetch latest ensemble score")
	}
	return score, nil
}

// FetchLatestConfidence gets the most recent confidence score for an article
func FetchLatestConfidence(db *sqlx.DB, articleID int64) (float64, error) {
	var confidence float64
	err := db.Get(&confidence, `
		SELECT json_extract(metadata, '$.confidence') as confidence 
		FROM llm_scores 
		WHERE article_id = ? AND model = 'ensemble'
		ORDER BY created_at DESC LIMIT 1`,
		articleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // Return 0 if no confidence found
		}
		return 0, handleError(err, "failed to fetch latest confidence")
	}
	return confidence, nil
}

// ArticleExistsBySimilarTitle checks if an article with a similar title exists
func ArticleExistsBySimilarTitle(db *sqlx.DB, title string) (bool, error) {
	// Normalize the title by converting to lowercase and removing punctuation
	normalized := strings.ToLower(title)
	normalized = strings.Map(func(r rune) rune {
		if strings.ContainsRune(",.!?:;\"'()[]{}", r) {
			return -1
		}
		return r
	}, normalized)

	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM articles 
			WHERE lower(replace(replace(replace(title, '.', ''), ',', ''), ':', '')) LIKE ?
		)`
	err := db.Get(&exists, query, "%"+normalized+"%")
	if err != nil {
		return false, handleError(err, "failed to check article existence by title")
	}
	return exists, nil
}

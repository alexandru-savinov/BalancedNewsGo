package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Errors
var (
	ErrArticleNotFound  = errors.New("article not found")
	ErrFeedbackNotFound = errors.New("feedback not found")
	ErrDuplicateURL     = errors.New("article with this URL already exists")
)

// Article represents a news article with bias information
type Article struct {
	ID             int64      `db:"id" json:"id"`
	Source         string     `db:"source" json:"source"`
	PubDate        time.Time  `db:"pub_date" json:"pub_date"`
	URL            string     `db:"url" json:"url"`
	Title          string     `db:"title" json:"title"`
	Content        string     `db:"content" json:"content"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	Status         *string    `db:"status" json:"status,omitempty"`
	FailCount      *int       `db:"fail_count" json:"fail_count,omitempty"`
	LastAttempt    *time.Time `db:"last_attempt" json:"last_attempt,omitempty"`
	Escalated      *bool      `db:"escalated" json:"escalated,omitempty"`
	CompositeScore *float64   `db:"composite_score" json:"composite_score,omitempty"`
	Confidence     *float64   `db:"confidence" json:"confidence,omitempty"`
	ScoreSource    *string    `db:"score_source" json:"score_source,omitempty"`
}

// LLMScore represents a political bias score from an LLM model
type LLMScore struct {
	ID        int64     `db:"id" json:"id"`
	ArticleID int64     `db:"article_id" json:"article_id"`
	Model     string    `db:"model" json:"model"`
	Score     float64   `db:"score" json:"score"`
	Metadata  string    `db:"metadata" json:"metadata"`
	Version   string    `db:"version" json:"version"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Feedback represents user feedback on an article
type Feedback struct {
	ID               int64     `db:"id" json:"id"`
	ArticleID        int64     `db:"article_id" json:"article_id"`
	UserID           string    `db:"user_id" json:"user_id"`
	FeedbackText     string    `db:"feedback_text" json:"feedback_text"`
	Category         string    `db:"category" json:"category"`
	EnsembleOutputID *int64    `db:"ensemble_output_id" json:"ensemble_output_id,omitempty"`
	Source           string    `db:"source" json:"source,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// Label represents a training label for the system
type Label struct {
	ID          int64     `db:"id" json:"id"`
	Data        string    `db:"data" json:"data"`
	Label       string    `db:"label" json:"label"`
	Source      string    `db:"source" json:"source"`
	DateLabeled time.Time `db:"date_labeled" json:"date_labeled"`
	Labeler     string    `db:"labeler" json:"labeler"`
	Confidence  float64   `db:"confidence" json:"confidence"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// ArticleFilter defines filters for retrieving articles
type ArticleFilter struct {
	Source  string
	Leaning string
	Limit   int
	Offset  int
}

// ArticleScore represents a score update for an article
type ArticleScore struct {
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
}

// ArticleFeedback represents feedback for an article
type ArticleFeedback struct {
	ArticleID        int64     `json:"article_id"`
	UserID           string    `json:"user_id"`
	FeedbackText     string    `json:"feedback_text"`
	Category         string    `json:"category"`
	EnsembleOutputID *int64    `json:"ensemble_output_id,omitempty"`
	Source           string    `json:"source,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// DBOperations defines the interface for database operations
type DBOperations interface {
	// Article retrieval operations
	GetArticleByID(ctx context.Context, id int64) (*Article, error)
	FetchArticleByID(ctx context.Context, id int64) (*Article, error) // Alias for GetArticleByID
	GetArticles(ctx context.Context, filter ArticleFilter) ([]*Article, error)
	FetchArticles(ctx context.Context, source, leaning string, limit, offset int) ([]*Article, error) // Used in handlers

	// Article creation/update operations
	InsertArticle(ctx context.Context, article *Article) (int64, error)
	UpdateArticleScore(ctx context.Context, articleID int64, score float64, confidence float64) error
	UpdateArticleScoreObj(ctx context.Context, articleID int64, score *ArticleScore, confidence float64) error
	ArticleExistsByURL(ctx context.Context, url string) (bool, error)

	// Feedback operations
	SaveArticleFeedback(ctx context.Context, feedback *ArticleFeedback) error
	InsertFeedback(ctx context.Context, feedback *Feedback) error

	// LLM Score operations
	FetchLLMScores(ctx context.Context, articleID int64) ([]LLMScore, error)
}

// DBInstance implements the DBOperations interface
type DBInstance struct {
	DB *sqlx.DB
}

// GetArticleByID retrieves an article by ID
func (d *DBInstance) GetArticleByID(ctx context.Context, id int64) (*Article, error) {
	return FetchArticleByID(d.DB, id)
}

// FetchArticleByID is an alias for GetArticleByID
func (d *DBInstance) FetchArticleByID(ctx context.Context, id int64) (*Article, error) {
	return FetchArticleByID(d.DB, id)
}

// GetArticles retrieves articles based on filter criteria
func (d *DBInstance) GetArticles(ctx context.Context, filter ArticleFilter) ([]*Article, error) {
	articles, err := FetchArticles(d.DB, filter.Source, filter.Leaning, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}

	// Convert from []Article to []*Article
	result := make([]*Article, len(articles))
	for i := range articles {
		result[i] = &articles[i]
	}
	return result, nil
}

// FetchArticles retrieves articles using source, leaning, limit and offset parameters
func (d *DBInstance) FetchArticles(ctx context.Context, source, leaning string, limit, offset int) ([]*Article, error) {
	articles, err := FetchArticles(d.DB, source, leaning, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert from []Article to []*Article
	result := make([]*Article, len(articles))
	for i := range articles {
		result[i] = &articles[i]
	}
	return result, nil
}

// InsertArticle inserts a new article
func (d *DBInstance) InsertArticle(ctx context.Context, article *Article) (int64, error) {
	return InsertArticle(d.DB, article)
}

// UpdateArticleScore updates an article's score
func (d *DBInstance) UpdateArticleScore(ctx context.Context, articleID int64, score float64, confidence float64) error {
	return UpdateArticleScore(d.DB, articleID, score, confidence)
}

// UpdateArticleScoreObj updates an article's score using an ArticleScore object
func (d *DBInstance) UpdateArticleScoreObj(ctx context.Context, articleID int64, score *ArticleScore, confidence float64) error {
	if score == nil {
		return errors.New("score cannot be nil")
	}
	return UpdateArticleScore(d.DB, articleID, score.Score, confidence)
}

// ArticleExistsByURL checks if an article exists by URL
func (d *DBInstance) ArticleExistsByURL(ctx context.Context, url string) (bool, error) {
	return ArticleExistsByURL(d.DB, url)
}

// SaveArticleFeedback saves article feedback
func (d *DBInstance) SaveArticleFeedback(ctx context.Context, feedback *ArticleFeedback) error {
	if feedback == nil {
		return errors.New("feedback cannot be nil")
	}
	dbFeedback := &Feedback{
		ArticleID:        feedback.ArticleID,
		UserID:           feedback.UserID,
		FeedbackText:     feedback.FeedbackText,
		Category:         feedback.Category,
		EnsembleOutputID: feedback.EnsembleOutputID,
		Source:           feedback.Source,
		CreatedAt:        time.Now(),
	}
	return InsertFeedback(d.DB, dbFeedback)
}

// InsertFeedback inserts article feedback
func (d *DBInstance) InsertFeedback(ctx context.Context, feedback *Feedback) error {
	return InsertFeedback(d.DB, feedback)
}

// FetchLLMScores retrieves LLM scores for an article
func (d *DBInstance) FetchLLMScores(ctx context.Context, articleID int64) ([]LLMScore, error) {
	return FetchLLMScores(d.DB, articleID)
}

// New creates a new database connection
func New(connString string) (*DBInstance, error) {
	db, err := sqlx.Open("sqlite", connString)
	if err != nil {
		return nil, err
	}
	return &DBInstance{DB: db}, nil
}

// Close closes the database connection
func (d *DBInstance) Close() error {
	return d.DB.Close()
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
	case strings.Contains(errMsg, "UNIQUE constraint") || strings.Contains(errMsg, "unique constraint"):
		return ErrDuplicateURL
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
	// Normalize input title: lowercase and remove punctuation/spaces
	cleanTitle := strings.ToLower(strings.TrimSpace(title))
	for _, r := range []string{"'", `"`, ",", "!", ".", "?", ";", ":", " "} {
		cleanTitle = strings.ReplaceAll(cleanTitle, r, "")
	}

	var exists bool
	err := db.Get(&exists, `
	    SELECT EXISTS(
	        SELECT 1 FROM articles
	        WHERE LOWER(
	            REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(title, "'", ""), '"', ""), ' ', ""), ',', ""), '!', ""), '.', ""), '?', ""), ';', "")
	        ) LIKE '%' || ? || '%'
	    )`, cleanTitle)
	if err != nil {
		return false, handleError(err, "failed to check for similar title")
	}
	return exists, nil
}

// InsertArticle creates a new article record within a transaction to ensure atomicity of the duplicate check and insert.
func InsertArticle(db *sqlx.DB, article *Article) (int64, error) {
	tx, err := db.Beginx()
	if err != nil {
		return 0, handleError(err, "failed to begin transaction for article insert")
	}
	// Ensure transaction is rolled back in case of error
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback() // Rollback on panic
			panic(p)          // Re-panic
		} else if err != nil {
			_ = tx.Rollback() // Rollback on error
		}
	}()

	// 1. Check if URL exists within the transaction
	var exists bool
	err = tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM articles WHERE url = ?)", article.URL)
	if err != nil && err != sql.ErrNoRows { // Allow ErrNoRows here, should return exists=false
		return 0, handleError(err, "failed to check article URL existence in transaction")
	}
	if exists {
		err = ErrDuplicateURL // Explicitly set the error to be returned
		return 0, err
	}

	// 2. Insert the article if it doesn't exist
	result, err := tx.NamedExec(`
        INSERT INTO articles (source, pub_date, url, title, content, created_at, composite_score, confidence, score_source,
                              status, fail_count, last_attempt, escalated)
        VALUES (:source, :pub_date, :url, :title, :content, :created_at, :composite_score, :confidence, :score_source,
                :status, :fail_count, :last_attempt, :escalated)`,
		article)
	if err != nil {
		log.Printf("[ERROR] Failed to insert article in transaction: %v", err)
		// handleError will check for UNIQUE constraint error here again just in case of race conditions outside the explicit check
		return 0, handleError(err, "failed to insert article")
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve last insert ID: %v", err)
		return 0, handleError(err, "failed to get inserted article ID")
	}

	// 3. Commit the transaction
	err = tx.Commit()
	if err != nil {
		return 0, handleError(err, "failed to commit transaction for article insert")
	}

	log.Printf("[INFO] Article inserted successfully with ID: %d", id)
	return id, nil
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

	// Add debug logging
	log.Printf("[DEBUG] FetchArticles query: %s with args: %v", query, args)

	// Use db.Unsafe() to allow scanning of null values
	unsafe := db.Unsafe()
	var articles []Article
	err := unsafe.Select(&articles, query, args...)
	if err != nil {
		log.Printf("[ERROR] FetchArticles failed: %v", err)
		return nil, handleError(err, "failed to fetch articles")
	}

	log.Printf("[INFO] FetchArticles found %d articles", len(articles))
	return articles, nil
}

// FetchArticleByID retrieves a single article by ID
func FetchArticleByID(db *sqlx.DB, id int64) (*Article, error) {
	log.Printf("[DEBUG] FetchArticleByID called with id: %d", id)
	if db == nil {
		log.Printf("[ERROR] Database connection is nil")
		return nil, errors.New("database connection is nil")
	}

	var article Article

	// Add retry logic with backoff for recently created articles
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		log.Printf("[DEBUG] Attempt %d to fetch article with id: %d", attempt+1, id)
		err = db.Get(&article, "SELECT * FROM articles WHERE id = ?", id)
		if err == nil {
			// Article found, return it
			log.Printf("[INFO] Article fetched successfully: %+v", article)
			return &article, nil
		}

		if err != sql.ErrNoRows {
			// For errors other than "no rows", log the specific error
			log.Printf("[ERROR] FetchArticleByID failed (attempt %d): %v", attempt+1, err)
			// Don't retry for database errors
			break
		}

		log.Printf("[INFO] FetchArticleByID: article not found, retrying after %v (attempt %d of %d)", retryDelay, attempt+1, maxRetries)
		// Only for "no rows" error, wait and retry
		if attempt < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	// Handle the final error
	if err == sql.ErrNoRows {
		log.Printf("[WARN] FetchArticleByID: article not found after %d attempts", maxRetries)
		return nil, ErrArticleNotFound
	}
	log.Printf("[ERROR] FetchArticleByID failed with database error: %v", err)
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
	log.Printf("[DEBUG][CONFIDENCE] UpdateArticleScoreLLM called with articleID=%d, score=%.4f, confidence=%.4f",
		articleID, score, confidence)

	result, err := exec.ExecContext(context.Background(), `
        UPDATE articles 
        SET composite_score = ?, confidence = ?, score_source = 'llm'
        WHERE id = ?`,
		score, confidence, articleID)

	if err != nil {
		log.Printf("[ERROR][CONFIDENCE] Failed to update article score in database: %v", err)
		return handleError(err, "failed to update article score (LLM)")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[ERROR][CONFIDENCE] Error getting rows affected: %v", err)
	} else {
		log.Printf("[DEBUG][CONFIDENCE] UpdateArticleScoreLLM affected %d rows for articleID=%d",
			rowsAffected, articleID)

		if rowsAffected == 0 {
			log.Printf("[WARN][CONFIDENCE] No rows updated for articleID=%d - article may not exist", articleID)
		}
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

// InitDB initializes and returns a database connection to the specified SQLite database file
func InitDB(dbPath string) (*sqlx.DB, error) {
	// Open SQLite database connection
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify database connection is working
	if err = db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Error closing DB after ping failure: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection properties
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Set busy_timeout to help with concurrent access
	_, err = db.Exec("PRAGMA busy_timeout = 5000") // 5 seconds
	if err != nil {
		log.Printf("Failed to set busy_timeout: %v", err)
		// Not fatal, but log it
	}

	// Drop existing tables to ensure fresh schema for testing/debugging
	dropSchema := `
	DROP TABLE IF EXISTS articles;
	DROP TABLE IF EXISTS llm_scores;
	DROP TABLE IF EXISTS feedback;
	DROP TABLE IF EXISTS labels;
	`
	_, err = db.Exec(dropSchema)
	if err != nil {
		log.Printf("Failed to drop existing tables: %v", err)
		// Not fatal, but log it
	}

	// Define the database schema
	schema := `
	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		pub_date TIMESTAMP NOT NULL,
		url TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'pending',
		fail_count INTEGER DEFAULT 0,
		last_attempt DATETIME,
		escalated BOOLEAN DEFAULT 0,
		composite_score REAL,
		confidence REAL,
		score_source TEXT
	);

	CREATE TABLE IF NOT EXISTS llm_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER NOT NULL,
		model TEXT NOT NULL,
		score REAL NOT NULL,
		metadata TEXT,
		version TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles (id)
	);

	CREATE TABLE IF NOT EXISTS feedback (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER NOT NULL,
		user_id TEXT,
		feedback_text TEXT,
		category TEXT,
		ensemble_output_id INTEGER,
		source TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles (id)
	);

	CREATE TABLE IF NOT EXISTS labels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		data TEXT NOT NULL,
		label TEXT NOT NULL,
		source TEXT NOT NULL,
		date_labeled TIMESTAMP NOT NULL,
		labeler TEXT NOT NULL,
		confidence REAL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	// Initialize database schema
	_, err = db.Exec(schema)
	if err != nil {
		log.Printf("Failed to initialize DB schema: %v", err)
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Error closing DB after schema init failure: %v", closeErr)
		}
		return nil, err
	}

	// Return the database connection
	return db, nil
}

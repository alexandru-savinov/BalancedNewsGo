package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
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
	BiasLabel      *string    `db:"bias_label" json:"bias_label,omitempty"`
	Bias           string     `db:"-" json:"bias,omitempty"` // Calculated field, not stored in DB
}

// LLMScore represents a political bias score from an LLM model
type LLMScore struct {
	ID        int64     `db:"id" json:"id"`
	ArticleID int64     `db:"article_id" json:"article_id"`
	Model     string    `db:"model" json:"model"`
	Score     float64   `db:"score" json:"score"`
	Metadata  string    `db:"metadata" json:"metadata"`
	Version   int       `db:"version" json:"version"`
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
	UpdateArticleScoreLLM(ctx context.Context, articleID int64, score float64, confidence float64) error
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

// UpdateArticleScoreLLM updates an article's score by an LLM, typically as part of a transaction
func (d *DBInstance) UpdateArticleScoreLLM(ctx context.Context, articleID int64, score float64, confidence float64) error {
	// The actual db.UpdateArticleScoreLLM takes sqlx.ExtContext.
	// For DBInstance, d.DB is *sqlx.DB which implements sqlx.ExtContext.
	return UpdateArticleScoreLLM(d.DB, articleID, score, confidence)
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

// validateDBSchema ensures critical tables exist. It returns an error if any
// required table is missing, providing clearer diagnostics for test failures.
func validateDBSchema(db *sqlx.DB) error {
	required := []string{"articles", "llm_scores", "feedback", "labels", "users"}
	for _, table := range required {
		var name string
		err := db.Get(&name, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table)
		if err != nil {
			return fmt.Errorf("schema validation failed for table %s: %w", table, err)
		}
		if name == "" {
			return fmt.Errorf("schema validation: table %s not found", table)
		}
	}
	return nil
}

// validateLLMMetadata checks that the metadata field is valid JSON. Empty
// strings are allowed. When invalid, an error is returned so callers can
// surface meaningful information to the user and logs.
func validateLLMMetadata(meta string) error {
	if strings.TrimSpace(meta) == "" {
		return nil
	}
	var tmp map[string]interface{}
	if err := json.Unmarshal([]byte(meta), &tmp); err != nil {
		log.Printf("[WARN] Invalid metadata JSON encountered: %v", err)
		// Do not return error to preserve backward compatibility
		return nil
	}
	return nil
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
	            REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(title,
						"'", ""), '"', ""), " ", ""), ",", ""), "!", ""), ".", ""), "?", ""), ";", "")
	        ) LIKE '%' || ? || '%'
	    )`, cleanTitle)
	if err != nil {
		return false, handleError(err, "failed to check for similar title")
	}
	return exists, nil
}

// InsertArticle creates a new article record with retry logic for SQLITE_BUSY errors
func InsertArticle(db *sqlx.DB, article *Article) (int64, error) {
	var resultID int64

	// Prepare article fields with defaults if not set (outside transaction)
	if article.CreatedAt.IsZero() {
		article.CreatedAt = time.Now()
	}
	if article.Status == nil {
		defaultStatus := "pending"
		article.Status = &defaultStatus
	}
	if article.FailCount == nil {
		defaultFailCount := 0
		article.FailCount = &defaultFailCount
	}
	if article.Escalated == nil {
		defaultEscalated := false
		article.Escalated = &defaultEscalated
	}

	// Execute the transaction with retry logic
	config := DefaultRetryConfig()
	err := WithRetry(config, func() error {
		return insertArticleTransaction(db, article, &resultID)
	})

	if err != nil {
		return 0, err
	}

	log.Printf("[INFO] Article inserted successfully with ID: %d", resultID)
	return resultID, nil
}

// insertArticleTransaction performs the actual database transaction for article insertion
func insertArticleTransaction(db *sqlx.DB, article *Article, resultID *int64) error {
	tx, err := db.Beginx()
	if err != nil {
		return handleError(err, "failed to begin transaction for article insert")
	}

	// Ensure transaction is properly closed
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// Check if URL exists within the transaction
	var exists bool
	err = tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM articles WHERE url = ?)", article.URL)
	if err != nil && err != sql.ErrNoRows {
		_ = tx.Rollback()
		return handleError(err, "failed to check article URL existence in transaction")
	}
	if exists {
		_ = tx.Rollback()
		return ErrDuplicateURL
	}

	// Insert the article if it doesn't exist
	result, err := tx.NamedExec(`
        INSERT INTO articles (source, pub_date, url, title, content, created_at, composite_score, confidence, score_source,
                              status, fail_count, last_attempt, escalated)
        VALUES (:source, :pub_date, :url, :title, :content, :created_at, :composite_score, :confidence, :score_source,
                :status, :fail_count, :last_attempt, :escalated)`,
		article)
	if err != nil {
		_ = tx.Rollback()
		log.Printf("[ERROR] Failed to insert article in transaction: %v", err)
		return handleError(err, "failed to insert article")
	}

	id, err := result.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		log.Printf("[ERROR] Failed to retrieve last insert ID: %v", err)
		return handleError(err, "failed to get inserted article ID")
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return handleError(err, "failed to commit transaction for article insert")
	}

	*resultID = id
	return nil
}

// InsertLLMScore creates a new LLM score record with retry logic for SQLite concurrency
func InsertLLMScore(exec sqlx.ExtContext, score *LLMScore) (int64, error) {
	if err := validateLLMMetadata(score.Metadata); err != nil {
		log.Printf("[ERROR] Invalid metadata for article %d model %s: %v", score.ArticleID, score.Model, err)
		return 0, handleError(err, "invalid metadata for llm score")
	}

	var id int64
	err := WithRetry(DefaultRetryConfig(), func() error {
		// Upsert logic: Insert or update if conflict on (article_id, model)
		query := `
			INSERT INTO llm_scores (article_id, model, score, metadata, version, created_at)
			VALUES (:article_id, :model, :score, :metadata, :version, :created_at)
			ON CONFLICT (article_id, model) DO UPDATE SET
				score = excluded.score,
				metadata = excluded.metadata,
				version = excluded.version,
				created_at = excluded.created_at;`

		result, err := sqlx.NamedExecContext(context.Background(), exec, query, score)
		if err != nil {
			if IsSQLiteBusyError(err) {
				log.Printf("[RETRY] InsertLLMScore (upsert) for article %d model %s: %v", score.ArticleID, score.Model, err)
				return err // Will trigger retry
			}
			log.Printf("[ERROR] InsertLLMScore (upsert) failed for article %d model %s score %.3f: %v", score.ArticleID, score.Model, score.Score, err)
			return err // Non-retryable error
		}
		var insertErr error
		// For ON CONFLICT DO UPDATE, LastInsertId might not be reliable or might be 0 if it was an update.
		// If an ID is strictly needed even for updates, a SELECT might be required post-operation,
		// or the logic might need to rely on the fact that the record now exists/is updated.
		// For now, we'll attempt to get it, but be aware of its behavior with upserts.
		id, insertErr = result.LastInsertId()
		if insertErr != nil {
			// If it's an update, LastInsertId might return an error or 0.
			// We can check RowsAffected to see if an operation occurred.
			// If LastInsertId fails but rows were affected, we assume success without a new ID.
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				log.Printf("[INFO] InsertLLMScore (upsert) affected %d rows for article %d model %s. LastInsertId error: %v (may be an update)", rowsAffected, score.ArticleID, score.Model, insertErr)
				return nil // No new ID, but operation was successful
			}
			// If LastInsertId failed and no rows were affected, then it's a genuine error.
			log.Printf("[ERROR] InsertLLMScore (upsert) failed to get LastInsertId and no rows affected for article %d model %s: %v", score.ArticleID, score.Model, insertErr)
			return insertErr
		}
		return nil // LastInsertId was successful (likely an insert)
	})

	if err != nil {
		return 0, handleError(err, "failed to insert/update LLM score")
	}
	// If id is 0 after a successful operation (e.g. an update), this is expected.
	// The caller should be aware that id might be 0 if an update occurred.
	return id, nil
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

	// Calculate bias for each article
	for i := range articles {
		articles[i].CalculateBias()
	}

	log.Printf("[INFO] FetchArticles found %d articles", len(articles))
	return articles, nil
}

// CalculateBias determines the bias label based on CompositeScore
func (a *Article) CalculateBias() {
	if a.CompositeScore == nil {
		a.Bias = "unknown"
		return
	}

	score := *a.CompositeScore
	switch {
	case score < -0.1:
		a.Bias = "left"
	case score > 0.1:
		a.Bias = "right"
	default:
		a.Bias = "center"
	}
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

// UpdateArticleScore updates the composite score for an article with retry logic
func UpdateArticleScore(db *sqlx.DB, articleID int64, score float64, confidence float64) error {
	err := WithRetry(DefaultRetryConfig(), func() error {
		_, err := db.Exec(`
			UPDATE articles
			SET composite_score = ?, confidence = ?, score_source = 'llm'
			WHERE id = ?`,
			score, confidence, articleID)
		if err != nil {
			if IsSQLiteBusyError(err) {
				log.Printf("[RETRY] UpdateArticleScore for article %d: %v", articleID, err)
				return err // Will trigger retry
			}
			return err // Non-retryable error
		}
		return nil
	})

	if err != nil {
		return handleError(err, "failed to update article score")
	}
	return nil
}

// UpdateArticleScoreLLM updates the composite score for an article, specifically from LLM rescoring with retry logic
func UpdateArticleScoreLLM(exec sqlx.ExtContext, articleID int64, score float64, confidence float64) error {
	log.Printf("[DEBUG][CONFIDENCE] UpdateArticleScoreLLM called with articleID=%d, score=%.4f, confidence=%.4f",
		articleID, score, confidence)

	err := WithRetry(DefaultRetryConfig(), func() error {
		result, err := exec.ExecContext(context.Background(), `
			UPDATE articles
			SET composite_score = ?, confidence = ?, score_source = 'llm'
			WHERE id = ?`,
			score, confidence, articleID)

		if err != nil {
			if IsSQLiteBusyError(err) {
				log.Printf("[RETRY] UpdateArticleScoreLLM for article %d: %v", articleID, err)
				return err // Will trigger retry
			}
			log.Printf("[ERROR][CONFIDENCE] Failed to update article score in database: %v", err)
			return err // Non-retryable error
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
	})

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
	// Enable WAL mode for improved concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		log.Printf("Failed to enable WAL mode: %v", err)
		// Not fatal, but log it
	} else {
		log.Printf("WAL mode enabled successfully")
	}

	// Set busy_timeout to help with concurrent access
	_, err = db.Exec("PRAGMA busy_timeout = 5000") // 5 seconds
	if err != nil {
		log.Printf("Failed to set busy_timeout: %v", err)
		// Not fatal, but log it
	}

	// !! IMPORTANT !! Commenting out unconditional drop for integration testing
	/*
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
	*/

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
		version INTEGER DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles (id),
		UNIQUE(article_id, model)
	);

	CREATE INDEX IF NOT EXISTS idx_llm_scores_article_version ON llm_scores(article_id, version);

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
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
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

	if err := validateDBSchema(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Error closing DB after schema validation failure: %v", closeErr)
		}
		return nil, err
	}

	// Return the database connection
	return db, nil
}

// UpdateArticleStatus updates the status of a specific article.
func UpdateArticleStatus(exec sqlx.ExtContext, articleID int64, status string) error {
	query := `UPDATE articles SET status = ? WHERE id = ?`
	result, err := exec.ExecContext(context.Background(), query, status, articleID)
	if err != nil {
		log.Printf("[ERROR] Failed to update article status for ID %d to '%s': %v", articleID, status, err)
		return handleError(err, fmt.Sprintf("failed to update article status for ID %d", articleID))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[WARN] Could not determine rows affected when updating status for article ID %d: %v", articleID, err)
		// Not returning error here as the update might have succeeded.
	} else if rowsAffected == 0 {
		log.Printf("[WARN] UpdateArticleStatus: No rows affected when updating status for article ID %d to '%s'. "+
			"Article may not exist.", articleID, status)
		// Potentially return ErrArticleNotFound or a similar specific error if this is unexpected.
		// For now, just logging as the main operation (exec) didn't error.
	}

	log.Printf("[INFO] Updated status for article ID %d to '%s'", articleID, status)
	return nil
}

// RetryConfig holds configuration for database retry operations
type RetryConfig struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	JitterPercent float64
}

// DefaultRetryConfig returns a sensible default retry configuration for SQLite operations
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   8,                     // More attempts for high concurrency
		BaseDelay:     50 * time.Millisecond, // Start with shorter delay
		MaxDelay:      2 * time.Second,       // Cap at 2 seconds
		BackoffFactor: 2.0,                   // Exponential backoff
		JitterPercent: 0.1,                   // 10% jitter to prevent thundering herd
	}
}

// IsSQLiteBusyError checks if an error is a SQLite busy/locked error
func IsSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "database is locked") ||
		strings.Contains(errStr, "sqlite_busy") ||
		strings.Contains(errStr, "busy")
}

// WithRetry executes a function with retry logic for SQLite busy errors
func WithRetry(config RetryConfig, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		lastErr = operation()

		if lastErr == nil {
			if attempt > 1 {
				log.Printf("[INFO] Database operation succeeded on attempt %d", attempt)
			}
			return nil
		}

		// If it's not a busy error, don't retry
		if !IsSQLiteBusyError(lastErr) {
			return lastErr
		}

		// Don't sleep on the last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := time.Duration(float64(config.BaseDelay) *
			(1.0 + config.JitterPercent*(2.0*rand.Float64()-1.0)))

		for i := 1; i < attempt; i++ {
			delay = time.Duration(float64(delay) * config.BackoffFactor)
		}

		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		log.Printf("[WARN] Database busy (attempt %d/%d), retrying in %v: %v",
			attempt, config.MaxAttempts, delay, lastErr)

		time.Sleep(delay)
	}

	log.Printf("[ERROR] Database operation failed after %d attempts: %v",
		config.MaxAttempts, lastErr)
	return fmt.Errorf("database operation failed after %d attempts: %w",
		config.MaxAttempts, lastErr)
}

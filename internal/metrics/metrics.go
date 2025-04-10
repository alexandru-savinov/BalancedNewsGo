package metrics

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type ValidationMetric struct {
	Day           string  `db:"day" json:"day"`
	Label         string  `db:"label" json:"label"`
	LabelCount    int     `db:"label_count" json:"label_count"`
	AvgConfidence float64 `db:"avg_confidence" json:"avg_confidence"`
}

type FeedbackSummary struct {
	Day           string `db:"day" json:"day"`
	Category      string `db:"category" json:"category"`
	FeedbackCount int    `db:"feedback_count" json:"feedback_count"`
}

type UncertaintyRate struct {
	Day               string  `db:"day" json:"day"`
	LowConfidenceRate float64 `db:"low_confidence_ratio" json:"low_confidence_ratio"`
}

type Disagreement struct {
	ArticleID          int64     `db:"article_id" json:"article_id"`
	DistinctCategories int       `db:"distinct_categories" json:"distinct_categories"`
	LastFeedbackTime   time.Time `db:"last_feedback_time" json:"last_feedback_time"`
}

type OutlierScore struct {
	ArticleID  int64   `db:"article_id" json:"article_id"`
	MaxScore   float64 `db:"max_score" json:"max_score"`
	MinScore   float64 `db:"min_score" json:"min_score"`
	ScoreRange float64 `db:"score_range" json:"score_range"`
	ScoreCount int     `db:"score_count" json:"score_count"`
}

func GetValidationMetrics(db *sqlx.DB) ([]ValidationMetric, error) {
	var metrics []ValidationMetric
	err := db.Select(&metrics, "SELECT * FROM validation_metrics")
	return metrics, err
}

func GetFeedbackSummary(db *sqlx.DB) ([]FeedbackSummary, error) {
	var summaries []FeedbackSummary
	err := db.Select(&summaries, "SELECT * FROM feedback_summary")
	return summaries, err
}

func GetUncertaintyRates(db *sqlx.DB) ([]UncertaintyRate, error) {
	var rates []UncertaintyRate
	err := db.Select(&rates, "SELECT * FROM uncertainty_rates")
	return rates, err
}

func GetDisagreements(db *sqlx.DB) ([]Disagreement, error) {
	var disagreements []Disagreement
	err := db.Select(&disagreements, "SELECT * FROM disagreement_rates")
	return disagreements, err
}

func GetOutlierScores(db *sqlx.DB) ([]OutlierScore, error) {
	var outliers []OutlierScore
	err := db.Select(&outliers, "SELECT * FROM outlier_scores")
	return outliers, err
}

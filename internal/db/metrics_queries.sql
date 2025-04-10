-- Validation metrics over time: label counts and average confidence
CREATE VIEW IF NOT EXISTS validation_metrics AS
SELECT
    DATE(date_labeled) AS day,
    label,
    COUNT(*) AS label_count,
    AVG(confidence) AS avg_confidence
FROM labels
GROUP BY day, label
ORDER BY day DESC;

-- Feedback summaries: counts by category over time
CREATE VIEW IF NOT EXISTS feedback_summary AS
SELECT
    DATE(created_at) AS day,
    category,
    COUNT(*) AS feedback_count
FROM feedback
GROUP BY day, category
ORDER BY day DESC;

-- Uncertainty rates: proportion of low-confidence labels per day
CREATE VIEW IF NOT EXISTS uncertainty_rates AS
SELECT
    DATE(date_labeled) AS day,
    COUNT(CASE WHEN confidence < 0.5 THEN 1 END) * 1.0 / COUNT(*) AS low_confidence_ratio
FROM labels
GROUP BY day
ORDER BY day DESC;

-- Disagreement rates: articles with mixed feedback categories
CREATE VIEW IF NOT EXISTS disagreement_rates AS
SELECT
    article_id,
    COUNT(DISTINCT category) AS distinct_categories,
    MAX(created_at) AS last_feedback_time
FROM feedback
GROUP BY article_id
HAVING distinct_categories > 1
ORDER BY last_feedback_time DESC;

-- Outlier counts: articles with extreme LLM scores
CREATE VIEW IF NOT EXISTS outlier_scores AS
SELECT
    article_id,
    MAX(score) AS max_score,
    MIN(score) AS min_score,
    MAX(score) - MIN(score) AS score_range,
    COUNT(*) AS score_count
FROM llm_scores
GROUP BY article_id
HAVING score_range > 1.0 -- threshold for outlier detection
ORDER BY score_range DESC;
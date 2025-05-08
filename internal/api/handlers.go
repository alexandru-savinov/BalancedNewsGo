package api

// Removed duplicated/older handler definitions.
// The active handlers are defined and registered in internal/api/api.go

// This function seems to be from an older version or a test file?
// If it's used, it needs access to a config.
/*
func processScoresForArticle(dbConn *sqlx.DB, articleID int64) (float64, float64, error) {
	scores, err := db.FetchLLMScores(dbConn, articleID)
	if err != nil {
		return 0, 0, err
	}

	// We need a config here!
	// Option 1: Load it? Might be inefficient.
	// Option 2: Pass it in? Requires changing function signature.
	cfg, err := llm.LoadCompositeScoreConfig() // Example: Load it
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load config: %w", err)
	}

	calculator := &llm.DefaultScoreCalculator{} // Remove Config init
	return calculator.CalculateScore(scores, cfg) // Pass cfg
}
*/

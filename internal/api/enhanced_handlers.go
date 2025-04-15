package api

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// EnhancedReanalyzeHandler is an improved version of reanalyzeHandler with better error handling
func EnhancedReanalyzeHandler(llmClient *llm.EnhancedLLMClient, dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start performance tracking
		start := time.Now()
		defer LogPerformance("EnhancedReanalyzeHandler", start)

		// Parse article ID
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			LogError("EnhancedReanalyzeHandler", fmt.Errorf("invalid article ID: %s", idStr))
			RespondError(c, StatusBadRequest, "Invalid article ID")
			return
		}
		articleID := int64(id)

		// Check if article exists
		article, err := db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			if err == db.ErrArticleNotFound {
				RespondError(c, StatusNotFound, "Article not found")
				return
			}
			LogError("EnhancedReanalyzeHandler", fmt.Errorf("failed to fetch article %d: %w", articleID, err))
			RespondError(c, StatusInternalServerError, "Failed to fetch article")
			return
		}

		// Parse request body for options
		var options struct {
			ForceProvider string `json:"force_provider"`
			ModelName     string `json:"model_name"`
		}
		if err := c.ShouldBindJSON(&options); err != nil {
			// Non-fatal, just log it
			LogWarning("EnhancedReanalyzeHandler", fmt.Sprintf("Invalid JSON body: %v", err))
		}

		// Set initial progress: Queued
		setProgress(articleID, "Queued", "Scoring job queued", 0, "InProgress", "", nil)

		// Respond to client immediately
		RespondSuccess(c, map[string]interface{}{
			"status":     "queued",
			"article_id": articleID,
			"message":    "Scoring job started",
		})

		// Start background processing
		go func() {
			// Use recover for panics
			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("Internal panic: %v", r)
					log.Printf("[EnhancedReanalyzeHandler %d] Recovered from panic: %s", articleID, errMsg)
					setProgress(articleID, "Error", "Internal error occurred", 0, "Error", errMsg, nil)
				}
			}()

			// Set progress: Starting
			setProgress(articleID, "Starting", "Initializing scoring process", 10, "InProgress", "", nil)

			// Perform the ensemble analysis
			compositeScore, err := llmClient.EnhancedEnsembleAnalyze(articleID, article.Content)

			// Handle errors
			if err != nil {
				// Check if it's a provider error
				if providerErr, ok := err.(*llm.ProviderError); ok {
					// Handle rate limiting specially
					if providerErr.IsRateLimitError() {
						resetTime, hasReset := providerErr.GetRateLimitReset()
						resetMsg := ""
						if hasReset {
							resetMsg = fmt.Sprintf(" Rate limit will reset at: %s", resetTime.Format(time.RFC3339))
						}

						errMsg := fmt.Sprintf("Rate limit exceeded.%s", resetMsg)
						log.Printf("[EnhancedReanalyzeHandler %d] %s", articleID, errMsg)

						// Create metadata for the progress update
						metadata := map[string]interface{}{
							"provider_name": providerErr.Provider,
							"reset_time":    resetTime.Format(time.RFC3339),
						}

						setProgress(articleID, "Rate Limited", errMsg, 100, "Error", errMsg, metadata)
						return
					}

					// Handle other provider errors
					errMsg := fmt.Sprintf("Provider error: %s", providerErr.Message)
					log.Printf("[EnhancedReanalyzeHandler %d] %s", articleID, errMsg)
					setProgress(articleID, "Provider Error", errMsg, 100, "Error", errMsg, providerErr.AsMap())
					return
				}

				// Handle generic errors
				errMsg := fmt.Sprintf("Error during analysis: %v", err)
				log.Printf("[EnhancedReanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, 100, "Error", errMsg, nil)
				return
			}

			// Success case
			log.Printf("[EnhancedReanalyzeHandler %d] Analysis completed successfully with score: %.2f",
				articleID, compositeScore)

			// Update the progress with the final score
			setProgress(articleID, "Complete", "Analysis completed successfully", 100, "Success", "", map[string]interface{}{
				"final_score": compositeScore,
			})
		}()
	}
}

// Helper function to calculate percentage
func percent(step, total int) int {
	return int((float64(step) / float64(total)) * 100)
}

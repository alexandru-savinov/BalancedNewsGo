package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors" // Fixed import
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Fixed version of reanalyzeHandler that allows direct score updates
func reanalyzeHandlerFixed(llmClient *llm.LLMClient, dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, apperrors.New("Invalid article ID", ErrValidation))
			LogError("reanalyzeHandler: invalid id", err)
			return
		}
		articleID := int64(id)

		// Check if article exists
		article, err := db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
				return
			}
			RespondError(c, apperrors.New("Failed to fetch article", ErrInternal))
			LogError("reanalyzeHandler: failed to fetch article", err)
			return
		}

		// Parse raw JSON body to check for score field
		var raw map[string]interface{}
		if err := c.ShouldBindJSON(&raw); err != nil {
			RespondError(c, apperrors.New("Invalid JSON body", ErrValidation))
			LogError("reanalyzeHandler: invalid JSON body", err)
			return
		}

		// Check if score field is present and validate it
		if scoreVal, hasScore := raw["score"]; hasScore {
			// Validate the score is within range
			scoreFloat, ok := scoreVal.(float64)
			if !ok || scoreFloat < -1.0 || scoreFloat > 1.0 {
				RespondError(c, apperrors.New("Score must be a number between -1.0 and 1.0", ErrValidation))
				LogError("reanalyzeHandler: invalid score value", nil)
				return
			}

			// If score is valid, update the article score directly and return
			confidence := 1.0 // Use maximum confidence for direct score updates
			err = db.UpdateArticleScoreLLM(dbConn, articleID, scoreFloat, confidence)
			if err != nil {
				RespondError(c, apperrors.New("Failed to update article score", ErrInternal))
				LogError("reanalyzeHandler: failed to update article score", err)
				return
			}

			// Return success response for direct score update
			RespondSuccess(c, map[string]interface{}{
				"status":     "score updated",
				"article_id": articleID,
				"score":      scoreFloat,
			})
			return
		}

		// API-first: Pre-flight LLM provider check (fail fast if unavailable)
		// Use the first model from config for a dry-run health check
		cfg, cfgErr := llm.LoadCompositeScoreConfig()
		if cfgErr != nil || len(cfg.Models) == 0 {
			RespondError(c, apperrors.New("LLM provider configuration unavailable", ErrInternal))
			LogError("reanalyzeHandler: LLM config unavailable", cfgErr)
			return
		}
		modelName := cfg.Models[0].ModelName
		// Set a short timeout for the pre-flight check
		originalTimeout := 10 * time.Second // Default/fallback
		llmClient.SetHTTPLLMTimeout(2 * time.Second)
		_, healthErr := llmClient.ScoreWithModel(article, modelName) // Corrected: pass article directly (it's already a pointer)
		llmClient.SetHTTPLLMTimeout(originalTimeout)
		if healthErr != nil {
			errMsg := healthErr.Error()
			if strings.Contains(errMsg, "503") || strings.Contains(errMsg, "Service Unavailable") {
				RespondError(c, apperrors.New("LLM provider unavailable (503)", ErrInternal))
				LogError("reanalyzeHandler: LLM provider unavailable (503)", healthErr)
				return
			}
			if strings.Contains(errMsg, "rate limit") {
				RespondError(c, apperrors.New("LLM provider rate limited", ErrInternal))
				LogError("reanalyzeHandler: LLM provider rate limited", healthErr)
				return
			}
			RespondError(c, apperrors.New("LLM provider error: "+errMsg, ErrInternal))
			LogError("reanalyzeHandler: LLM provider error", healthErr)
			return
		}

		// Set initial progress: Queued
		setProgress(articleID, "Queued", "Scoring job queued", 0, "InProgress", "", nil)

		go func() {
			// Use recover for panics
			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("Internal panic: %v", r)
					log.Printf("[reanalyzeHandler %d] Recovered from panic: %s", articleID, errMsg)
					setProgress(articleID, "Error", "Internal error occurred", 0, "Error", errMsg, nil)
				}
			}()

			// Set progress: Starting
			setProgress(articleID, "Starting", "Starting scoring process", 0, "InProgress", "", nil)

			// Load configuration
			cfg, err := llm.LoadCompositeScoreConfig()
			if err != nil {
				errMsg := fmt.Sprintf("Failed to load scoring config: %v", err)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, 0, "Error", errMsg, nil)
				return
			}

			// Load article
			article, err := llmClient.GetArticle(articleID)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to load article: %v", err)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, 0, "Error", errMsg, nil)
				return
			}

			// Define total steps for progress calculation
			totalSteps := len(cfg.Models) + 3 // +1 delete, +N models, +1 calculate, +1 store
			stepNum := 1

			// Step 1: Delete old scores
			setProgress(articleID, "Preparing", "Deleting old scores", percent(stepNum, totalSteps), "InProgress", "", nil)
			err = llmClient.DeleteScores(articleID)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to delete old scores: %v", err)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			stepNum++

			// Step 2: Score with each model
			var anySuccess bool
			for _, m := range cfg.Models {
				label := fmt.Sprintf("Scoring with %s", m.ModelName)
				setProgress(articleID, label, label, percent(stepNum, totalSteps), "InProgress", "", nil)

				_, scoreErr := llmClient.ScoreWithModel(&article, m.ModelName)
				log.Printf("[reanalyzeHandler %d] Model %s scoring result: err=%v", articleID, m.ModelName, scoreErr)

				if scoreErr != nil {
					log.Printf("[reanalyzeHandler %d] Actual error received from ScoreWithModel for model %s: (%T) %v", articleID, m.ModelName, scoreErr, scoreErr)
					log.Printf("[reanalyzeHandler %d] Error scoring with model %s, continuing with available models: %v", articleID, m.ModelName, scoreErr)
					// Do not return, just log and continue
				} else {
					anySuccess = true
				}
				stepNum++
			}
			if !anySuccess {
				// If all models failed, abort
				errMsg := "All LLM models failed to score the article. No scores available."
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			log.Printf("[reanalyzeHandler %d] Scoring loop finished (at least one model succeeded).", articleID)

			// Step 3: Fetch scores and Calculate Final Composite Score
			setProgress(articleID, "Calculating", "Fetching scores for final calculation", percent(stepNum, totalSteps), "InProgress", "", nil)
			scores, fetchErr := llmClient.FetchScores(articleID) // Corrected: Use exported method from LLMClient
			if fetchErr != nil {
				errMsg := fmt.Sprintf("Failed to fetch scores for calculation: %v", fetchErr)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			finalScoreValue, _, calcErr := llm.ComputeCompositeScoreWithConfidenceFixed(scores)
			if calcErr != nil {
				errMsg := fmt.Sprintf("Failed to calculate final score: %v", calcErr)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			log.Printf("[reanalyzeHandler %d] Calculated final score: %f", articleID, finalScoreValue)
			stepNum++

			// Step 4: Store ensemble score (Note: Ensure StoreEnsembleScore uses the calculated score if needed, or update article object)
			// Assuming StoreEnsembleScore implicitly uses the latest scores from DB or updates the article object passed to it.
			// If StoreEnsembleScore needs the calculated value explicitly, the call needs modification.
			log.Printf("[reanalyzeHandler %d] Attempting to store ensemble score.", articleID)
			actualFinalScore, storeErr := llmClient.StoreEnsembleScore(&article) // Capture both return values
			if storeErr != nil {
				errMsg := fmt.Sprintf("Error storing ensemble score: %v", storeErr)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				// Even if storing failed, report the score that was calculated before the failure
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, &actualFinalScore)
				return
			}
			// Send "Storing results" message AFTER successful storage
			setProgress(articleID, "Storing results", "Storing ensemble score", percent(stepNum, totalSteps), "InProgress", "", nil)
			// stepNum++ // No need to increment stepNum here, as the next step is the final one (100%)

			// Step 5: Final success step
			log.Printf("[reanalyzeHandler %d] Scoring complete. Final score reported: %f", articleID, actualFinalScore) // Log the score being reported
			setProgress(articleID, "Complete", "Scoring complete", 100, "Success", "", &actualFinalScore)               // Use actualFinalScore
		}()

		RespondSuccess(c, map[string]interface{}{
			"status":     "reanalyze queued",
			"article_id": articleID,
		})
	}
}

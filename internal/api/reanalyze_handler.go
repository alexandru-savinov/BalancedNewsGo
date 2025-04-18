package api

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// reanalyzeHandler handles article rescoring requests
func reanalyzeHandler(llmClient *llm.LLMClient, dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse article ID
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, errors.New("Invalid article ID", errors.ErrValidation))
			return
		}
		articleID := int64(id)

		// Check if article exists
		article, err := db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			if err == db.ErrArticleNotFound {
				RespondError(c, errors.New("Article not found", errors.ErrNotFound))
				return
			}
			RespondError(c, errors.HandleError(err, "failed to fetch article"))
			return
		}

		// Parse raw JSON body to check for forbidden fields
		var raw map[string]interface{}
		if err := c.ShouldBindJSON(&raw); err != nil {
			RespondError(c, errors.HandleError(err, "invalid request body"))
			return
		}

		// Check for forbidden score field
		if _, hasScore := raw["score"]; hasScore {
			RespondError(c, errors.New("Payload must not contain 'score' field", errors.ErrValidation))
			return
		}

		// Pre-flight LLM provider check
		cfg, err := llm.LoadCompositeScoreConfig()
		if err != nil || len(cfg.Models) == 0 {
			RespondError(c, errors.HandleError(err, "LLM provider configuration unavailable"))
			return
		}

		// Health check with first model
		modelName := cfg.Models[0].ModelName
		originalTimeout := 10 * time.Second
		llmClient.SetHTTPLLMTimeout(2 * time.Second)
		_, err = llmClient.ScoreWithModel(*article, modelName) // Note the dereferencing here
		llmClient.SetHTTPLLMTimeout(originalTimeout)

		if err != nil {
			var appErr *errors.AppError
			if err == llm.ErrBothLLMKeysRateLimited {
				appErr = errors.New("LLM provider rate limited", errors.ErrRateLimit)
			} else if err == llm.ErrLLMServiceUnavailable {
				appErr = errors.New("LLM provider unavailable", errors.ErrLLMService)
			} else {
				appErr = errors.HandleError(err, "LLM provider error")
			}
			RespondError(c, appErr)
			return
		}

		// Set initial progress: Queued
		setProgress(articleID, "Queued", "Scoring job queued", 0, "InProgress", "", nil)

		// Process article scoring in background
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("Internal panic: %v", r)
					log.Printf("[reanalyzeHandler %d] Recovered from panic: %s", articleID, errMsg)
					setProgress(articleID, "Error", "Internal error occurred", 0, "Error", errMsg, nil)
				}
			}()

			processArticleScoring(article, llmClient, dbConn)
		}()

		RespondSuccess(c, map[string]interface{}{
			"status":     "reanalyze queued",
			"article_id": articleID,
		})
	}
}

func processArticleScoring(article *db.Article, llmClient *llm.LLMClient, dbConn *sqlx.DB) {
	articleID := article.ID
	stepNum := 1
	totalSteps := 4 // Delete scores, Score with models, Calculate final, Store result

	// Step 1: Delete existing scores
	err := db.DeleteLLMScores(dbConn, articleID)
	if err != nil {
		appErr := errors.HandleError(err, "failed to delete existing scores")
		setProgress(articleID, "Error", appErr.Message, percent(stepNum, totalSteps), "Error", appErr.Error(), nil)
		LogError("processArticleScoring", appErr)
		return
	}
	stepNum++

	// Step 2: Score with each model
	cfg, err := llm.LoadCompositeScoreConfig()
	if err != nil {
		appErr := errors.HandleError(err, "failed to load LLM configuration")
		setProgress(articleID, "Error", appErr.Message, percent(stepNum, totalSteps), "Error", appErr.Error(), nil)
		LogError("processArticleScoring", appErr)
		return
	}

	anySuccess := false
	for _, m := range cfg.Models {
		setProgress(articleID, "Scoring", fmt.Sprintf("Using model %s", m.ModelName), percent(stepNum, totalSteps), "InProgress", "", nil)

		_, err := llmClient.ScoreWithModel(article, m.ModelName)
		if err != nil {
			log.Printf("[processArticleScoring %d] Error scoring with model %s: %v", articleID, m.ModelName, err)
			// Continue with other models unless it's a rate limit error
			if strings.Contains(err.Error(), "rate limit") {
				appErr := errors.New("Rate limit error", errors.ErrRateLimit)
				setProgress(articleID, "Error", appErr.Message, percent(stepNum, totalSteps), "Error", appErr.Error(), nil)
				LogError("processArticleScoring", appErr)
				return
			}
		} else {
			anySuccess = true
		}
	}

	if !anySuccess {
		appErr := errors.New("All models failed to score article", errors.ErrLLMService)
		setProgress(articleID, "Error", appErr.Message, percent(stepNum, totalSteps), "Error", appErr.Error(), nil)
		LogError("processArticleScoring", appErr)
		return
	}
	stepNum++

	// Step 3: Calculate final composite score
	setProgress(articleID, "Calculating", "Computing final score", percent(stepNum, totalSteps), "InProgress", "", nil)
	scores, err := db.FetchLLMScores(dbConn, articleID)
	if err != nil {
		appErr := errors.HandleError(err, "failed to fetch scores for calculation")
		setProgress(articleID, "Error", appErr.Message, percent(stepNum, totalSteps), "Error", appErr.Error(), nil)
		LogError("processArticleScoring", appErr)
		return
	}

	finalScore, confidence, err := llm.ComputeCompositeScoreWithConfidence(scores)
	if err != nil {
		appErr := errors.HandleError(err, "failed to calculate final score")
		setProgress(articleID, "Error", appErr.Message, percent(stepNum, totalSteps), "Error", appErr.Error(), nil)
		LogError("processArticleScoring", appErr)
		return
	}
	stepNum++

	// Step 4: Store final result
	err = db.UpdateArticleScore(dbConn, articleID, finalScore, confidence)
	if err != nil {
		appErr := errors.HandleError(err, "failed to store final score")
		setProgress(articleID, "Error", appErr.Message, percent(stepNum, totalSteps), "Error", appErr.Error(), &finalScore)
		LogError("processArticleScoring", appErr)
		return
	}

	// Success
	setProgress(articleID, "Complete", "Scoring complete", 100, "Success", "", &finalScore)
}

func percent(step, total int) int {
	return (step * 100) / total
}

type ProgressUpdate struct {
	State      string   `json:"state"`
	Message    string   `json:"message"`
	Progress   int      `json:"progress"`
	Status     string   `json:"status"`
	Error      string   `json:"error,omitempty"`
	FinalScore *float64 `json:"final_score,omitempty"`
}

func setProgress(articleID int64, state string, message string, progress int, status string, errorMsg string, finalScore *float64) {
	update := ProgressUpdate{
		State:      state,
		Message:    message,
		Progress:   progress,
		Status:     status,
		Error:      errorMsg,
		FinalScore: finalScore,
	}
	log.Printf("[Progress %d] %s: %s (%d%%)", articleID, state, message, progress)
	// Additional progress tracking logic can be added here
}

package api

import (
	"fmt"
	"strconv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
)

// getArticleByIDHandlerWithDB returns a handler function for fetching an article by ID
func getArticleByIDHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get article ID from path parameter
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			RespondError(c, NewAppError(ErrValidation, "Invalid article ID"))
			return
		}

		// Fetch article from database
		article, err := dbOps.FetchArticleByID(nil, id)
		if err != nil {
			if err == db.ErrArticleNotFound {
				RespondError(c, NewAppError(ErrNotFound, "Article not found"))
			} else {
				RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article"))
			}
			return
		}

		// Fetch LLM scores
		scores, err := dbOps.FetchLLMScores(nil, id)
		if err != nil {
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch LLM scores"))
			return
		}

		// Use the DefaultScoreCalculator to get composite score and confidence
		calculator := &llm.DefaultScoreCalculator{
			Config: &llm.CompositeScoreConfig{
				MinScore:       -1.0,
				MaxScore:       1.0,
				DefaultMissing: 0.0,
			},
		}
		compositeScore, confidence, err := calculator.CalculateScore(scores)
		if err != nil {
			// Log error but continue with default values
			fmt.Printf("Error calculating composite score: %v\n", err)
			compositeScore = 0.0
			confidence = 0.0
		}

		// Prepare response with article details and scores
		response := map[string]interface{}{
			"id":              article.ID,
			"title":           article.Title,
			"content":         article.Content,
			"url":             article.URL,
			"source":          article.Source,
			"pub_date":        article.PubDate,
			"created_at":      article.CreatedAt,
			"composite_score": compositeScore,
			"confidence":      confidence,
		}

		// Include individual model scores
		modelScores := make([]map[string]interface{}, 0, len(scores))
		for _, score := range scores {
			if score.Model != "ensemble" {
				modelScores = append(modelScores, map[string]interface{}{
					"model":      score.Model,
					"score":      score.Score,
					"metadata":   score.Metadata,
					"created_at": score.CreatedAt,
				})
			}
		}
		response["model_scores"] = modelScores

		RespondSuccess(c, response)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// Legacy handlers for server-rendered HTML pages
// These will be used when the --legacy-html flag is enabled

func legacyArticlesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		source := c.Query("source")
		leaning := c.Query("leaning")

		limit := 20

		offset := 0
		if o := c.Query("offset"); o != "" {
			if _, err := fmt.Sscanf(o, "%d", &offset); err != nil {
				offset = 0
			}
		}

		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		if err != nil {
			c.String(500, "Error fetching articles")
			return
		}

		html := ""
		for _, a := range articles {
			// Fetch scores for this article
			scores, err := db.FetchLLMScores(dbConn, a.ID)
			var compositeScore float64
			var avgConfidence float64
			if err == nil && len(scores) > 0 {
				var weightedSum, sumWeights float64
				for _, s := range scores {
					var meta struct {
						Confidence float64 `json:"confidence"`
					}
					_ = json.Unmarshal([]byte(s.Metadata), &meta)
					weightedSum += s.Score * meta.Confidence
					sumWeights += meta.Confidence
				}
				if sumWeights > 0 {
					compositeScore = weightedSum / sumWeights
					avgConfidence = sumWeights / float64(len(scores))
				}
			}

			html += `<div>
				<h3>
					<a href="/article/` + strconv.FormatInt(a.ID, 10) + `"
					   hx-get="/article/` + strconv.FormatInt(a.ID, 10) + `"
					   hx-target="#articles" hx-swap="innerHTML">` + a.Title + `</a>
				</h3>
				<p>` + a.Source + ` | ` + a.PubDate.Format("2006-01-02") + `</p>
				<p>Score: ` + fmt.Sprintf("%.2f", compositeScore) + ` | Confidence: ` + fmt.Sprintf("%.0f%%", avgConfidence*100) + `</p>
			</div>`
		}

		c.Header("Content-Type", "text/html")
		c.String(200, html)
	}
}

func legacyArticleDetailHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid article ID")
			return
		}

		article, err := db.FetchArticleByID(dbConn, id)
		if err != nil {
			c.String(http.StatusNotFound, "Article not found")
			return
		}

		scores, err := db.FetchLLMScores(dbConn, article.ID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch scores")
			return
		}

		var compositeScore float64
		var avgConfidence float64
		if len(scores) > 0 {
			var weightedSum, sumWeights float64
			for _, s := range scores {
				var meta struct {
					Confidence float64 `json:"confidence"`
				}
				_ = json.Unmarshal([]byte(s.Metadata), &meta)
				weightedSum += s.Score * meta.Confidence
				sumWeights += meta.Confidence
			}
			if sumWeights > 0 {
				compositeScore = weightedSum / sumWeights
				avgConfidence = sumWeights / float64(len(scores))
			}
		}

		c.HTML(http.StatusOK, "article.html", gin.H{
			"Article":        article,
			"CompositeScore": compositeScore,
			"Confidence":     avgConfidence,
		})
	}
}

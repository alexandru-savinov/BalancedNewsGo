package api

import (
	"net/http"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// debugSchemaHandler returns the database schema for debugging purposes
func debugSchemaHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get table names
		var tables []string
		err := dbConn.Select(&tables, "SELECT name FROM sqlite_master WHERE type='table'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success":       false,
				"error_message": "Failed to get table names: " + err.Error(),
			})
			return
		}

		// Get schema for each table
		schemas := make(map[string][]map[string]interface{})
		for _, table := range tables {
			var columns []map[string]interface{}
			err := dbConn.Select(&columns, "PRAGMA table_info("+table+")")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success":       false,
					"error_message": "Failed to get schema for table " + table + ": " + err.Error(),
				})
				return
			}
			schemas[table] = columns
		}

		// Get sample data for feedback table
		var feedbackSamples []db.Feedback
		err = dbConn.Select(&feedbackSamples, "SELECT * FROM feedback LIMIT 5")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success":       false,
				"error_message": "Failed to get feedback samples: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"tables":           tables,
				"schemas":          schemas,
				"feedback_samples": feedbackSamples,
			},
		})
	}
}

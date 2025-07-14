package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// getSourcesHandler handles GET /api/sources
// @Summary Get all sources
// @Description Get a list of all sources with optional filtering and pagination
// @Tags Sources
// @Accept json
// @Produce json
// @Param enabled query boolean false "Filter by enabled status"
// @Param channel_type query string false "Filter by channel type"
// @Param category query string false "Filter by category (left, center, right)"
// @Param include_stats query boolean false "Include source statistics"
// @Param limit query integer false "Number of items per page (default: 50, max: 100)"
// @Param offset query integer false "Pagination offset (default: 0)"
// @Success 200 {object} StandardResponse{data=models.SourceListResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/sources [get]
func getSourcesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse query parameters
		var enabled *bool
		if enabledStr := c.Query("enabled"); enabledStr != "" {
			if enabledVal, err := strconv.ParseBool(enabledStr); err == nil {
				enabled = &enabledVal
			} else {
				RespondError(c, NewAppError(ErrValidation, "Invalid enabled parameter"))
				return
			}
		}

		channelType := c.Query("channel_type")
		category := c.Query("category")

		// Parse pagination parameters
		limit := 50 // default
		if limitStr := c.Query("limit"); limitStr != "" {
			if limitVal, err := strconv.Atoi(limitStr); err == nil && limitVal > 0 {
				if limitVal > 100 {
					limitVal = 100 // max limit
				}
				limit = limitVal
			}
		}

		offset := 0 // default
		if offsetStr := c.Query("offset"); offsetStr != "" {
			if offsetVal, err := strconv.Atoi(offsetStr); err == nil && offsetVal >= 0 {
				offset = offsetVal
			}
		}

		// Validate category if provided
		if category != "" && !models.IsValidCategory(category) {
			RespondError(c, NewAppError(ErrValidation, "Invalid category"))
			return
		}

		// Validate channel type if provided
		if channelType != "" && !models.IsValidChannelType(channelType) {
			RespondError(c, NewAppError(ErrValidation, "Invalid channel type"))
			return
		}

		// Fetch sources from database
		sources, err := db.FetchSources(dbConn, enabled, channelType, category, limit, offset)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch sources"))
			return
		}

		// Convert to response format
		sourcesWithStats := make([]models.SourceWithStats, len(sources))
		for i, source := range sources {
			sourcesWithStats[i] = models.SourceWithStats{
				Source: models.Source{
					ID:            source.ID,
					Name:          source.Name,
					ChannelType:   source.ChannelType,
					FeedURL:       source.FeedURL,
					Category:      source.Category,
					Enabled:       source.Enabled,
					DefaultWeight: source.DefaultWeight,
					LastFetchedAt: source.LastFetchedAt,
					ErrorStreak:   source.ErrorStreak,
					Metadata:      source.Metadata,
					CreatedAt:     source.CreatedAt,
					UpdatedAt:     source.UpdatedAt,
				},
			}

			// TODO: Add stats if requested in future enhancement
		}

		// Get total count for pagination (simplified - using current result count)
		total := int64(len(sources))

		response := models.SourceListResponse{
			Sources: sourcesWithStats,
			Total:   total,
			Limit:   limit,
			Offset:  offset,
		}

		RespondSuccess(c, response)
	}
}

// createSourceHandler handles POST /api/sources
// @Summary Create a new source
// @Description Create a new news source
// @Tags Sources
// @Accept json
// @Produce json
// @Param source body models.CreateSourceRequest true "Source object"
// @Success 201 {object} StandardResponse{data=models.Source}
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/sources [post]
func createSourceHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateSourceRequest
		if err := c.ShouldBind(&req); err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid request body: "+err.Error()))
			return
		}

		// Validate request
		if err := req.Validate(); err != nil {
			RespondError(c, NewAppError(ErrValidation, err.Error()))
			return
		}

		// Check if source name already exists
		exists, err := db.SourceExistsByName(dbConn, req.Name)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to check source existence"))
			return
		}
		if exists {
			RespondError(c, NewAppError(ErrConflict, "Source with this name already exists"))
			return
		}

		// Set default weight if not provided
		if req.DefaultWeight == 0 {
			req.DefaultWeight = 1.0
		}

		// Create source
		source := &db.Source{
			Name:          req.Name,
			ChannelType:   req.ChannelType,
			FeedURL:       req.FeedURL,
			Category:      req.Category,
			Enabled:       true, // New sources are enabled by default
			DefaultWeight: req.DefaultWeight,
			ErrorStreak:   0,
			Metadata:      req.Metadata,
		}

		id, err := db.InsertSource(dbConn, source)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to create source"))
			return
		}

		// Fetch the created source to return complete data
		createdSource, err := db.FetchSourceByID(dbConn, id)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch created source"))
			return
		}

		// Convert to response model
		responseSource := models.Source{
			ID:            createdSource.ID,
			Name:          createdSource.Name,
			ChannelType:   createdSource.ChannelType,
			FeedURL:       createdSource.FeedURL,
			Category:      createdSource.Category,
			Enabled:       createdSource.Enabled,
			DefaultWeight: createdSource.DefaultWeight,
			LastFetchedAt: createdSource.LastFetchedAt,
			ErrorStreak:   createdSource.ErrorStreak,
			Metadata:      createdSource.Metadata,
			CreatedAt:     createdSource.CreatedAt,
			UpdatedAt:     createdSource.UpdatedAt,
		}

		c.JSON(http.StatusCreated, StandardResponse{
			Success: true,
			Data:    responseSource,
		})
	}
}

// getSourceByIDHandler handles GET /api/sources/:id
// @Summary Get source by ID
// @Description Get a specific source by its ID
// @Tags Sources
// @Accept json
// @Produce json
// @Param id path integer true "Source ID"
// @Param include_stats query boolean false "Include source statistics"
// @Success 200 {object} StandardResponse{data=models.SourceWithStats}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/sources/{id} [get]
func getSourceByIDHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid source ID"))
			return
		}

		// Fetch source from database
		source, err := db.FetchSourceByID(dbConn, id)
		if err != nil {
			if err.Error() == "source not found" {
				RespondError(c, NewAppError(ErrNotFound, "Source not found"))
				return
			}
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch source"))
			return
		}

		// Convert to response model
		responseSource := models.SourceWithStats{
			Source: models.Source{
				ID:            source.ID,
				Name:          source.Name,
				ChannelType:   source.ChannelType,
				FeedURL:       source.FeedURL,
				Category:      source.Category,
				Enabled:       source.Enabled,
				DefaultWeight: source.DefaultWeight,
				LastFetchedAt: source.LastFetchedAt,
				ErrorStreak:   source.ErrorStreak,
				Metadata:      source.Metadata,
				CreatedAt:     source.CreatedAt,
				UpdatedAt:     source.UpdatedAt,
			},
		}

		// TODO: Add stats in future enhancement

		RespondSuccess(c, responseSource)
	}
}

// updateSourceHandler handles PUT /api/sources/:id
// @Summary Update source
// @Description Update an existing source
// @Tags Sources
// @Accept json
// @Produce json
// @Param id path integer true "Source ID"
// @Param source body models.UpdateSourceRequest true "Source update object"
// @Success 200 {object} StandardResponse{data=models.Source}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/sources/{id} [put]
func updateSourceHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid source ID"))
			return
		}

		var req models.UpdateSourceRequest
		if err := c.ShouldBind(&req); err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid request body: "+err.Error()))
			return
		}

		// Validate request
		if err := req.Validate(); err != nil {
			RespondError(c, NewAppError(ErrValidation, err.Error()))
			return
		}

		// Check if source exists
		_, err = db.FetchSourceByID(dbConn, id)
		if err != nil {
			if err.Error() == "source not found" {
				RespondError(c, NewAppError(ErrNotFound, "Source not found"))
				return
			}
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch source"))
			return
		}

		// Check if name already exists (if name is being updated)
		if req.Name != nil {
			exists, err := db.SourceExistsByName(dbConn, *req.Name)
			if err != nil {
				RespondError(c, NewAppError(ErrInternal, "Failed to check source name existence"))
				return
			}
			if exists {
				// Check if it's the same source (allowed) or different source (conflict)
				existingSource, err := db.FetchSourceByID(dbConn, id)
				if err != nil {
					RespondError(c, NewAppError(ErrInternal, "Failed to fetch existing source"))
					return
				}
				if existingSource.Name != *req.Name {
					RespondError(c, NewAppError(ErrConflict, "Source with this name already exists"))
					return
				}
			}
		}

		// Convert request to update map
		updates := req.ToUpdateMap()
		if len(updates) == 0 {
			RespondError(c, NewAppError(ErrValidation, "No updates provided"))
			return
		}

		// Update source
		err = db.UpdateSource(dbConn, id, updates)
		if err != nil {
			if err.Error() == "source not found" {
				RespondError(c, NewAppError(ErrNotFound, "Source not found"))
				return
			}
			RespondError(c, NewAppError(ErrInternal, "Failed to update source"))
			return
		}

		// Fetch updated source
		updatedSource, err := db.FetchSourceByID(dbConn, id)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch updated source"))
			return
		}

		// Convert to response model
		responseSource := models.Source{
			ID:            updatedSource.ID,
			Name:          updatedSource.Name,
			ChannelType:   updatedSource.ChannelType,
			FeedURL:       updatedSource.FeedURL,
			Category:      updatedSource.Category,
			Enabled:       updatedSource.Enabled,
			DefaultWeight: updatedSource.DefaultWeight,
			LastFetchedAt: updatedSource.LastFetchedAt,
			ErrorStreak:   updatedSource.ErrorStreak,
			Metadata:      updatedSource.Metadata,
			CreatedAt:     updatedSource.CreatedAt,
			UpdatedAt:     updatedSource.UpdatedAt,
		}

		RespondSuccess(c, responseSource)
	}
}

// deleteSourceHandler handles DELETE /api/sources/:id
// @Summary Delete source (soft delete)
// @Description Disable a source (soft delete)
// @Tags Sources
// @Accept json
// @Produce json
// @Param id path integer true "Source ID"
// @Success 200 {object} StandardResponse{data=string}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/sources/{id} [delete]
func deleteSourceHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid source ID"))
			return
		}

		// Check if source exists
		_, err = db.FetchSourceByID(dbConn, id)
		if err != nil {
			if err.Error() == "source not found" {
				RespondError(c, NewAppError(ErrNotFound, "Source not found"))
				return
			}
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch source"))
			return
		}

		// Soft delete (disable) the source
		err = db.SoftDeleteSource(dbConn, id)
		if err != nil {
			if err.Error() == "source not found" {
				RespondError(c, NewAppError(ErrNotFound, "Source not found"))
				return
			}
			RespondError(c, NewAppError(ErrInternal, "Failed to delete source"))
			return
		}

		RespondSuccess(c, "Source disabled successfully")
	}
}

// getSourceStatsHandler handles GET /api/sources/:id/stats
// @Summary Get source statistics
// @Description Get detailed statistics for a specific source
// @Tags Sources
// @Accept json
// @Produce json
// @Param id path integer true "Source ID"
// @Success 200 {object} StandardResponse{data=models.SourceStats}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/sources/{id}/stats [get]
func getSourceStatsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid source ID"))
			return
		}

		// Check if source exists
		_, err = db.FetchSourceByID(dbConn, id)
		if err != nil {
			if err.Error() == "source not found" {
				RespondError(c, NewAppError(ErrNotFound, "Source not found"))
				return
			}
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch source"))
			return
		}

		// TODO: Implement source statistics aggregation
		// For now, return placeholder stats
		stats := models.SourceStats{
			SourceID:     id,
			ArticleCount: 0,
			AvgScore:     nil,
			ComputedAt:   time.Now(),
		}

		RespondSuccess(c, stats)
	}
}

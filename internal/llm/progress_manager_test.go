package llm

import (
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewProgressManager(t *testing.T) {
	pm := NewProgressManager(time.Minute)
	assert.NotNil(t, pm)
	assert.NotNil(t, pm.progressMap)
}

func TestProgressManagerSetGet(t *testing.T) {
	pm := NewProgressManager(time.Minute)

	// Set and get for a new ID
	articleID := int64(123)
	progress := &models.ProgressState{
		Step:        "Analyzing content",
		Message:     "Processing article content",
		Percent:     50,
		Status:      "InProgress",
		LastUpdated: time.Now().Unix(),
	}

	pm.SetProgress(articleID, progress)

	// Get the progress
	retrieved := pm.GetProgress(articleID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, progress.Status, retrieved.Status)
	assert.Equal(t, progress.Step, retrieved.Step)
	assert.Equal(t, progress.Percent, retrieved.Percent)

	// Get non-existent progress
	retrieved = pm.GetProgress(int64(999))
	assert.Nil(t, retrieved)

	// Update existing progress
	updatedProgress := &models.ProgressState{
		Step:        "Analysis done",
		Message:     "Processing complete",
		Percent:     100,
		Status:      "Completed",
		LastUpdated: time.Now().Unix(),
	}
	pm.SetProgress(articleID, updatedProgress)

	// Get the updated progress
	retrieved = pm.GetProgress(articleID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, updatedProgress.Status, retrieved.Status)
	assert.Equal(t, updatedProgress.Step, retrieved.Step)
	assert.Equal(t, updatedProgress.Percent, retrieved.Percent)
}

// TestManualCleanup tests the cleanup functionality directly rather than
// waiting for the background routine to run
func TestManualCleanup(t *testing.T) {
	pm := NewProgressManager(time.Hour) // Long interval so auto-cleanup doesn't interfere

	// Add a completed progress entry (should be cleaned up)
	completedID := int64(1)
	completedProgress := &models.ProgressState{
		Step:        "Complete",
		Status:      "Success",
		Percent:     100,
		LastUpdated: time.Now().Add(-10 * time.Minute).Unix(), // Older than 5 minutes
	}
	pm.SetProgress(completedID, completedProgress)

	// Add an error progress entry (should be cleaned up)
	errorID := int64(2)
	errorProgress := &models.ProgressState{
		Step:        "Error",
		Status:      "Error",
		Percent:     50,
		LastUpdated: time.Now().Add(-10 * time.Minute).Unix(), // Older than 5 minutes
	}
	pm.SetProgress(errorID, errorProgress)

	// Add a stale in-progress entry (should be cleaned up)
	staleID := int64(3)
	staleProgress := &models.ProgressState{
		Step:        "Stalled",
		Status:      "InProgress",
		Percent:     25,
		LastUpdated: time.Now().Add(-120 * time.Minute).Unix(), // Older than 30 minutes
	}
	pm.SetProgress(staleID, staleProgress)

	// Add a recent in-progress entry (should remain)
	recentID := int64(4)
	recentProgress := &models.ProgressState{
		Step:        "Fresh",
		Status:      "InProgress",
		Percent:     75,
		LastUpdated: time.Now().Unix(), // Current
	}
	pm.SetProgress(recentID, recentProgress)

	// Manually trigger cleanup
	pm.cleanup()

	// Check what remains
	assert.Nil(t, pm.GetProgress(completedID), "Completed progress should be cleaned up")
	assert.Nil(t, pm.GetProgress(errorID), "Error progress should be cleaned up")
	assert.Nil(t, pm.GetProgress(staleID), "Stale progress should be cleaned up")
	assert.NotNil(t, pm.GetProgress(recentID), "Recent in-progress should remain")
}

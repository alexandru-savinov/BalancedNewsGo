package models

// ProgressState represents the progress of a scoring operation
// @Description Progress state for long-running operations
type ProgressState struct {
	Step         string   `json:"step" example:"Scoring"`               // Current detailed step
	Message      string   `json:"message" example:"Processing article"` // User-friendly message
	Percent      int      `json:"percent" example:"75"`                 // Progress percentage
	Status       string   `json:"status" example:"InProgress"`          // Overall status
	Error        string   `json:"error,omitempty"`                      // Error message if failed
	ErrorDetails string   `json:"error_details,omitempty"`              // Structured error details (JSON string)
	FinalScore   *float64 `json:"final_score,omitempty" example:"0.25"` // Final score if completed
	LastUpdated  int64    `json:"last_updated" example:"1609459200"`    // Timestamp
}

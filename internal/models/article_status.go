package models

// ArticleStatus represents the processing status of an article.
// These constants should be used for the `articles.status` column.
const (
	ArticleStatusPending           = "pending"
	ArticleStatusProcessing        = "processing" // Optional: if we want to mark articles actively being processed
	ArticleStatusScored            = "scored"
	ArticleStatusFailedAllInvalid  = "failed_all_invalid"
	ArticleStatusFailedZeroConf    = "failed_zero_confidence"
	ArticleStatusFailedError       = "failed_error"        // For other generic errors during scoring
	ArticleStatusNeedsManualReview = "needs_manual_review" // Optional: for other types of failures or edge cases
)

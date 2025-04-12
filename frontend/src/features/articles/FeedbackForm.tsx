import React, { useState } from 'react';
import { submitFeedback, FeedbackPayload } from '../../services/feedbackService';
import styles from './FeedbackForm.module.css'; // We'll create this CSS module next

interface FeedbackFormProps {
  articleId: number;
  onSubmitSuccess?: () => void; // Optional callback on successful submission
  onCancel?: () => void; // Optional callback for cancelling
}

const FeedbackForm: React.FC<FeedbackFormProps> = ({ articleId, onSubmitSuccess, onCancel }) => {
  const [comment, setComment] = useState('');
  const [rating, setRating] = useState<number | undefined>(undefined); // Example: 1-5 rating
  const [tags, setTags] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const availableTags = ["Parse Error", "Model Disagreement", "Low Confidence", "Fallback Used", "Other"]; // From productContext.md

  const handleTagChange = (tag: string) => {
    setTags(prevTags =>
      prevTags.includes(tag)
        ? prevTags.filter(t => t !== tag)
        : [...prevTags, tag]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!articleId || isSubmitting) return;

    setIsSubmitting(true);
    setError(null);
    setSuccessMessage(null);

    const payload: FeedbackPayload = {
      article_id: articleId,
      user_comment: comment || undefined, // Send undefined if empty
      rating: rating,
      tags: tags.length > 0 ? tags : undefined, // Send undefined if empty
    };

    try {
      const result = await submitFeedback(payload);
      setSuccessMessage(result.message || 'Feedback submitted successfully!');
      // Clear form after successful submission
      setComment('');
      setRating(undefined);
      setTags([]);
      onSubmitSuccess?.(); // Call the success callback if provided
      // Optionally hide the form after a delay
      // setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err: any) {
      setError(err.message || 'Failed to submit feedback.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className={styles.feedbackForm}>
      <h4>Submit Feedback for Article {articleId}</h4>

      {error && <div className={styles.error}>{error}</div>}
      {successMessage && <div className={styles.success}>{successMessage}</div>}

      <div className={styles.formGroup}>
        <label htmlFor={`feedback-comment-${articleId}`}>Comment (Optional):</label>
        <textarea
          id={`feedback-comment-${articleId}`}
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          rows={3}
          disabled={isSubmitting}
        />
      </div>

      {/* Example: Rating Input (Could be stars, radio buttons, etc.) */}
      <div className={styles.formGroup}>
        <label>Rating (Optional):</label>
        <div>
          {[1, 2, 3, 4, 5].map(r => (
            <button
              key={r}
              type="button"
              onClick={() => setRating(r)}
              className={`${styles.ratingButton} ${rating === r ? styles.selected : ''}`}
              disabled={isSubmitting}
            >
              {r}★
            </button>
          ))}
           {rating && <button type="button" onClick={() => setRating(undefined)} className={styles.clearButton} disabled={isSubmitting}>Clear</button>}
        </div>
      </div>


      <div className={styles.formGroup}>
        <label>Tags (Optional):</label>
        <div className={styles.tagGroup}>
          {availableTags.map(tag => (
            <label key={tag} className={styles.tagLabel}>
              <input
                type="checkbox"
                checked={tags.includes(tag)}
                onChange={() => handleTagChange(tag)}
                disabled={isSubmitting}
              />
              {tag}
            </label>
          ))}
        </div>
      </div>

      <div className={styles.formActions}>
        <button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Submitting...' : 'Submit Feedback'}
        </button>
        {onCancel && (
          <button type="button" onClick={onCancel} disabled={isSubmitting} className={styles.cancelButton}>
            Cancel
          </button>
        )}
      </div>
    </form>
  );
};

export default FeedbackForm;
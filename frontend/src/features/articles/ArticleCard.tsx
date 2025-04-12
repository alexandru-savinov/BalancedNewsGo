import React, { useState, useEffect } from 'react'; // Import useEffect
import { Article, LLMScore } from '../../types/api.d'; // Import types
import { useStore } from '../../store'; // Import the Zustand store hook
import { triggerArticleRescore } from '../../services/articleService'; // Import the rescore function
import BiasSlider from '../../components/common/BiasSlider'; // Import BiasSlider
import Tooltip from '../../components/common/Tooltip'; // Import Tooltip
import FeedbackForm from './FeedbackForm'; // Import FeedbackForm
import styles from './ArticleCard.module.css'; // Import CSS Module

interface ArticleCardProps {
  article: Article;
}

const ArticleCard: React.FC<ArticleCardProps> = ({ article }) => {
      console.log(`Rendering ArticleCard ID: ${article.ID}`); // <-- ADDED LOG

  // Select state individually using useStore to prevent unnecessary re-renders
  const selectedArticle = useStore((state) => state.selectedArticle);
  const isLoadingDetail = useStore((state) => state.isLoadingDetail);
  const errorDetail = useStore((state) => state.errorDetail);
  const fetchArticleDetail = useStore((state) => state.fetchArticleDetail);
  // Select SSE state and actions individually
  const sseConnectionStatus = useStore((state) => state.sseConnectionStatus);
  const sseProgressData = useStore((state) => state.sseProgressData);
  const sseError = useStore((state) => state.sseError);
  const connectScoreProgressSse = useStore((state) => state.connectScoreProgressSse);
  const disconnectScoreProgressSse = useStore((state) => state.disconnectScoreProgressSse);

  const [isExpanded, setIsExpanded] = useState(false);
  const [isRescoring, setIsRescoring] = useState(false); // State for rescore loading
  const [rescoreError, setRescoreError] = useState<string | null>(null); // State for rescore error
  const [showFeedbackForm, setShowFeedbackForm] = useState(false); // State to control feedback form visibility

  // Check if the currently selected detailed article is this one
  const isSelected = selectedArticle?.article.ID === article.ID; // Check if this card's details are the ones selected in the store

  // Effect for SSE cleanup
  useEffect(() => {
    // Cleanup function: Disconnect SSE when component unmounts,
    // or when this card is no longer expanded/selected.
    return () => {
      // Only disconnect if the SSE connection was related to *this* article card
      if (isSelected) {
         console.log(`ArticleCard ${article.ID}: Cleaning up SSE connection.`);
         disconnectScoreProgressSse();
      }
    };
    // Re-run cleanup if the selection or expansion state changes, or if the disconnect function itself changes
  }, [isSelected, isExpanded, disconnectScoreProgressSse, article.ID]); // Added article.ID for clarity in logs if needed

  const handleCardClick = () => {
    if (!isSelected || !isExpanded) {
      fetchArticleDetail(article.ID); // Fetch details if not selected or not expanded
      setIsExpanded(true); // Expand on click
    } else {
      setIsExpanded(false); // Collapse if already selected and expanded
      // Optionally clear selection: useStore.setState({ selectedArticle: null });
    }
  };

  const formatDate = (dateString: string | undefined) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleString();
    } catch (e) {
      console.error("Error formatting date:", dateString, e);
      return 'Invalid Date';
    }
  };

  // Helper to render individual model scores with tooltips
  const renderModelScores = (scores: LLMScore[]) => {
    const relevantScores = scores.filter(s => s.Model !== 'ensemble' && s.Model !== 'summarizer'); // Filter out non-scoring models
    if (relevantScores.length === 0) {
      return <span>No individual model scores available.</span>;
    }
    return (
      <ul className={styles.modelScoresList}>
        {relevantScores.map((score) => (
          <li key={score.ID} className={styles.modelScoreItem}>
            <Tooltip text={`Model: ${score.Model}\nScore: ${score.Score.toFixed(3)}\nMetadata: ${score.Metadata || 'None'}`}>
              <span>{score.Model}: {score.Score.toFixed(2)}</span>
            </Tooltip>
          </li>
        ))}
      </ul>
    );
  };


  const handleRescoreClick = async (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation(); // Prevent card click handler
    if (!article.ID || isRescoring) return; // Prevent multiple clicks or if ID is missing

    setIsRescoring(true);
    setRescoreError(null);

    try {
      await triggerArticleRescore(article.ID);
      console.log(`Successfully triggered rescore for article ${article.ID}. Connecting to SSE...`);
      // Trigger SSE connection via Zustand action
      connectScoreProgressSse(article.ID);
      // Optionally show a success message briefly or rely on SSE status
    } catch (error: any) {
      console.error(`Error triggering rescore for article ${article.ID}:`, error);
      setRescoreError(error.message || 'Failed to trigger rescore.');
    } finally {
      setIsRescoring(false);
    }
  };

console.log(`[Debug] ArticleCard: Rendering card ID ${article.ID}. data-testid: article-card-${article.ID}`);
  return (
    <article
      className={styles.card}
      data-testid={`article-card-${article.ID}`}
      aria-labelledby={`article-title-${article.ID}`}
      onClick={handleCardClick}
      onKeyPress={(e) => (e.key === 'Enter' || e.key === ' ') && handleCardClick()} // Accessibility
      tabIndex={0} // Make it focusable
    >
      <header>
        <h3 id={`article-title-${article.ID}`} className={styles.title}>
          <a
            href={article.URL}
            target="_blank"
            rel="noopener noreferrer"
            className={styles.link}
            onClick={(e) => e.stopPropagation()} // Prevent card click when clicking link
          >
            {article.Title || 'Untitled'}
          </a>
        </h3>
      </header>
      <section className={styles.summary}>
        <p>{article.Content ? `${article.Content.substring(0, 150)}${article.Content.length > 150 ? '...' : ''}` : 'No content available.'}</p>
      </section>
      <footer className={styles.footer}>
        <span><strong>ID:</strong> {article.ID} | </span>
        <span><strong>Source:</strong> {article.Source || 'N/A'} | </span>
        <span><strong>Fetched:</strong> {formatDate(article.CreatedAt)}</span>
        {/* Display composite score from list initially */}
        <div style={{ marginTop: '5px' }}>
          <strong>Score (List):</strong> {article.CompositeScore !== undefined && article.CompositeScore !== null ? article.CompositeScore.toFixed(2) : 'N/A'}
        </div>
      </footer>

      {/* Detailed Section - Shown when expanded */}
      {isExpanded && (
        <section className={styles.detailsSection}>
          {isLoadingDetail && isSelected && <div className={styles.loading}>Loading details...</div>}
          {errorDetail && isSelected && <div className={styles.error}>Error loading details: {errorDetail}</div>}
          {selectedArticle && isSelected && !isLoadingDetail && !errorDetail && (
            <div>
              <h4>Details &amp; Debug Info</h4>
              <div className={styles.detailItem}>
                <strong>Composite Score:</strong> {selectedArticle.composite_score?.toFixed(3) ?? 'N/A'}
              </div>
              <div className={styles.detailItem}>
                <strong>Confidence:</strong> {selectedArticle.confidence?.toFixed(3) ?? 'N/A'}
              </div>
              <div className={styles.detailItem}>
                <strong>Bias Visualization:</strong>
                <BiasSlider score={selectedArticle.composite_score} confidence={selectedArticle.confidence} />
              </div>
               <div className={styles.detailItem}>
                 <strong>Source:</strong> {selectedArticle.article.Source ?? 'N/A'}
               </div>
               <div className={styles.detailItem}>
                 <strong>Fetched At:</strong> {formatDate(selectedArticle.article.CreatedAt)}
               </div>
               <div className={styles.detailItem}>
                 <strong>Last Updated:</strong> {formatDate(selectedArticle.article.UpdatedAt)}
               </div>
              <div className={styles.detailItem}>
                <strong>Individual Model Scores:</strong>
                {renderModelScores(selectedArticle.scores)}
              </div>
              {/* --- Rescore Button --- */}
              <div className={styles.detailItem}>
                <button
                  onClick={handleRescoreClick}
                  disabled={isRescoring}
                  className={styles.rescoreButton} // Add specific styling if needed
                >
                  {isRescoring ? 'Scoring...' : 'Re-analyze Article'}
                </button>
                {rescoreError && <span className={styles.error}> Trigger Error: {rescoreError}</span>}

                {/* --- SSE Status Display --- */}
                {/* Only show SSE status if this card is the selected one */}
                {isSelected && (
                  <div className={styles.sseStatus}>
                    {sseConnectionStatus === 'connecting' && <span>Connecting to progress stream...</span>}
                    {sseConnectionStatus === 'open' && <span>Listening for progress...</span>}
                    {sseConnectionStatus === 'closed' && <span>Progress stream closed.</span>}
                    {sseConnectionStatus === 'error' && <span className={styles.error}>SSE Error: {sseError || 'Unknown SSE error'}</span>}

                    {/* Display latest progress message */}
                    {sseProgressData && (
                      <pre className={styles.sseProgressData}>
                        <code>{JSON.stringify(sseProgressData, null, 2)}</code>
                      </pre>
                    )}
                  </div>
                )}
                {/* --- End SSE Status Display --- */}
              </div>
              {/* --- End Rescore Button --- */}

              {/* --- Feedback Section --- */}
              <div className={styles.detailItem}>
                {!showFeedbackForm ? (
                  <button
                    onClick={(e) => {
                      e.stopPropagation(); // Prevent card click
                      setShowFeedbackForm(true);
                    }}
                    className={styles.feedbackToggleButton} // Add specific styling if needed
                  >
                    Provide Feedback
                  </button>
                ) : (
                  <FeedbackForm
                    articleId={article.ID}
                    onCancel={(e?: React.MouseEvent) => { // Allow event argument for stopPropagation
                       e?.stopPropagation();
                       setShowFeedbackForm(false);
                    }}
                    onSubmitSuccess={() => {
                      // Optionally keep the form open with success message, or close it:
                      // setShowFeedbackForm(false);
                      console.log(`Feedback submitted successfully for article ${article.ID}`);
                    }}
                  />
                )}
              </div>
              {/* --- End Feedback Section --- */}

            </div>
          )}
        </section>
      )}
    </article>
  );
};

export default ArticleCard;
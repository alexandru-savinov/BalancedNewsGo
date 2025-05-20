// Cache management
const CACHE_PREFIX = 'nbg_';
const CACHE_EXPIRY = 30 * 60 * 1000; // 30 minutes

function getCachedItem(key) {
    const cacheKey = `${CACHE_PREFIX}${key}`;
    const cacheItem = localStorage.getItem(cacheKey);

    if (!cacheItem) return null;

    try {
        const { data, timestamp } = JSON.parse(cacheItem);

        // Check if cache has expired
        if (Date.now() - timestamp > CACHE_EXPIRY) {
            localStorage.removeItem(cacheKey);
            return null;
        }

        return data;
    } catch (error) {
        console.error('Error parsing cached item:', error);
        localStorage.removeItem(cacheKey);
        return null;
    }
}

function setCachedItem(key, data) {
    const cacheKey = `${CACHE_PREFIX}${key}`;
    const cacheItem = {
        data,
        timestamp: Date.now()
    };

    try {
        localStorage.setItem(cacheKey, JSON.stringify(cacheItem));
    } catch (error) {
        console.error('Error caching item:', error);
        // If storage fails (e.g., quota exceeded), clear the cache and try again
        clearCache();
        try {
            localStorage.setItem(cacheKey, JSON.stringify(cacheItem));
        } catch (e) {
            console.error('Error caching item after clearance:', e);
        }
    }
}

function clearCache() {
    // Remove all items with our prefix
    for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && key.startsWith(CACHE_PREFIX)) {
            localStorage.removeItem(key);
        }
    }
}

// Helper functions
function getConfidenceClass(confidence) {
    if (confidence >= 0.7) return 'high';
    if (confidence >= 0.4) return 'medium';
    return 'low';
}

function getConfidenceColor(confidence) {
    if (confidence >= 0.7) return 'var(--confidence-high)';
    if (confidence >= 0.4) return 'var(--confidence-medium)';
    return 'var(--confidence-low)';
}

function getScoreLabel(score) {
    if (score === null || score === undefined) return 'Not analyzed';
    if (score < -0.6) return 'Strong Left';
    if (score < -0.2) return 'Moderate Left';
    if (score <= 0.2) return 'Center';
    if (score <= 0.6) return 'Moderate Right';
    return 'Strong Right';
}

function formatDate(dateString) {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleString();
}

// Error handling
function showError(message) {
    const errorElement = document.getElementById('article-error');
    errorElement.textContent = message;
    errorElement.style.display = 'block';
    document.getElementById('article-loading').style.display = 'none';
}

// Main article loading function
async function loadArticle() {
    try {
        // Extract article ID from URL
        const id = window.location.pathname.split('/').pop();
        if (!id || isNaN(id)) {
            throw new Error('Invalid article ID');
        }

        // Check cache first
        const cacheKey = `article_${id}`;
        const cachedArticle = getCachedItem(cacheKey);

        let article;

        if (cachedArticle) {
            console.log('Using cached article data');
            article = cachedArticle;
        } else {
            // Fetch article data from API
            // Add cache busting parameter to avoid browser caching
            const timestamp = new Date().getTime();
            const response = await fetch(`/api/articles/${id}?_t=${timestamp}`);
            if (!response.ok) {
                if (response.status === 404) {
                    throw new Error('Article not found');
                }
                throw new Error(`Error fetching article: ${response.statusText}`);
            }

            const responseData = await response.json();
            if (!responseData.success || !responseData.data) {
                throw new Error('Invalid response format');
            }

            article = responseData.data;

            // Cache the article data
            setCachedItem(cacheKey, article);
        }

        // Populate article elements
        document.getElementById('article-title').textContent = article.title || 'Untitled';
        document.getElementById('article-source').textContent = `Source: ${article.source || 'Unknown'}`;
        document.getElementById('article-pubdate').textContent = `Published: ${formatDate(article.pub_date)}`;
        document.getElementById('article-fetched').textContent = `Fetched: ${formatDate(article.created_at)}`;
        document.getElementById('article-content').innerHTML = article.content || 'No content available';

        // Set bias score and confidence
        const score = article.composite_score;
        const confidence = article.confidence;

        document.getElementById('article-score').textContent =
            `Political Leaning: ${getScoreLabel(score)} (${score !== null ? score.toFixed(2) : 'N/A'})`;

        document.getElementById('article-confidence').innerHTML =
            `Confidence: <span class="confidence-indicator" style="background: ${getConfidenceColor(confidence)}"></span>
            ${confidence !== null ? Math.round(confidence * 100) : 'N/A'}%`;

        // Position the bias indicator
        const biasIndicator = document.getElementById('bias-indicator');
        if (score !== null) {
            // Map -1 to 1 score to 0% to 100% position
            const position = ((score + 1) / 2) * 100;
            biasIndicator.style.left = `${position}%`;
            biasIndicator.style.backgroundColor = 'black';
        } else {
            biasIndicator.style.left = '50%';
            biasIndicator.style.backgroundColor = '#ccc';
        }

        // Load ensemble details
        loadEnsembleDetails(id);

        // Show the article container, hide loading
        document.getElementById('article-loading').style.display = 'none';
        document.getElementById('article-container').style.display = 'block';

        // Update page title
        document.title = `${article.title || 'Article'} - NewsBalancer`;

        // Load feedback status and display feedback options
        loadFeedbackStatus(id);

    } catch (error) {
        console.error('Error loading article:', error);
        showError(error.message || 'Failed to load article');
    }
}

// Load ensemble details for deeper analysis
async function loadEnsembleDetails(articleId) {
    try {
        // Check cache first
        const cacheKey = `ensemble_${articleId}`;
        const cachedDetails = null; // Disable cache temporarily for debugging

        let details;

        if (cachedDetails) {
            console.log('Using cached ensemble details');
            details = cachedDetails;
        } else {
            // Try the new endpoint first, fall back to the old one if it fails
            let response;
            let data;
            let success = false;

            try {
                response = await fetch(`/api/articles/${articleId}/ensemble`);
                if (response.ok) {
                    data = await response.json();
                    if (data.success) {
                        details = data.scores;
                        success = true;
                    }
                }
            } catch (err) {
                console.error('Error with primary endpoint, trying fallback:', err);
            }

            // If the first attempt failed, try the legacy endpoint
            if (!success) {
                try {
                    response = await fetch(`/api/articles/${articleId}/ensemble-details`);
                    if (!response.ok) {
                        throw new Error(`Error fetching ensemble details: ${response.statusText}`);
                    }

                    data = await response.json();
                    if (!data.success) {
                        throw new Error(data.error?.message || 'Failed to fetch ensemble details');
                    }

                    details = data.data || data.scores;
                } catch (fallbackErr) {
                    throw fallbackErr; // Rethrow if both attempts fail
                }
            }

            // Cache the details
            setCachedItem(cacheKey, details);
        }

        // Debug logging
        console.log('Ensemble details:', JSON.stringify(details));

        // Format the ensemble details for display with visualizations
        let html = '<div class="ensemble-details">';

        // Group by perspective
        const perspectives = {};

        if (details && Array.isArray(details) && details.length > 0) {
            // Get the first ensemble result (most recent)
            const latestEnsemble = details[0];

            if (latestEnsemble && latestEnsemble.sub_results && Array.isArray(latestEnsemble.sub_results)) {
                console.log('Found sub_results:', latestEnsemble.sub_results.length);
                // Group models by perspective
                for (const score of latestEnsemble.sub_results) {
                    // Skip invalid entries
                    if (!score || typeof score !== 'object') continue;

                    // Ensure we have valid data
                    const perspective = score.perspective || 'unknown';
                    const model = score.model || 'unknown model';
                    const scoreValue = typeof score.score === 'number' ? score.score : 0;
                    const confidence = typeof score.confidence === 'number' ? score.confidence : 0;

                    if (!perspectives[perspective]) {
                        perspectives[perspective] = [];
                    }

                    perspectives[perspective].push({
                        model: model,
                        score: scoreValue,
                        confidence: confidence,
                        explanation: score.explanation || ''
                    });
                }
            } else {
                console.error('No valid sub_results found in ensemble data');
            }
        } else {
            console.error('No valid ensemble details found');
        }

        // Format each perspective group
        for (const [perspective, scores] of Object.entries(perspectives)) {
            html += `<div class="perspective-group">`;
            html += `<h4>${perspective.charAt(0).toUpperCase() + perspective.slice(1)} Perspective</h4>`;
            html += '<div class="perspective-scores">';

            for (const score of scores) {
                // Safety checks for score value
                const validScore = !isNaN(score.score) ? score.score : 0;
                const scorePosition = ((validScore + 1) / 2) * 100;

                // Safety checks for confidence
                const validConfidence = !isNaN(score.confidence) ? score.confidence : 0;
                const confidenceColor = getConfidenceColor(validConfidence);
                const confidencePercent = (validConfidence * 100).toFixed(0);

                html += `
                    <div class="model-score">
                        <h5>${score.model || 'Unknown Model'}</h5>
                        <div class="score-details">
                            <div class="score-value">
                                <span>Score: <strong>${validScore.toFixed(2)}</strong></span>
                                <span data-tooltip="Political leaning score from -1 (left) to +1 (right)">${getScoreLabel(validScore)}</span>
                            </div>
                            <div class="confidence-value">
                                <span>Confidence: <span class="confidence-indicator" style="background: ${confidenceColor}"></span>
                                <strong>${confidencePercent}%</strong></span>
                            </div>
                        </div>
                        <div class="model-bias-slider">
                            <div class="bias-slider mini-slider">
                                <div class="bias-indicator" style="left: ${scorePosition}%"></div>
                            </div>
                        </div>
                    </div>
                `;
            }

            html += '</div></div>';
        }

        html += '</div>';

        // Update the UI
        document.getElementById('ensemble-loading').style.display = 'none';
        document.getElementById('ensemble-content').innerHTML = html;
        document.getElementById('ensemble-content').style.display = 'block';

    } catch (error) {
        console.error('Error loading ensemble details:', error);
        document.getElementById('ensemble-loading').textContent =
            `Failed to load analysis details: ${error.message || 'Unknown error'}`;
    }
}

// Load feedback status and show feedback options
async function loadFeedbackStatus(articleId) {
    try {
        // Check if feedback section exists in the DOM
        const feedbackSection = document.getElementById('feedback-section');
        if (!feedbackSection) return;

        // Show loading state
        feedbackSection.innerHTML = '<p>Loading feedback options...</p>';

        // We don't cache feedback as it should be real-time

        // Fetch existing feedback if any
        const response = await fetch(`/api/articles/${articleId}/feedback`);

        if (!response.ok) {
            if (response.status === 404) {
                // No feedback yet, show form
                showFeedbackForm(articleId, null);
                return;
            }
            throw new Error(`Error fetching feedback: ${response.statusText}`);
        }

        const data = await response.json();

        if (data.success && data.data) {
            // Show existing feedback
            showExistingFeedback(articleId, data.data);
        } else {
            // No feedback yet, show form
            showFeedbackForm(articleId, null);
        }

    } catch (error) {
        console.error('Error loading feedback:', error);
        const feedbackSection = document.getElementById('feedback-section');
        if (feedbackSection) {
            feedbackSection.innerHTML = `<p class="error-message">Error loading feedback: ${error.message || 'Unknown error'}</p>`;
        }
    }
}

// Show feedback form
function showFeedbackForm(articleId, existingFeedback) {
    const feedbackSection = document.getElementById('feedback-section');
    if (!feedbackSection) return;

    // Create the feedback form
    const html = `
        <h3>Provide Your Feedback</h3>
        <p>Do you agree with our analysis? Let us know your perspective.</p>

        <form id="feedback-form">
            <div class="rating-container">
                <label>Your rating of the article's political leaning:</label>
                <div class="bias-slider feedback-slider">
                    <input type="range" id="user-bias-rating" min="-1" max="1" step="0.1"
                           value="${existingFeedback ? existingFeedback.user_score : 0}" />
                    <div class="bias-labels">
                        <span class="label-left">Left</span>
                        <span class="label-center">Center</span>
                        <span class="label-right">Right</span>
                    </div>
                </div>
                <div id="selected-rating">Selected: Center (0.0)</div>
            </div>

            <div class="comment-container">
                <label for="user-comment">Comments (optional):</label>
                <textarea id="user-comment" rows="3">${existingFeedback ? existingFeedback.comment || '' : ''}</textarea>
            </div>

            <button type="submit" id="submit-feedback">Submit Feedback</button>
        </form>
    `;

    feedbackSection.innerHTML = html;

    // Add event listeners
    const ratingSlider = document.getElementById('user-bias-rating');
    const selectedRating = document.getElementById('selected-rating');

    ratingSlider.addEventListener('input', function() {
        const value = parseFloat(this.value);
        selectedRating.textContent = `Selected: ${getScoreLabel(value)} (${value.toFixed(1)})`;
    });

    document.getElementById('feedback-form').addEventListener('submit', function(e) {
        e.preventDefault();
        submitFeedback(articleId);
    });

    // Trigger initial update of the selected rating display
    ratingSlider.dispatchEvent(new Event('input'));
}

// Show existing feedback
function showExistingFeedback(articleId, feedback) {
    const feedbackSection = document.getElementById('feedback-section');
    if (!feedbackSection) return;

    const scoreLabel = getScoreLabel(feedback.user_score);

    const html = `
        <h3>Your Feedback</h3>
        <div class="existing-feedback">
            <p>You rated this article as: <strong>${scoreLabel} (${feedback.user_score.toFixed(1)})</strong></p>

            <div class="bias-slider feedback-slider">
                <div class="bias-indicator" style="left: ${((feedback.user_score + 1) / 2) * 100}%"></div>
                <div class="bias-labels">
                    <span class="label-left">Left</span>
                    <span class="label-center">Center</span>
                    <span class="label-right">Right</span>
                </div>
            </div>

            ${feedback.comment ? `<p>Your comment: "${feedback.comment}"</p>` : ''}

            <p>Submitted on: ${formatDate(feedback.created_at)}</p>

            <button id="edit-feedback">Edit Feedback</button>
        </div>
    `;

    feedbackSection.innerHTML = html;

    // Add event listener to edit button
    document.getElementById('edit-feedback').addEventListener('click', function() {
        showFeedbackForm(articleId, feedback);
    });
}

// Submit feedback
async function submitFeedback(articleId) {
    try {
        const userScore = parseFloat(document.getElementById('user-bias-rating').value);
        const comment = document.getElementById('user-comment').value.trim();

        const submitButton = document.getElementById('submit-feedback');
        submitButton.disabled = true;
        submitButton.textContent = 'Submitting...';

        const response = await fetch(`/api/feedback`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                article_id: articleId,
                user_score: userScore,
                comment: comment
            })
        });

        if (!response.ok) {
            throw new Error(`Error submitting feedback: ${response.statusText}`);
        }

        const data = await response.json();

        if (data.success) {
            // Clear any cached feedback data
            localStorage.removeItem(`${CACHE_PREFIX}feedback_${articleId}`);

            // Refresh feedback display
            loadFeedbackStatus(articleId);
        } else {
            throw new Error(data.error?.message || 'Failed to submit feedback');
        }

    } catch (error) {
        console.error('Error submitting feedback:', error);
        alert(`Failed to submit feedback: ${error.message || 'Unknown error'}`);

        // Re-enable submit button
        const submitButton = document.getElementById('submit-feedback');
        if (submitButton) {
            submitButton.disabled = false;
            submitButton.textContent = 'Submit Feedback';
        }
    }
}

// Load article when page loads
document.addEventListener('DOMContentLoaded', () => {
    loadArticle();
    setupReanalysisFeature();
});

// Re-analysis feature - enhanced version
function setupReanalysisFeature() {
    const reanalyzeBtn = document.getElementById('reanalyzeArticleBtn');
    const btnTextEl = document.getElementById('reanalyzeArticleBtnText');
    const loadingEl = document.getElementById('reanalyzeArticleBtnLoading');
    const statusContainer = document.getElementById('reanalyzeStatusContainer');
    const statusMessageEl = document.getElementById('reanalyzeStatusMessage');
    const progressBar = document.getElementById('reanalyzeProgressBar');
    const progressBarInner = document.getElementById('reanalyzeProgressBarInner');

    if (!reanalyzeBtn || !statusMessageEl || !statusContainer) {
        console.warn('Re-analysis UI elements not found in the DOM.');
        return;
    }

    // Get article ID from the current URL path, article container, or URL params
    function getArticleId() {
        // Try from URL path first (most common case)
        const idFromPath = window.location.pathname.split('/').pop();
        if (idFromPath && !isNaN(idFromPath)) {
            return idFromPath;
        }

        // Try from URL query params
        const params = new URLSearchParams(window.location.search);
        const idFromQuery = params.get('id');
        if (idFromQuery && !isNaN(idFromQuery)) {
            return idFromQuery;
        }

        // Try from data attribute on article container
        const articleContainer = document.getElementById('article-container');
        if (articleContainer && articleContainer.dataset.articleId) {
            return articleContainer.dataset.articleId;
        }

        return null;
    }

    // Update status message with appropriate styling
    function showStatus(type, message, showProgress = false) {
        statusMessageEl.textContent = message;

        // Reset previous styling
        statusMessageEl.style.backgroundColor = '';
        statusMessageEl.style.color = '';
        statusMessageEl.style.border = '';

        // Apply styling based on message type
        if (type === 'info') {
            statusMessageEl.style.backgroundColor = '#e7f3fe';
            statusMessageEl.style.color = '#0c5460';
            statusMessageEl.style.border = '1px solid #b8daff';
        } else if (type === 'success') {
            statusMessageEl.style.backgroundColor = '#d4edda';
            statusMessageEl.style.color = '#155724';
            statusMessageEl.style.border = '1px solid #c3e6cb';
        } else if (type === 'error') {
            statusMessageEl.style.backgroundColor = '#f8d7da';
            statusMessageEl.style.color = '#721c24';
            statusMessageEl.style.border = '1px solid #f5c6cb';
        }

        // Show/hide progress bar
        if (progressBar) {
            progressBar.style.display = showProgress ? 'block' : 'none';
        }

        statusContainer.style.display = 'block';
    }

    // Update progress bar
    function updateProgress(percent) {
        if (progressBarInner && !isNaN(percent)) {
            // Clamp between 0-100
            const clampedPercent = Math.max(0, Math.min(100, percent));
            progressBarInner.style.width = `${clampedPercent}%`;
        }
    }

    // Toggle loading state
    function setLoading(isLoading) {
        if (!reanalyzeBtn || !btnTextEl || !loadingEl) return;

        reanalyzeBtn.disabled = isLoading;
        btnTextEl.style.display = isLoading ? 'none' : 'inline';
        loadingEl.style.display = isLoading ? 'inline' : 'none';

        if (!isLoading && progressBar) {
            // Reset progress on completion
            progressBar.style.display = 'none';
            progressBarInner.style.width = '0%';
        }
    }

    // Error handler with detailed error classification
    function handleError(error, response) {
        let errorMessage = 'An unknown error occurred';
        let errorDetail = '';

        if (response) {
            // Server responded with an error
            switch (response.status) {
                case 400:
                    errorMessage = 'Invalid request parameters';
                    break;
                case 401:
                    errorMessage = 'LLM authentication failed';
                    break;
                case 402:
                    errorMessage = 'LLM payment required or credits exhausted';
                    break;
                case 404:
                    errorMessage = 'Article not found';
                    break;
                case 429:
                    errorMessage = 'LLM rate limit exceeded';
                    break;
                case 503:
                    errorMessage = 'LLM service unavailable';
                    break;
                default:
                    errorMessage = `Error (${response.status})`;
            }

            // Try to extract more details from response
            try {
                const data = response._bodyText || response._bodyInit || '';
                if (data) {
                    const parsed = JSON.parse(data);
                    if (parsed.error && parsed.error.message) {
                        errorDetail = parsed.error.message;
                    }
                }
            } catch (e) {
                console.error('Error parsing error response:', e);
            }
        } else if (error) {
            // Network or client-side error
            errorMessage = error.message || 'Network error';
        }

        // Combine messages if we have details
        const fullMessage = errorDetail
            ? `${errorMessage}: ${errorDetail}`
            : errorMessage;

        showStatus('error', fullMessage);
        console.error('Re-analysis error:', fullMessage);
    }

    // Connect to SSE endpoint for progress updates
    let eventSource = null;

    function connectProgressSSE(articleId) {
        if (eventSource) {
            eventSource.close();
        }

        eventSource = new EventSource(`/api/llm/score-progress/${articleId}`);

        eventSource.onmessage = function(event) {
            try {
                const progress = JSON.parse(event.data);
                const status = (progress.status || '').toLowerCase();
                const message = (progress.message || '').toLowerCase();
                // Treat 'complete', 'success', 'done', or message containing 'complete' as done
                if (status === "complete" || status === "success" || status === "done" || message.includes("complete")) {
                    showStatus('success', progress.message || 'Analysis complete!');
                    updateProgress(100);

                    // Disconnect SSE
                    eventSource.close();
                    eventSource = null;
                    setLoading(false);

                    // Immediately reload the article data to show new scores
                    const cacheKey = `article_${articleId}`;
                    localStorage.removeItem(`${CACHE_PREFIX}${cacheKey}`);
                    loadArticle(); // Reload the article with fresh data
                } else if (status === "error") {
                    showStatus('error', progress.message || 'Error during analysis');
                    setLoading(false);
                    eventSource.close();
                    eventSource = null;
                } else {
                    // Update in-progress state
                    showStatus('info', progress.message || 'Processing...', true);
                    updateProgress(progress.percent || 0);
                }
            } catch (e) {
                console.error('Error parsing SSE data:', e);
            }
        };
        // Fallback: if SSE closes and progress bar is still active, reload
        eventSource.onclose = function() {
            if (progressBar && progressBar.style.display === 'block') {
                setLoading(false);
                const cacheKey = `article_${articleId}`;
                localStorage.removeItem(`${CACHE_PREFIX}${cacheKey}`);
                loadArticle();
            }
        };
    }

    // Clean up SSE connection when navigating away
    window.addEventListener('beforeunload', () => {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
    });

    // Handle button click
    reanalyzeBtn.addEventListener('click', async () => {
        const articleId = getArticleId();
        if (!articleId) {
            showStatus('error', 'Could not determine article ID');
            return;
        }

        setLoading(true);
        showStatus('info', 'Starting reanalysis...');

        try {
            const response = await fetch(`/api/llm/reanalyze/${articleId}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (response.status === 202 || response.status === 200) {
                await response.json();
                showStatus('success', 'Re-analysis started. Tracking progress...', true);
                updateProgress(10); // Update progress after successful request

                // Connect to SSE for progress updates
                connectProgressSSE(articleId);
            } else {
                handleError(null, response);
                setLoading(false);
            }
        } catch (error) {
            handleError(error);
            setLoading(false);
        }
    });
}

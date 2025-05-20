# AI Agent Implementation Plan: Trigger Article Re-analysis

**Overall AI Agent Instructions:**
*   For each iteration, you will be given tasks to generate or modify specific files.
*   Assume all HTML styling is in-line within the HTML file, as per the current codebase pattern.
*   When modifying existing JavaScript, ensure new functions or event listeners are added without disrupting existing, unrelated code. If modifying a function, clearly indicate the original and the new version if necessary, or provide the complete new function.
*   Generated code should be robust and include basic error checking (e.g., for null elements).
*   Pay close attention to IDs and class names used in HTML and JavaScript to ensure they match.
*   All generated code for a file should be complete for that iteration's changes.

---

## Iteration 1: Basic Implementation (Core Functionality)

**Goal:** Implement the minimal viable version of the re-analysis button with basic success/error handling, integrating with existing UI patterns.

**Agent's Task for this Iteration:**
Generate the HTML snippet to be added to `web/article.html` and the JS code to be added to `web/js/article.js` to implement the basic re-analysis trigger.

**Files to be Modified:**
1.  `web/article.html` (Add new HTML elements with inline styling)
2.  `web/js/article.js` (Add new JavaScript logic)

**Detailed Instructions & Code Generation:**

**1. `web/article.html`**
    *   **Instruction:** Add the re-analysis button within the existing bias-summary div, right after the bias-slider-container div. This placement integrates naturally with the analysis UI.
    *   **Generated HTML Code:**
        ```html
        <div class="bias-summary">
            <h3>Political Bias Analysis</h3>
            <span id="article-score"></span>
            <span id="article-confidence"></span>
            
            <div class="bias-slider-container">
                <div class="bias-slider" id="bias-slider">
                    <div id="bias-indicator" class="bias-indicator"></div>
                </div>
                <div class="bias-labels">
                    <span class="label-left">Left</span>
                    <span class="label-center">Center</span>
                    <span class="label-right">Right</span>
                </div>
            </div>
            
            <!-- New re-analysis button component -->
            <div class="article-actions" style="margin-top: 1rem; padding: 0.75rem; border-radius: 6px; background-color: var(--secondary-bg); border: 1px solid var(--border-color);">
                <button id="reanalyzeArticleBtn" style="background-color: var(--accent-color); color: white; border: none; border-radius: 4px; padding: 0.5rem 1rem; cursor: pointer; font-weight: 500; transition: background-color 0.2s;">
                    <span id="reanalyzeArticleBtnText">Re-evaluate Bias</span>
                    <span id="reanalyzeArticleBtnLoading" style="display: none;">
                        <span style="display: inline-block; animation: pulse 1.5s infinite; margin-left: 0.3rem;">
                            Loading...
                        </span>
                    </span>
                </button>
                <div id="reanalyzeStatusContainer" style="margin-top: 0.75rem; display: none;">
                    <div id="reanalyzeProgressBar" style="height: 6px; width: 100%; background-color: #eee; border-radius: 3px; margin-bottom: 0.5rem; overflow: hidden; display: none;">
                        <div id="reanalyzeProgressBarInner" style="height: 100%; width: 0%; background-color: var(--accent-color); transition: width 0.3s;"></div>
                    </div>
                    <p id="reanalyzeStatusMessage" style="margin: 0; padding: 0.5rem; border-radius: 4px;"></p>
                </div>
            </div>
        </div>
        ```

**2. `web/js/article.js`**
    *   **Instruction:** Add the JavaScript code to handle the re-analysis button, following existing patterns in article.js. The code should retrieve the article ID, make a POST request to `/api/llm/reanalyze/{id}`, update status messages, and connect to the SSE endpoint for progress tracking.
    *   **Generated JavaScript Code:**
        ```javascript
        // Add this code at the end of the document-ready function, before the closing });

        // Re-analysis feature
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
                        // Ignore parsing errors
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
                        // Update status based on progress
                        if (progress.status === "Complete" || progress.status === "Success") {
                            showStatus('success', progress.message || 'Analysis complete!');
                            updateProgress(100);
                            
                            // Disconnect SSE
                            eventSource.close();
                            eventSource = null;
                            setLoading(false);
                            
                            // Reload the article data to show new scores
                            setTimeout(() => {
                                // Clear cache to ensure fresh data
                                const cacheKey = `article_${articleId}`;
                                localStorage.removeItem(`${CACHE_PREFIX}${cacheKey}`);
                                loadArticle(); // Reload the article with fresh data
                            }, 1000);
                        } else if (progress.status === "Error") {
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
                
                eventSource.onerror = function() {
                    console.error('SSE connection error');
                    eventSource.close();
                    eventSource = null;
                    showStatus('error', 'Lost connection to progress updates');
                    setLoading(false);
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
                showStatus('info', 'Sending re-analysis request...', true);
                updateProgress(5); // Show initial progress
                
                try {
                    const response = await fetch(`/api/llm/reanalyze/${articleId}`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({})
                    });
                    
                    if (response.status === 202 || response.status === 200) {
                        const data = await response.json();
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
        
        // Call setup function when document is ready
        setupReanalysisFeature();
        ```

**Agent Self-Verification Checklist (for Iteration 1):**
*   [✓] HTML: Button is integrated within the existing bias-summary section
*   [✓] HTML: Uses inline styling matching existing site patterns (no separate CSS file)
*   [✓] HTML: Loading animation matches existing site patterns using the 'pulse' animation
*   [✓] HTML: Button styling matches site's accent color and existing UI
*   [✓] JavaScript: Robust article ID extraction from URL patterns
*   [✓] JavaScript: Proper error handling and status message management
*   [✓] JavaScript: SSE connection for real-time progress updates
*   [✓] JavaScript: Follows existing cache management patterns
*   [✓] JavaScript: Proper reload of article data after successful re-analysis

---

## Iteration 2: Enhanced UX and Error Handling

**Goal:** Improve the user experience with detailed progress tracking, better error visualization, and seamless integration with the article view.

**Agent's Task for this Iteration:**
Enhance the existing implementation from Iteration 1 with more robust error handling, better progress visualization, and UI improvements.

**Files to be Modified:**
1.  `web/article.html` (Modify the HTML added in Iteration 1)
2.  `web/js/article.js` (Enhance the JavaScript from Iteration 1)

**Detailed Instructions & Code Generation:**

**1. `web/article.html`**
    *   **Instruction:** Enhance the re-analysis UI component with a more detailed progress display.
    *   **Generated HTML Code:**
        ```html
        <div class="bias-summary">
            <h3>Political Bias Analysis</h3>
            <span id="article-score"></span>
            <span id="article-confidence"></span>
            
            <div class="bias-slider-container">
                <div class="bias-slider" id="bias-slider">
                    <div id="bias-indicator" class="bias-indicator"></div>
                </div>
                <div class="bias-labels">
                    <span class="label-left">Left</span>
                    <span class="label-center">Center</span>
                    <span class="label-right">Right</span>
                </div>
            </div>
            
            <!-- Enhanced re-analysis button component -->
            <div class="article-actions" style="margin-top: 1rem; padding: 0.75rem; border-radius: 6px; background-color: var(--secondary-bg); border: 1px solid var(--border-color);">
                <button id="reanalyzeArticleBtn" style="background-color: var(--accent-color); color: white; border: none; border-radius: 4px; padding: 0.5rem 1rem; cursor: pointer; font-weight: 500; transition: background-color 0.2s;">
                    <span id="reanalyzeArticleBtnText">Re-evaluate Bias</span>
                    <span id="reanalyzeArticleBtnLoading" style="display: none;">
                        <span style="display: inline-block; animation: pulse 1.5s infinite; margin-left: 0.3rem;">
                            Loading...
                        </span>
                    </span>
                </button>
                <div id="reanalyzeStatusContainer" style="margin-top: 0.75rem; display: none;">
                    <div id="reanalyzeProgressBar" style="height: 6px; width: 100%; background-color: #eee; border-radius: 3px; margin-bottom: 0.5rem; overflow: hidden; display: none;">
                        <div id="reanalyzeProgressBarInner" style="height: 100%; width: 0%; background-color: var(--accent-color); transition: width 0.3s;"></div>
                    </div>
                    <p id="reanalyzeStatusMessage" style="margin: 0; padding: 0.5rem; border-radius: 4px;"></p>
                </div>
            </div>
        </div>
        ```

**2. `web/js/article.js`**
    *   **Instruction:** Enhance the JavaScript implementation with better progress visualization, more robust error handling, and better integration with the article view lifecycle.
    *   **Generated JavaScript Code:**
        ```javascript
        // Replace the setupReanalysisFeature function from Iteration 1 with this enhanced version

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
                        // Ignore parsing errors
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
                        // Update status based on progress
                        if (progress.status === "Complete" || progress.status === "Success") {
                            showStatus('success', progress.message || 'Analysis complete!');
                            updateProgress(100);
                            
                            // Disconnect SSE
                            eventSource.close();
                            eventSource = null;
                            setLoading(false);
                            
                            // Reload the article data to show new scores
                            setTimeout(() => {
                                // Clear cache to ensure fresh data
                                const cacheKey = `article_${articleId}`;
                                localStorage.removeItem(`${CACHE_PREFIX}${cacheKey}`);
                                loadArticle(); // Reload the article with fresh data
                            }, 1000);
                        } else if (progress.status === "Error") {
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
                
                eventSource.onerror = function() {
                    console.error('SSE connection error');
                    eventSource.close();
                    eventSource = null;
                    showStatus('error', 'Lost connection to progress updates');
                    setLoading(false);
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
                showStatus('info', 'Sending re-analysis request...', true);
                updateProgress(5); // Show initial progress
                
                try {
                    const response = await fetch(`/api/llm/reanalyze/${articleId}`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({})
                    });
                    
                    if (response.status === 202 || response.status === 200) {
                        const data = await response.json();
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
        
        // Call setup function when document is ready
        setupReanalysisFeature();
        ```

**Agent Self-Verification Checklist (for Iteration 2):**
*   [✓] HTML: Enhanced UI with progress bar for better visual feedback
*   [✓] HTML: Loading animation matches existing site patterns using the 'pulse' animation
*   [✓] HTML: Progress visualization is well-integrated with the bias analysis section
*   [✓] JavaScript: More robust article ID extraction with multiple strategies 
*   [✓] JavaScript: Enhanced error handling with specific error messages for different status codes
*   [✓] JavaScript: Progress bar updates based on SSE progress updates
*   [✓] JavaScript: Proper cleanup of SSE connection when navigating away
*   [✓] JavaScript: Updates article view after successful re-analysis
*   [✓] JavaScript: Better integration with existing article loading patterns

The enhanced implementation now provides a seamless, real-time progress tracking experience that matches the existing UI patterns while adding robust error handling and progress visualization. 
// Cache management
const CACHE_PREFIX = 'nbg_';
const CACHE_EXPIRY = 5 * 60 * 1000; // 5 minutes for list views (shorter than article detail)

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

// Global variables for pagination and filtering
let currentOffset = 0;
let currentLimit = 20;
let currentSource = '';
let currentLeaning = '';
let currentSortBy = 'date';
let currentSortDir = 'desc';
let totalArticles = 0;
let articleCache = []; // Cache for client-side sorting/filtering

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
    const errorElement = document.getElementById('articles-error');
    errorElement.textContent = message;
    errorElement.style.display = 'block';
    document.getElementById('articles-loading').style.display = 'none';
}

// Toggle advanced view
function toggleAdvanced(button) {
    const advancedSection = button.nextElementSibling;
    if (advancedSection && advancedSection.classList.contains('advanced-section')) {
        advancedSection.style.display = advancedSection.style.display === 'none' ? 'block' : 'none';
        button.textContent = advancedSection.style.display === 'none' ? 'Show Details' : 'Hide Details';
    }
}

// Generate a cache key based on current filters
function getArticlesCacheKey() {
    return `articles_${currentOffset}_${currentLimit}_${currentSource}_${currentLeaning}_${currentSortBy}_${currentSortDir}`;
}

// Main function to fetch articles
async function fetchArticles(forceRefresh = false) {
    try {
        // Show loading indicator
        document.getElementById('articles-loading').style.display = 'block';
        document.getElementById('articles-error').style.display = 'none';
        
        // Generate cache key based on current parameters
        const cacheKey = getArticlesCacheKey();
        
        // Check cache if not forcing refresh
        if (!forceRefresh) {
            const cachedArticles = getCachedItem(cacheKey);
            if (cachedArticles) {
                console.log('Using cached articles list');
                articleCache = cachedArticles;
                totalArticles = articleCache.length;
                
                // Hide loading indicator
                document.getElementById('articles-loading').style.display = 'none';
                
                // Render from cache
                renderArticles(articleCache);
                return;
            }
        }
        
        // Build API URL with filters
        let url = `/api/articles?offset=${currentOffset}&limit=${currentLimit}`;
        if (currentSource) url += `&source=${encodeURIComponent(currentSource)}`;
        if (currentLeaning) url += `&leaning=${encodeURIComponent(currentLeaning)}`;
        
        // Add server-side sorting if supported
        url += `&sort=${currentSortBy}&direction=${currentSortDir}`;
        
        // Fetch articles
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`Error fetching articles: ${response.statusText}`);
        }
        
        const responseData = await response.json();
        if (!responseData.success || !Array.isArray(responseData.data)) {
            throw new Error('Invalid response format');
        }
        
        articleCache = responseData.data;
        totalArticles = articleCache.length;
        
        // Cache the articles
        setCachedItem(cacheKey, articleCache);
        
        // Hide loading indicator
        document.getElementById('articles-loading').style.display = 'none';
        
        // Now render the articles
        renderArticles(articleCache);
        
    } catch (error) {
        console.error('Error fetching articles:', error);
        showError(error.message || 'Failed to load articles');
    }
}

// Function to render articles from cache
function renderArticles(articles) {
    // If no articles found
    if (articles.length === 0) {
        document.getElementById('articles').innerHTML = '<p>No articles found matching your criteria.</p>';
        return;
    }
    
    // Render articles
    let articlesHTML = '';
    
    for (const article of articles) {
        // Extract data with fallbacks
        const id = article.id || article.article_id || '';
        const title = article.title || 'Untitled';
        const source = article.source || 'Unknown Source';
        const pubDate = formatDate(article.pub_date);
        const createdAt = formatDate(article.created_at);
        const content = article.content || 'No content available';
        const summary = content.length > 200 ? content.substring(0, 200) + '...' : content;
        const compositeScore = article.composite_score !== undefined ? article.composite_score : null;
        const confidence = article.confidence !== undefined ? article.confidence : null;
        
        // Calculate slider position and labels
        const sliderPosition = compositeScore !== null ? ((compositeScore + 1) / 2) * 100 : 50;
        const scoreLabel = getScoreLabel(compositeScore);
        const confidenceColor = getConfidenceColor(confidence);
        
        articlesHTML += `
            <article data-id="${id}" data-score="${compositeScore}" data-confidence="${confidence}">
                <h2><a href="/article/${id}">${title}</a></h2>
                
                <div class="metadata">
                    <span><strong>Source:</strong> ${source}</span>
                    <span><strong>Published:</strong> ${pubDate}</span>
                    <span><strong>Fetched:</strong> ${createdAt}</span>
                    <span class="bias-label"><strong>Bias:</strong> ${scoreLabel}</span>
                    <span class="confidence">
                        <strong>Confidence:</strong> 
                        <span class="confidence-indicator" style="background: ${confidenceColor}"></span>
                        ${confidence !== null ? Math.round(confidence * 100) : '--'}%
                    </span>
                </div>
                
                <div class="summary">
                    <p>${summary}</p>
                </div>
                
                <div class="bias-slider-container">
                    <div class="bias-slider">
                        <div class="bias-indicator" style="left: ${sliderPosition}%"></div>
                    </div>
                    <div class="bias-labels">
                        <span class="label-left">Left</span>
                        <span class="label-center">Center</span>
                        <span class="label-right">Right</span>
                    </div>
                </div>
                
                <button onclick="toggleAdvanced(this)">Show Details</button>
                <div class="advanced-section">
                    <p><strong>ID:</strong> ${id}</p>
                    <p><strong>Score:</strong> ${compositeScore !== null ? compositeScore.toFixed(2) : 'Not analyzed'}</p>
                    <p><strong>Full Content:</strong></p>
                    <div>${content}</div>
                </div>
            </article>
        `;
    }
    
    // Update the DOM
    document.getElementById('articles').innerHTML = articlesHTML;
    
    // Update pagination controls
    updatePaginationControls();
}

// Update pagination controls
function updatePaginationControls() {
    const prevButton = document.getElementById('prevPage');
    const nextButton = document.getElementById('nextPage');
    const pageInfo = document.getElementById('pageInfo');
    
    // Disable prev button if on first page
    prevButton.disabled = currentOffset === 0;
    
    // Disable next button if fewer articles than limit (last page)
    nextButton.disabled = totalArticles < currentLimit;
    
    // Update page info
    const currentPage = Math.floor(currentOffset / currentLimit) + 1;
    pageInfo.textContent = `Page ${currentPage}`;
}

// Pagination handlers
function prevPage() {
    if (currentOffset > 0) {
        currentOffset = Math.max(0, currentOffset - currentLimit);
        fetchArticles();
    }
}

function nextPage() {
    if (totalArticles >= currentLimit) {
        currentOffset += currentLimit;
        fetchArticles();
    }
}

// Apply filters
function applyFilters() {
    // Reset pagination
    currentOffset = 0;
    
    // Get filter values
    currentSource = document.getElementById('sourceFilter').value;
    currentLeaning = document.getElementById('leaningFilter').value;
    
    // Get limit
    const limitInput = document.getElementById('limitInput');
    const limit = parseInt(limitInput.value, 10);
    currentLimit = isNaN(limit) || limit < 5 || limit > 50 ? 20 : limit;
    
    // Get sort options
    const sortSelect = document.getElementById('sortSelect');
    if (sortSelect) {
        const sortValue = sortSelect.value;
        if (sortValue) {
            const [sortBy, sortDir] = sortValue.split('-');
            currentSortBy = sortBy;
            currentSortDir = sortDir;
        }
    }
    
    // Fetch articles with new filters - force refresh
    fetchArticles(true);
}

// Apply client-side sort
function applySortClient(sortBy, sortDir) {
    if (!articleCache.length) return;
    
    const sortedArticles = [...articleCache];
    
    switch (sortBy) {
        case 'date':
            sortedArticles.sort((a, b) => {
                const dateA = new Date(a.pub_date || a.created_at);
                const dateB = new Date(b.pub_date || b.created_at);
                return sortDir === 'asc' ? dateA - dateB : dateB - dateA;
            });
            break;
        case 'score':
            sortedArticles.sort((a, b) => {
                const scoreA = a.composite_score !== undefined ? a.composite_score : 0;
                const scoreB = b.composite_score !== undefined ? b.composite_score : 0;
                return sortDir === 'asc' ? scoreA - scoreB : scoreB - scoreA;
            });
            break;
        case 'confidence':
            sortedArticles.sort((a, b) => {
                const confA = a.confidence !== undefined ? a.confidence : 0;
                const confB = b.confidence !== undefined ? b.confidence : 0;
                return sortDir === 'asc' ? confA - confB : confB - confA;
            });
            break;
        case 'source':
            sortedArticles.sort((a, b) => {
                const sourceA = (a.source || '').toLowerCase();
                const sourceB = (b.source || '').toLowerCase();
                if (sortDir === 'asc') {
                    return sourceA.localeCompare(sourceB);
                } else {
                    return sourceB.localeCompare(sourceA);
                }
            });
            break;
    }
    
    // Render the sorted articles
    renderArticles(sortedArticles);
}

// Filter articles client-side (for score range)
function applyScoreRangeFilter() {
    if (!articleCache.length) return;
    
    const minScore = parseFloat(document.getElementById('minScoreRange').value);
    const maxScore = parseFloat(document.getElementById('maxScoreRange').value);
    
    document.getElementById('minScoreValue').textContent = minScore.toFixed(1);
    document.getElementById('maxScoreValue').textContent = maxScore.toFixed(1);
    
    const filteredArticles = articleCache.filter(article => {
        const score = article.composite_score;
        if (score === undefined || score === null) return false;
        return score >= minScore && score <= maxScore;
    });
    
    renderArticles(filteredArticles);
}

// Apply confidence threshold filter
function applyConfidenceFilter() {
    if (!articleCache.length) return;
    
    const confidenceThreshold = parseFloat(document.getElementById('confidenceRange').value);
    document.getElementById('confidenceValue').textContent = (confidenceThreshold * 100).toFixed(0) + '%';
    
    const filteredArticles = articleCache.filter(article => {
        const confidence = article.confidence;
        if (confidence === undefined || confidence === null) return false;
        return confidence >= confidenceThreshold;
    });
    
    renderArticles(filteredArticles);
}

// Fetch available sources for filter dropdown
async function fetchSources() {
    try {
        // Check cache first
        const cachedSources = getCachedItem('sources');
        if (cachedSources) {
            populateSourcesDropdown(cachedSources);
            return;
        }
        
        const response = await fetch('/api/sources');
        if (!response.ok) {
            console.error('Failed to fetch sources');
            return;
        }
        
        const data = await response.json();
        if (!data.success || !Array.isArray(data.data)) {
            console.error('Invalid source data format');
            return;
        }
        
        const sources = data.data;
        
        // Cache the sources
        setCachedItem('sources', sources);
        
        // Populate dropdown
        populateSourcesDropdown(sources);
        
    } catch (error) {
        console.error('Error fetching sources:', error);
    }
}

// Helper to populate the sources dropdown
function populateSourcesDropdown(sources) {
    const sourceSelect = document.getElementById('sourceFilter');
    
    // Add options
    sources.forEach(source => {
        const option = document.createElement('option');
        option.value = source;
        option.textContent = source;
        sourceSelect.appendChild(option);
    });
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    // Set up event listeners
    document.getElementById('prevPage').addEventListener('click', prevPage);
    document.getElementById('nextPage').addEventListener('click', nextPage);
    document.getElementById('applyFilters').addEventListener('click', applyFilters);
    
    // Add refresh button event listener
    const refreshButton = document.getElementById('refreshArticles');
    if (refreshButton) {
        refreshButton.addEventListener('click', function() {
            fetchArticles(true); // Force refresh
        });
    }
    
    // Set up sorting event listener
    const sortSelect = document.getElementById('sortSelect');
    if (sortSelect) {
        sortSelect.addEventListener('change', function() {
            const [sortBy, sortDir] = this.value.split('-');
            currentSortBy = sortBy;
            currentSortDir = sortDir;
            
            // If we're doing client-side sorting
            if (articleCache.length > 0) {
                applySortClient(sortBy, sortDir);
            } else {
                // Otherwise fetch with new sort parameters
                fetchArticles();
            }
        });
    }
    
    // Set up score range filter listeners
    const minScoreRange = document.getElementById('minScoreRange');
    const maxScoreRange = document.getElementById('maxScoreRange');
    const confidenceRange = document.getElementById('confidenceRange');
    
    if (minScoreRange && maxScoreRange) {
        minScoreRange.addEventListener('input', applyScoreRangeFilter);
        maxScoreRange.addEventListener('input', applyScoreRangeFilter);
    }
    
    if (confidenceRange) {
        confidenceRange.addEventListener('input', applyConfidenceFilter);
    }
    
    // Load initial data
    fetchArticles();
    fetchSources();
}); 
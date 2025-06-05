/**
 * Articles API Service
 *
 * Handles all article-related API operations including:
 * - Fetching articles list with pagination and filtering
 * - Getting individual article details
 * - Managing article scoring and bias analysis
 * - Handling article re-analysis requests
 * - Managing article feedback
 */

class ArticlesService {
    constructor(apiClient) {
        this.client = apiClient || new ApiClient();
        this.baseUrl = '/articles';
    }

    /**
     * Get paginated list of articles
     * @param {Object} options - Query options
     * @returns {Promise} Articles response
     */
    async getArticles(options = {}) {
        const params = {
            limit: options.limit || 20,
            offset: options.offset || 0,
            source: options.source,
            bias_min: options.biasMin,
            bias_max: options.biasMax,
            sort: options.sort || 'pub_date',
            order: options.order || 'desc',
            search: options.search
        };

        // Remove undefined parameters
        Object.keys(params).forEach(key => {
            if (params[key] === undefined) {
                delete params[key];
            }
        });

        try {
            return await this.client.getJson(this.baseUrl, { params });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getArticles',
                params
            });
        }
    }

    /**
     * Get article by ID
     * @param {number} id - Article ID
     * @returns {Promise} Article data
     */
    async getArticle(id) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        try {
            return await this.client.getJson(`${this.baseUrl}/${id}`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getArticle',
                articleId: id
            });
        }
    }

    /**
     * Get article bias analysis
     * @param {number} id - Article ID
     * @returns {Promise} Bias analysis data
     */
    async getArticleBias(id) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        try {
            return await this.client.getJson(`${this.baseUrl}/${id}/bias`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getArticleBias',
                articleId: id
            });
        }
    }

    /**
     * Get article summary
     * @param {number} id - Article ID
     * @returns {Promise} Summary data
     */
    async getArticleSummary(id) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        try {
            return await this.client.getJson(`${this.baseUrl}/${id}/summary`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getArticleSummary',
                articleId: id
            });
        }
    }

    /**
     * Create new article
     * @param {Object} articleData - Article data
     * @returns {Promise} Created article data
     */
    async createArticle(articleData) {
        if (!articleData || !articleData.url) {
            throw new Error('Article data with URL is required');
        }

        try {
            return await this.client.postJson(this.baseUrl, articleData);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'createArticle',
                articleData
            });
        }
    }

    /**
     * Trigger article re-analysis
     * @param {number} id - Article ID
     * @param {Object} options - Re-analysis options
     * @returns {Promise} Re-analysis response
     */
    async reanalyzeArticle(id, options = {}) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        const requestData = {
            force: options.force || false,
            models: options.models || null
        };

        try {
            return await this.client.postJson(`/llm/reanalyze/${id}`, requestData);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'reanalyzeArticle',
                articleId: id,
                options
            });
        }
    }

    /**
     * Get re-analysis progress via Server-Sent Events
     * @param {number} id - Article ID
     * @param {Function} onProgress - Progress callback
     * @param {Function} onError - Error callback
     * @param {Function} onComplete - Completion callback
     * @returns {EventSource} EventSource instance
     */
    createProgressStream(id, onProgress, onError, onComplete) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        const eventSource = this.client.createEventSource(`/progress/${id}`);

        eventSource.addEventListener('message', (event) => {
            try {
                const data = JSON.parse(event.data);

                if (data.error) {
                    const error = this.client.errorHandler.handle(new Error(data.error), {
                        operation: 'progressStream',
                        articleId: id
                    });
                    onError && onError(error);
                } else if (data.complete) {
                    onComplete && onComplete(data);
                    eventSource.close();
                } else {
                    onProgress && onProgress(data);
                }
            } catch (parseError) {
                const error = this.client.errorHandler.handle(parseError, {
                    operation: 'progressStreamParse',
                    articleId: id,
                    eventData: event.data
                });
                onError && onError(error);
            }
        });

        eventSource.addEventListener('error', (event) => {
            const error = this.client.errorHandler.handle(new Error('Progress stream error'), {
                operation: 'progressStreamConnection',
                articleId: id,
                event
            });
            onError && onError(error);
        });

        return eventSource;
    }

    /**
     * Submit manual bias score
     * @param {number} id - Article ID
     * @param {number} score - Bias score (-1 to 1)
     * @param {Object} metadata - Additional metadata
     * @returns {Promise} Score submission response
     */
    async submitManualScore(id, score, metadata = {}) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        if (typeof score !== 'number' || score < -1 || score > 1) {
            throw new Error('Score must be a number between -1 and 1');
        }

        const requestData = {
            score,
            metadata: {
                timestamp: new Date().toISOString(),
                userAgent: navigator.userAgent,
                ...metadata
            }
        };

        try {
            return await this.client.postJson(`/manual-score/${id}`, requestData);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'submitManualScore',
                articleId: id,
                score,
                metadata
            });
        }
    }

    /**
     * Get article feedback
     * @param {number} id - Article ID
     * @returns {Promise} Feedback data
     */
    async getArticleFeedback(id) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        try {
            return await this.client.getJson('/feedback', {
                params: { article_id: id }
            });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getArticleFeedback',
                articleId: id
            });
        }
    }

    /**
     * Submit article feedback
     * @param {Object} feedbackData - Feedback data
     * @returns {Promise} Feedback submission response
     */
    async submitFeedback(feedbackData) {
        if (!feedbackData || !feedbackData.article_id) {
            throw new Error('Feedback data with article_id is required');
        }

        const requestData = {
            ...feedbackData,
            timestamp: new Date().toISOString()
        };

        try {
            return await this.client.postJson('/feedback', requestData);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'submitFeedback',
                feedbackData
            });
        }
    }

    /**
     * Get available news sources
     * @returns {Promise} List of news sources
     */
    async getSources() {
        try {
            return await this.client.getJson('/sources');
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getSources'
            });
        }
    }

    /**
     * Get ensemble scoring details
     * @param {number} id - Article ID
     * @returns {Promise} Ensemble details
     */
    async getEnsembleDetails(id) {
        if (!id || typeof id !== 'number') {
            throw new Error('Article ID is required and must be a number');
        }

        try {
            return await this.client.getJson(`${this.baseUrl}/${id}/ensemble`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getEnsembleDetails',
                articleId: id
            });
        }
    }

    /**
     * Get article statistics
     * @param {Object} options - Query options
     * @returns {Promise} Statistics data
     */
    async getStatistics(options = {}) {
        const params = {
            days: options.days || 7,
            source: options.source
        };

        try {
            return await this.client.getJson('/stats', { params });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getStatistics',
                options
            });
        }
    }

    /**
     * Search articles
     * @param {string} query - Search query
     * @param {Object} options - Search options
     * @returns {Promise} Search results
     */
    async searchArticles(query, options = {}) {
        if (!query || typeof query !== 'string') {
            throw new Error('Search query is required and must be a string');
        }

        const params = {
            q: query,
            limit: options.limit || 20,
            offset: options.offset || 0,
            source: options.source,
            bias_min: options.biasMin,
            bias_max: options.biasMax
        };

        try {
            return await this.client.getJson('/search', { params });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'searchArticles',
                query,
                options
            });
        }
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ArticlesService;
} else {
    window.ArticlesService = ArticlesService;
}

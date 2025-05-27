/**
 * API Client with Circuit Breaker and Error Handling
 *
 * Main HTTP client for the NewsBalancer frontend that provides:
 * - Circuit breaker protection for service resilience
 * - Centralized error handling and classification
 * - Retry logic with exponential backoff
 * - Rate limiting awareness
 * - Request/response interceptors
 * - Authentication handling
 * - Progress tracking for long-running operations
 */

class ApiClient {
    constructor(options = {}) {
        // Configuration
        this.baseURL = options.baseURL || '/api';
        this.timeout = options.timeout || 30000; // 30 seconds
        this.defaultHeaders = {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            ...options.headers
        };

        // Initialize dependencies
        this.errorHandler = new ErrorHandler(options.errorHandler);
        this.circuitBreakerFactory = new CircuitBreakerFactory();

        // Retry configuration
        this.retryConfig = {
            maxAttempts: 3,
            baseDelay: 1000,        // 1s base delay
            maxDelay: 10000,        // 10s max delay
            backoffFactor: 2,       // Exponential backoff
            retryableStatusCodes: [408, 429, 500, 502, 503, 504],
            ...options.retryConfig
        };

        // Circuit breaker configuration for different services
        this.circuitBreakerConfigs = {
            articles: {
                failureThreshold: 5,
                recoveryTimeout: 30000,
                monitoringPeriod: 10000
            },
            llm: {
                failureThreshold: 3,     // More sensitive for LLM
                recoveryTimeout: 60000,  // Longer recovery for LLM
                monitoringPeriod: 15000
            },
            admin: {
                failureThreshold: 5,
                recoveryTimeout: 30000,
                monitoringPeriod: 10000
            },
            ...options.circuitBreakerConfigs
        };

        // Request interceptors
        this.requestInterceptors = [];
        this.responseInterceptors = [];

        // Progress tracking
        this.progressCallbacks = new Map();

        // Rate limiting tracking
        this.rateLimitInfo = {
            limit: null,
            remaining: null,
            reset: null,
            retryAfter: null
        };

        this.setupDefaultInterceptors();
    }

    /**
     * Setup default request/response interceptors
     */
    setupDefaultInterceptors() {
        // Request interceptor for authentication and headers
        this.addRequestInterceptor((config) => {
            // Add timestamp for request tracking
            config.headers['X-Request-Timestamp'] = Date.now().toString();

            // Add request ID for debugging
            config.headers['X-Request-ID'] = this.generateRequestId();

            return config;
        });

        // Response interceptor for rate limiting and error handling
        this.addResponseInterceptor(
            (response) => {
                this.updateRateLimitInfo(response);
                return response;
            },
            (error) => {
                return Promise.reject(this.errorHandler.handle(error));
            }
        );
    }

    /**
     * Add request interceptor
     * @param {Function} onFulfilled - Success handler
     * @param {Function} onRejected - Error handler
     */
    addRequestInterceptor(onFulfilled, onRejected) {
        this.requestInterceptors.push({ onFulfilled, onRejected });
    }

    /**
     * Add response interceptor
     * @param {Function} onFulfilled - Success handler
     * @param {Function} onRejected - Error handler
     */
    addResponseInterceptor(onFulfilled, onRejected) {
        this.responseInterceptors.push({ onFulfilled, onRejected });
    }

    /**
     * Make HTTP request with circuit breaker and retry logic
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async request(config) {
        // Apply request interceptors
        let requestConfig = await this.applyRequestInterceptors(config);

        // Get appropriate circuit breaker
        const service = this.getServiceFromUrl(requestConfig.url);
        const circuitBreaker = this.getCircuitBreaker(service);

        // Execute request with circuit breaker protection
        return circuitBreaker.execute(async () => {
            return this.executeRequestWithRetry(requestConfig);
        });
    }

    /**
     * Execute request with retry logic
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async executeRequestWithRetry(config) {
        let lastError;

        for (let attempt = 1; attempt <= this.retryConfig.maxAttempts; attempt++) {
            try {
                const response = await this.executeRequest(config);
                return await this.applyResponseInterceptors(response, null);
            } catch (error) {
                lastError = error;

                // Check if error is retryable
                if (!this.isRetryableError(error) || attempt === this.retryConfig.maxAttempts) {
                    throw await this.applyResponseInterceptors(null, error);
                }

                // Calculate delay for next attempt
                const delay = this.calculateRetryDelay(attempt);
                console.warn(`Request failed (attempt ${attempt}/${this.retryConfig.maxAttempts}), retrying in ${delay}ms:`, error);

                await this.sleep(delay);
            }
        }

        throw lastError;
    }

    /**
     * Execute the actual HTTP request
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async executeRequest(config) {
        const url = this.buildUrl(config.url);
        const options = this.buildFetchOptions(config);

        // Create abort controller for timeout
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.timeout);

        try {
            const response = await fetch(url, {
                ...options,
                signal: controller.signal
            });

            clearTimeout(timeoutId);

            // Check if response is ok
            if (!response.ok) {
                throw response;
            }

            return response;
        } catch (error) {
            clearTimeout(timeoutId);
            throw error;
        }
    }

    /**
     * Apply request interceptors
     * @param {Object} config - Request configuration
     * @returns {Object} Modified configuration
     */
    async applyRequestInterceptors(config) {
        let modifiedConfig = { ...config };

        for (const interceptor of this.requestInterceptors) {
            try {
                if (interceptor.onFulfilled) {
                    modifiedConfig = await interceptor.onFulfilled(modifiedConfig);
                }
            } catch (error) {
                if (interceptor.onRejected) {
                    modifiedConfig = await interceptor.onRejected(error);
                } else {
                    throw error;
                }
            }
        }

        return modifiedConfig;
    }

    /**
     * Apply response interceptors
     * @param {Response} response - HTTP response
     * @param {Error} error - Error if any
     * @returns {*} Modified response or error
     */
    async applyResponseInterceptors(response, error) {
        let result = response;
        let currentError = error;

        for (const interceptor of this.responseInterceptors) {
            try {
                if (currentError && interceptor.onRejected) {
                    result = await interceptor.onRejected(currentError);
                    currentError = null; // Error was handled
                } else if (result && interceptor.onFulfilled) {
                    result = await interceptor.onFulfilled(result);
                }
            } catch (interceptorError) {
                currentError = interceptorError;
                result = null;
            }
        }

        if (currentError) {
            throw currentError;
        }

        return result;
    }

    /**
     * Get circuit breaker for service
     * @param {string} service - Service name
     * @returns {CircuitBreaker} Circuit breaker instance
     */
    getCircuitBreaker(service) {
        const config = this.circuitBreakerConfigs[service] || this.circuitBreakerConfigs.articles;
        return this.circuitBreakerFactory.getCircuitBreaker(service, config);
    }

    /**
     * Determine service from URL
     * @param {string} url - Request URL
     * @returns {string} Service name
     */
    getServiceFromUrl(url) {
        if (url.includes('/llm/') || url.includes('/reanalyze')) {
            return 'llm';
        } else if (url.includes('/admin/') || url.includes('/feeds/') || url.includes('/refresh')) {
            return 'admin';
        }
        return 'articles';
    }

    /**
     * Check if error is retryable
     * @param {*} error - Error to check
     * @returns {boolean} Whether error is retryable
     */
    isRetryableError(error) {
        // Don't retry circuit breaker errors
        if (error && error.name === 'CircuitBreakerError') {
            return false;
        }

        // Retry network errors
        if (error instanceof TypeError && error.message.includes('fetch')) {
            return true;
        }

        // Retry timeout errors
        if (error && error.name === 'AbortError') {
            return true;
        }

        // Retry specific HTTP status codes
        if (error instanceof Response) {
            return this.retryConfig.retryableStatusCodes.includes(error.status);
        }

        return false;
    }

    /**
     * Calculate retry delay with exponential backoff
     * @param {number} attempt - Current attempt number
     * @returns {number} Delay in milliseconds
     */
    calculateRetryDelay(attempt) {
        const delay = this.retryConfig.baseDelay * Math.pow(this.retryConfig.backoffFactor, attempt - 1);
        return Math.min(delay, this.retryConfig.maxDelay);
    }

    /**
     * Build full URL from relative path
     * @param {string} path - API path
     * @returns {string} Full URL
     */
    buildUrl(path) {
        if (path.startsWith('http')) {
            return path;
        }
        return `${this.baseURL}${path.startsWith('/') ? '' : '/'}${path}`;
    }

    /**
     * Build fetch options from config
     * @param {Object} config - Request configuration
     * @returns {Object} Fetch options
     */
    buildFetchOptions(config) {
        const options = {
            method: config.method || 'GET',
            headers: {
                ...this.defaultHeaders,
                ...config.headers
            }
        };

        // Add body for non-GET requests
        if (config.data && options.method !== 'GET') {
            if (typeof config.data === 'string') {
                options.body = config.data;
            } else {
                options.body = JSON.stringify(config.data);
            }
        }

        // Add query parameters for GET requests
        if (config.params && options.method === 'GET') {
            const url = new URL(this.buildUrl(config.url), window.location.origin);
            Object.entries(config.params).forEach(([key, value]) => {
                if (value !== null && value !== undefined) {
                    url.searchParams.append(key, value);
                }
            });
            config.url = url.pathname + url.search;
        }

        return options;
    }

    /**
     * Update rate limiting information from response headers
     * @param {Response} response - HTTP response
     */
    updateRateLimitInfo(response) {
        const headers = response.headers;

        this.rateLimitInfo = {
            limit: parseInt(headers.get('X-RateLimit-Limit')) || null,
            remaining: parseInt(headers.get('X-RateLimit-Remaining')) || null,
            reset: parseInt(headers.get('X-RateLimit-Reset')) || null,
            retryAfter: parseInt(headers.get('Retry-After')) || null
        };
    }

    /**
     * Generate unique request ID
     * @returns {string} Request ID
     */
    generateRequestId() {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Sleep for specified milliseconds
     * @param {number} ms - Milliseconds to sleep
     * @returns {Promise} Sleep promise
     */
    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    // Convenience methods for HTTP verbs

    /**
     * GET request
     * @param {string} url - Request URL
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async get(url, config = {}) {
        return this.request({ ...config, method: 'GET', url });
    }

    /**
     * POST request
     * @param {string} url - Request URL
     * @param {*} data - Request data
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async post(url, data, config = {}) {
        return this.request({ ...config, method: 'POST', url, data });
    }

    /**
     * PUT request
     * @param {string} url - Request URL
     * @param {*} data - Request data
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async put(url, data, config = {}) {
        return this.request({ ...config, method: 'PUT', url, data });
    }

    /**
     * DELETE request
     * @param {string} url - Request URL
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async delete(url, config = {}) {
        return this.request({ ...config, method: 'DELETE', url });
    }

    /**
     * PATCH request
     * @param {string} url - Request URL
     * @param {*} data - Request data
     * @param {Object} config - Request configuration
     * @returns {Promise} Response promise
     */
    async patch(url, data, config = {}) {
        return this.request({ ...config, method: 'PATCH', url, data });
    }

    // Utility methods for JSON responses

    /**
     * GET request that returns JSON
     * @param {string} url - Request URL
     * @param {Object} config - Request configuration
     * @returns {Promise} JSON response promise
     */
    async getJson(url, config = {}) {
        const response = await this.get(url, config);
        return response.json();
    }

    /**
     * POST request that returns JSON
     * @param {string} url - Request URL
     * @param {*} data - Request data
     * @param {Object} config - Request configuration
     * @returns {Promise} JSON response promise
     */
    async postJson(url, data, config = {}) {
        const response = await this.post(url, data, config);
        return response.json();
    }

    // Server-Sent Events support

    /**
     * Create SSE connection with error handling
     * @param {string} url - SSE endpoint URL
     * @param {Object} options - SSE options
     * @returns {EventSource} EventSource instance
     */
    createEventSource(url, options = {}) {
        const fullUrl = this.buildUrl(url);
        const eventSource = new EventSource(fullUrl);

        // Add error handling
        eventSource.addEventListener('error', (event) => {
            const error = new Error('SSE connection error');
            error.event = event;
            this.errorHandler.handle(error, { url: fullUrl, type: 'sse' });
        });

        return eventSource;
    }

    // Monitoring and debugging methods

    /**
     * Get rate limiting information
     * @returns {Object} Rate limit info
     */
    getRateLimitInfo() {
        return { ...this.rateLimitInfo };
    }

    /**
     * Get circuit breaker status for all services
     * @returns {Object} Circuit breaker status
     */
    getCircuitBreakerStatus() {
        return this.circuitBreakerFactory.getAllStatus();
    }

    /**
     * Reset all circuit breakers
     */
    resetCircuitBreakers() {
        this.circuitBreakerFactory.resetAll();
    }

    /**
     * Get health status of API client
     * @returns {Object} Health status
     */
    getHealthStatus() {
        const circuitStatus = this.getCircuitBreakerStatus();
        const overallHealth = Object.values(circuitStatus).every(status =>
            status.state === 'CLOSED' && status.healthPercentage > 70
        );

        return {
            healthy: overallHealth,
            rateLimiting: this.rateLimitInfo,
            circuitBreakers: circuitStatus,
            timestamp: new Date().toISOString()
        };
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ApiClient;
} else {
    window.ApiClient = ApiClient;
}

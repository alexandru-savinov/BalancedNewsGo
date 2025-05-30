/**
 * Centralized Error Handler
 *
 * Handles all types of errors that can occur in the application:
 * - HTTP errors (4xx, 5xx)
 * - Network errors (connection, timeout)
 * - Circuit breaker errors
 * - LLM service errors (rate limiting, authentication, credits)
 * - Validation errors
 * - Application errors
 */

class ErrorHandler {
    constructor(options = {}) {
        this.showUserFriendlyMessages = options.showUserFriendlyMessages !== false;
        this.enableConsoleLogging = options.enableConsoleLogging !== false;
        this.enableErrorReporting = options.enableErrorReporting || false;
        this.reportingEndpoint = options.reportingEndpoint || '/api/errors';

        // Error classification patterns
        this.errorPatterns = {
            network: /network|connection|timeout|fetch/i,
            llm: /llm|openrouter|rate.?limit|credit|auth/i,
            validation: /validation|invalid|required|format/i,
            permission: /permission|unauthorized|forbidden/i,
            notFound: /not.?found|missing/i
        };

        // User-friendly error messages
        this.userMessages = {
            network: 'Connection problem. Please check your internet connection and try again.',
            timeout: 'Request timed out. Please try again.',
            rateLimit: 'Too many requests. Please wait a moment and try again.',
            authentication: 'Authentication failed. Please refresh the page and try again.',
            credits: 'Service temporarily unavailable due to account limits.',
            permission: 'You don\'t have permission to perform this action.',
            notFound: 'The requested resource was not found.',
            validation: 'Please check your input and try again.',
            server: 'Server error. Our team has been notified.',
            circuit: 'Service is temporarily unavailable. Please try again later.',
            unknown: 'An unexpected error occurred. Please try again.'
        };
    }

    /**
     * Handle any error and return standardized error object
     * @param {Error|Response|Object} error - The error to handle
     * @param {Object} context - Additional context about the error
     * @returns {Object} Standardized error object
     */
    handle(error, context = {}) {
        const errorInfo = this.classifyError(error, context);

        // Log error for debugging
        if (this.enableConsoleLogging) {
            this.logError(errorInfo, context);
        }

        // Report error if enabled
        if (this.enableErrorReporting && errorInfo.reportable) {
            this.reportError(errorInfo, context).catch(err => {
                console.warn('Failed to report error:', err);
            });
        }

        return errorInfo;
    }

    /**
     * Classify error type and extract relevant information
     * @param {*} error - The error to classify
     * @param {Object} context - Additional context
     * @returns {Object} Classified error information
     */
    classifyError(error, context = {}) {
        const baseError = {
            timestamp: new Date().toISOString(),
            userMessage: this.userMessages.unknown,
            technicalMessage: 'Unknown error',
            code: 'UNKNOWN_ERROR',
            type: 'unknown',
            retryable: false,
            retryAfter: null,
            statusCode: null,
            details: {},
            reportable: true,
            context
        };

        try {
            // Handle Response objects (fetch API)
            if (error instanceof Response) {
                return this.handleHttpResponse(error, baseError);
            }

            // Handle Error objects
            if (error instanceof Error) {
                return this.handleErrorObject(error, baseError);
            }

            // Handle API error objects
            if (error && typeof error === 'object' && error.error) {
                return this.handleApiError(error, baseError);
            }

            // Handle string errors
            if (typeof error === 'string') {
                return {
                    ...baseError,
                    technicalMessage: error,
                    userMessage: this.getUserMessage(error),
                    type: this.detectErrorType(error)
                };
            }

            return baseError;

        } catch (classificationError) {
            console.error('Error during error classification:', classificationError);
            return {
                ...baseError,
                technicalMessage: 'Error classification failed',
                details: { originalError: error, classificationError: classificationError.message }
            };
        }
    }

    /**
     * Handle HTTP Response errors
     * @param {Response} response - HTTP response object
     * @param {Object} baseError - Base error object
     * @returns {Object} Error information
     */
    async handleHttpResponse(response, baseError) {
        const statusCode = response.status;
        const statusText = response.statusText;

        // Try to extract error details from response body
        let responseBody = null;
        try {
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                responseBody = await response.json();
            } else {
                responseBody = await response.text();
            }
        } catch (parseError) {
            console.warn('Failed to parse error response body:', parseError);
        }

        const errorInfo = {
            ...baseError,
            statusCode,
            technicalMessage: `HTTP ${statusCode}: ${statusText}`,
            details: { responseBody, headers: this.extractHeaders(response) }
        };

        // Classify based on status code
        switch (Math.floor(statusCode / 100)) {
            case 4: // 4xx Client errors
                return this.handleClientError(statusCode, responseBody, errorInfo);
            case 5: // 5xx Server errors
                return this.handleServerError(statusCode, responseBody, errorInfo);
            default:
                return errorInfo;
        }
    }

    /**
     * Handle 4xx client errors
     * @param {number} statusCode - HTTP status code
     * @param {*} responseBody - Response body
     * @param {Object} errorInfo - Base error info
     * @returns {Object} Error information
     */
    handleClientError(statusCode, responseBody, errorInfo) {
        const updates = {
            type: 'client',
            reportable: statusCode >= 500 // Only report server errors by default
        };

        switch (statusCode) {
            case 400:
                updates.code = 'BAD_REQUEST';
                updates.userMessage = this.userMessages.validation;
                updates.type = 'validation';
                break;
            case 401:
                updates.code = 'UNAUTHORIZED';
                updates.userMessage = this.userMessages.authentication;
                updates.type = 'authentication';
                break;
            case 403:
                updates.code = 'FORBIDDEN';
                updates.userMessage = this.userMessages.permission;
                updates.type = 'permission';
                break;
            case 404:
                updates.code = 'NOT_FOUND';
                updates.userMessage = this.userMessages.notFound;
                updates.type = 'notFound';
                break;
            case 408:
                updates.code = 'REQUEST_TIMEOUT';
                updates.userMessage = this.userMessages.timeout;
                updates.type = 'timeout';
                updates.retryable = true;
                break;
            case 429:
                updates.code = 'RATE_LIMITED';
                updates.userMessage = this.userMessages.rateLimit;
                updates.type = 'rateLimit';
                updates.retryable = true;
                updates.retryAfter = this.extractRetryAfter(responseBody);
                break;
            default:
                updates.code = `CLIENT_ERROR_${statusCode}`;
                updates.userMessage = 'Request error. Please check your input and try again.';
        }

        // Extract specific error message from response
        if (responseBody && responseBody.error && responseBody.error.message) {
            updates.technicalMessage = responseBody.error.message;
            if (responseBody.error.code) {
                updates.code = responseBody.error.code;
            }
        }

        return { ...errorInfo, ...updates };
    }

    /**
     * Handle 5xx server errors
     * @param {number} statusCode - HTTP status code
     * @param {*} responseBody - Response body
     * @param {Object} errorInfo - Base error info
     * @returns {Object} Error information
     */
    handleServerError(statusCode, responseBody, errorInfo) {
        const updates = {
            type: 'server',
            retryable: true,
            reportable: true,
            userMessage: this.userMessages.server
        };

        switch (statusCode) {
            case 500:
                updates.code = 'INTERNAL_SERVER_ERROR';
                break;
            case 502:
                updates.code = 'BAD_GATEWAY';
                updates.userMessage = 'Service temporarily unavailable. Please try again.';
                break;
            case 503:
                updates.code = 'SERVICE_UNAVAILABLE';
                updates.userMessage = 'Service temporarily unavailable. Please try again.';
                updates.retryAfter = this.extractRetryAfter(responseBody);
                break;
            case 504:
                updates.code = 'GATEWAY_TIMEOUT';
                updates.userMessage = this.userMessages.timeout;
                break;
            default:
                updates.code = `SERVER_ERROR_${statusCode}`;
        }

        return { ...errorInfo, ...updates };
    }

    /**
     * Handle Error objects
     * @param {Error} error - Error object
     * @param {Object} baseError - Base error object
     * @returns {Object} Error information
     */
    handleErrorObject(error, baseError) {
        const errorInfo = {
            ...baseError,
            technicalMessage: error.message,
            details: {
                name: error.name,
                stack: error.stack
            }
        };

        // Handle specific error types
        switch (error.name) {
            case 'TypeError':
                if (error.message.includes('fetch')) {
                    return {
                        ...errorInfo,
                        code: 'NETWORK_ERROR',
                        type: 'network',
                        userMessage: this.userMessages.network,
                        retryable: true
                    };
                }
                break;

            case 'AbortError':
                return {
                    ...errorInfo,
                    code: 'REQUEST_ABORTED',
                    type: 'timeout',
                    userMessage: this.userMessages.timeout,
                    retryable: true
                };

            case 'CircuitBreakerError':
                return {
                    ...errorInfo,
                    code: error.code || 'CIRCUIT_BREAKER_ERROR',
                    type: 'circuit',
                    userMessage: this.userMessages.circuit,
                    retryable: true,
                    retryAfter: error.retryAfter
                };

            default:
                // Detect error type from message
                const detectedType = this.detectErrorType(error.message);
                return {
                    ...errorInfo,
                    type: detectedType,
                    userMessage: this.getUserMessage(error.message, detectedType)
                };
        }

        return errorInfo;
    }

    /**
     * Handle API error response objects
     * @param {Object} apiError - API error object
     * @param {Object} baseError - Base error object
     * @returns {Object} Error information
     */
    handleApiError(apiError, baseError) {
        const error = apiError.error || {};

        return {
            ...baseError,
            code: error.code || 'API_ERROR',
            technicalMessage: error.message || 'API error occurred',
            userMessage: this.getUserMessage(error.message, error.code),
            type: this.detectErrorType(error.message || error.code),
            retryable: error.retryable || false,
            retryAfter: error.retryAfter || null,
            details: {
                requestId: apiError.requestId,
                timestamp: apiError.timestamp,
                ...error
            }
        };
    }

    /**
     * Detect error type from message
     * @param {string} message - Error message
     * @returns {string} Error type
     */
    detectErrorType(message) {
        if (!message) return 'unknown';

        const lowerMessage = message.toLowerCase();

        for (const [type, pattern] of Object.entries(this.errorPatterns)) {
            if (pattern.test(lowerMessage)) {
                return type;
            }
        }

        return 'unknown';
    }

    /**
     * Get user-friendly message for error
     * @param {string} message - Technical error message
     * @param {string} type - Error type
     * @returns {string} User-friendly message
     */
    getUserMessage(message, type) {
        if (!this.showUserFriendlyMessages) {
            return message;
        }

        const detectedType = type || this.detectErrorType(message);
        return this.userMessages[detectedType] || this.userMessages.unknown;
    }

    /**
     * Extract retry-after value from response
     * @param {*} responseBody - Response body
     * @returns {number|null} Retry after seconds
     */
    extractRetryAfter(responseBody) {
        if (responseBody && responseBody.error && responseBody.error.retryAfter) {
            return responseBody.error.retryAfter;
        }
        return null;
    }

    /**
     * Extract relevant headers from response
     * @param {Response} response - HTTP response
     * @returns {Object} Relevant headers
     */
    extractHeaders(response) {
        const relevantHeaders = [
            'x-ratelimit-limit',
            'x-ratelimit-remaining',
            'x-ratelimit-reset',
            'retry-after',
            'x-request-id'
        ];

        const headers = {};
        relevantHeaders.forEach(header => {
            const value = response.headers.get(header);
            if (value) {
                headers[header] = value;
            }
        });

        return headers;
    }

    /**
     * Log error for debugging
     * @param {Object} errorInfo - Error information
     * @param {Object} context - Additional context
     */
    logError(errorInfo, context) {
        const logLevel = errorInfo.type === 'server' || errorInfo.statusCode >= 500 ? 'error' : 'warn';

        console[logLevel]('Error handled:', {
            type: errorInfo.type,
            code: errorInfo.code,
            message: errorInfo.technicalMessage,
            statusCode: errorInfo.statusCode,
            retryable: errorInfo.retryable,
            context,
            timestamp: errorInfo.timestamp
        });
    }

    /**
     * Report error to monitoring service
     * @param {Object} errorInfo - Error information
     * @param {Object} context - Additional context
     * @returns {Promise} Reporting promise
     */
    async reportError(errorInfo, context) {
        if (!this.enableErrorReporting) return;

        try {
            const reportData = {
                ...errorInfo,
                context: {
                    ...context,
                    url: window.location.href,
                    userAgent: navigator.userAgent,
                    timestamp: Date.now()
                }
            };

            // Don't report the error if it fails to avoid infinite loops
            await fetch(this.reportingEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(reportData)
            });

        } catch (reportingError) {
            // Silently fail - don't want error reporting to break the app
            console.debug('Error reporting failed:', reportingError);
        }
    }

    /**
     * Create error display for UI
     * @param {Object} errorInfo - Error information
     * @param {Object} options - Display options
     * @returns {Object} Display information
     */
    createErrorDisplay(errorInfo, options = {}) {
        return {
            title: this.getErrorTitle(errorInfo.type),
            message: options.showTechnical ? errorInfo.technicalMessage : errorInfo.userMessage,
            type: errorInfo.type,
            retryable: errorInfo.retryable,
            retryAfter: errorInfo.retryAfter,
            actions: this.getErrorActions(errorInfo),
            details: options.showDetails ? errorInfo.details : null
        };
    }

    /**
     * Get error title based on type
     * @param {string} type - Error type
     * @returns {string} Error title
     */
    getErrorTitle(type) {
        const titles = {
            network: 'Connection Error',
            timeout: 'Request Timeout',
            rateLimit: 'Rate Limited',
            authentication: 'Authentication Required',
            permission: 'Access Denied',
            notFound: 'Not Found',
            validation: 'Invalid Input',
            server: 'Server Error',
            circuit: 'Service Unavailable',
            unknown: 'Error'
        };

        return titles[type] || 'Error';
    }

    /**
     * Get available actions for error type
     * @param {Object} errorInfo - Error information
     * @returns {Array} Available actions
     */
    getErrorActions(errorInfo) {
        const actions = [];

        if (errorInfo.retryable) {
            actions.push({
                label: 'Try Again',
                action: 'retry',
                primary: true
            });
        }

        if (errorInfo.type === 'authentication') {
            actions.push({
                label: 'Refresh Page',
                action: 'refresh',
                primary: true
            });
        }

        actions.push({
            label: 'Go Back',
            action: 'back',
            primary: false
        });

        return actions;
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ErrorHandler;
} else {
    window.ErrorHandler = ErrorHandler;
}

/**
 * Circuit Breaker Implementation
 *
 * Implements the circuit breaker pattern to prevent cascade failures
 * and provide fallback behavior when services are unavailable.
 *
 * States:
 * - CLOSED: Normal operation, requests flow through
 * - OPEN: Service is down, requests fail fast
 * - HALF_OPEN: Testing if service is back, limited requests allowed
 */

class CircuitBreaker {
    constructor(options = {}) {
        // Configuration with defaults
        this.failureThreshold = options.failureThreshold || 5;
        this.recoveryTimeout = options.recoveryTimeout || 30000; // 30 seconds
        this.monitoringPeriod = options.monitoringPeriod || 10000; // 10 seconds
        this.name = options.name || 'CircuitBreaker';

        // State management
        this.state = 'CLOSED';
        this.failureCount = 0;
        this.lastFailureTime = null;
        this.nextAttemptTime = null;
        this.halfOpenAttempts = 0;
        this.maxHalfOpenAttempts = 3;

        // Metrics for monitoring
        this.metrics = {
            totalRequests: 0,
            successfulRequests: 0,
            failedRequests: 0,
            timeouts: 0,
            circuitOpenTime: 0,
            lastResetTime: Date.now()
        };

        // Event listeners for monitoring
        this.listeners = {
            stateChange: [],
            failure: [],
            success: [],
            open: [],
            halfOpen: [],
            close: []
        };
    }

    /**
     * Execute a function with circuit breaker protection
     * @param {Function} fn - Async function to execute
     * @param {*} args - Arguments to pass to the function
     * @returns {Promise} - Result of function execution
     */
    async execute(fn, ...args) {
        this.metrics.totalRequests++;

        // Check if circuit is open
        if (this.state === 'OPEN') {
            if (this.shouldAttemptReset()) {
                this.moveToHalfOpen();
            } else {
                const error = new CircuitBreakerError(
                    'Circuit breaker is OPEN',
                    'CIRCUIT_OPEN',
                    this.nextAttemptTime
                );
                this.emit('failure', error);
                throw error;
            }
        }

        // Execute the function
        try {
            const result = await fn(...args);
            this.onSuccess();
            return result;
        } catch (error) {
            this.onFailure(error);
            throw error;
        }
    }

    /**
     * Handle successful execution
     */
    onSuccess() {
        this.metrics.successfulRequests++;
        this.emit('success');

        if (this.state === 'HALF_OPEN') {
            this.halfOpenAttempts++;
            if (this.halfOpenAttempts >= this.maxHalfOpenAttempts) {
                this.moveToClosed();
            }
        } else if (this.state === 'CLOSED') {
            // Reset failure count on successful request
            this.failureCount = 0;
        }
    }

    /**
     * Handle failed execution
     * @param {Error} error - The error that occurred
     */
    onFailure(error) {
        this.metrics.failedRequests++;
        this.failureCount++;
        this.lastFailureTime = Date.now();

        this.emit('failure', error);

        // Check if we should open the circuit
        if (this.failureCount >= this.failureThreshold) {
            this.moveToOpen();
        }
    }

    /**
     * Move circuit to OPEN state
     */
    moveToOpen() {
        if (this.state !== 'OPEN') {
            this.state = 'OPEN';
            this.nextAttemptTime = Date.now() + this.recoveryTimeout;
            this.metrics.circuitOpenTime = Date.now();

            this.emit('stateChange', 'OPEN');
            this.emit('open');

            console.warn(`[${this.name}] Circuit breaker OPENED after ${this.failureCount} failures`);
        }
    }

    /**
     * Move circuit to HALF_OPEN state
     */
    moveToHalfOpen() {
        this.state = 'HALF_OPEN';
        this.halfOpenAttempts = 0;

        this.emit('stateChange', 'HALF_OPEN');
        this.emit('halfOpen');

        console.info(`[${this.name}] Circuit breaker moved to HALF_OPEN for testing`);
    }

    /**
     * Move circuit to CLOSED state
     */
    moveToClosed() {
        this.state = 'CLOSED';
        this.failureCount = 0;
        this.lastFailureTime = null;
        this.nextAttemptTime = null;
        this.halfOpenAttempts = 0;

        this.emit('stateChange', 'CLOSED');
        this.emit('close');

        console.info(`[${this.name}] Circuit breaker CLOSED - service recovered`);
    }

    /**
     * Check if we should attempt to reset the circuit
     * @returns {boolean}
     */
    shouldAttemptReset() {
        return Date.now() >= this.nextAttemptTime;
    }

    /**
     * Get current circuit breaker status
     * @returns {Object} Status information
     */
    getStatus() {
        return {
            state: this.state,
            failureCount: this.failureCount,
            failureThreshold: this.failureThreshold,
            lastFailureTime: this.lastFailureTime,
            nextAttemptTime: this.nextAttemptTime,
            metrics: { ...this.metrics },
            healthPercentage: this.getHealthPercentage()
        };
    }

    /**
     * Calculate health percentage based on recent requests
     * @returns {number} Health percentage (0-100)
     */
    getHealthPercentage() {
        const { successfulRequests, totalRequests } = this.metrics;
        if (totalRequests === 0) return 100;
        return Math.round((successfulRequests / totalRequests) * 100);
    }

    /**
     * Reset circuit breaker to initial state
     */
    reset() {
        this.state = 'CLOSED';
        this.failureCount = 0;
        this.lastFailureTime = null;
        this.nextAttemptTime = null;
        this.halfOpenAttempts = 0;

        // Reset metrics
        this.metrics = {
            totalRequests: 0,
            successfulRequests: 0,
            failedRequests: 0,
            timeouts: 0,
            circuitOpenTime: 0,
            lastResetTime: Date.now()
        };

        this.emit('stateChange', 'CLOSED');
        console.info(`[${this.name}] Circuit breaker manually reset`);
    }

    /**
     * Add event listener
     * @param {string} event - Event name
     * @param {Function} listener - Event listener function
     */
    on(event, listener) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(listener);
    }

    /**
     * Remove event listener
     * @param {string} event - Event name
     * @param {Function} listener - Event listener function
     */
    off(event, listener) {
        if (this.listeners[event]) {
            const index = this.listeners[event].indexOf(listener);
            if (index > -1) {
                this.listeners[event].splice(index, 1);
            }
        }
    }

    /**
     * Emit event to listeners
     * @param {string} event - Event name
     * @param {...*} args - Event arguments
     */
    emit(event, ...args) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(listener => {
                try {
                    listener(...args);
                } catch (error) {
                    console.error(`Error in circuit breaker event listener for ${event}:`, error);
                }
            });
        }
    }

    /**
     * Create a wrapped function with circuit breaker protection
     * @param {Function} fn - Function to wrap
     * @returns {Function} Wrapped function
     */
    wrap(fn) {
        return (...args) => this.execute(fn, ...args);
    }
}

/**
 * Circuit Breaker Error class
 */
class CircuitBreakerError extends Error {
    constructor(message, code = 'CIRCUIT_BREAKER_ERROR', retryAfter = null) {
        super(message);
        this.name = 'CircuitBreakerError';
        this.code = code;
        this.retryAfter = retryAfter;
        this.timestamp = new Date().toISOString();
    }
}

/**
 * Circuit Breaker Factory for creating multiple circuit breakers
 */
class CircuitBreakerFactory {
    constructor() {
        this.breakers = new Map();
    }

    /**
     * Get or create a circuit breaker for a specific service
     * @param {string} name - Service name
     * @param {Object} options - Circuit breaker options
     * @returns {CircuitBreaker}
     */
    getCircuitBreaker(name, options = {}) {
        if (!this.breakers.has(name)) {
            this.breakers.set(name, new CircuitBreaker({
                ...options,
                name
            }));
        }
        return this.breakers.get(name);
    }

    /**
     * Get status of all circuit breakers
     * @returns {Object} Status of all breakers
     */
    getAllStatus() {
        const status = {};
        for (const [name, breaker] of this.breakers) {
            status[name] = breaker.getStatus();
        }
        return status;
    }

    /**
     * Reset all circuit breakers
     */
    resetAll() {
        for (const breaker of this.breakers.values()) {
            breaker.reset();
        }
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { CircuitBreaker, CircuitBreakerError, CircuitBreakerFactory };
} else {
    window.CircuitBreaker = CircuitBreaker;
    window.CircuitBreakerError = CircuitBreakerError;
    window.CircuitBreakerFactory = CircuitBreakerFactory;
}

/**
 * Simple Server-Sent Events (SSE) Client
 * Lightweight wrapper for basic SSE functionality with automatic reconnection
 */

/**
 * Simple SSE client with automatic reconnection
 */
export class SSEClient {
    #eventSource = null;
    #url = null;
    #listeners = new Map();
    #reconnectAttempts = 0;
    #maxReconnectAttempts = 3;
    #reconnectDelay = 1000;
    #isConnected = false;

    constructor(options = {}) {
        this.options = {
            maxReconnectAttempts: 3,
            reconnectDelay: 1000,
            withCredentials: false,
            ...options
        };
        this.#maxReconnectAttempts = this.options.maxReconnectAttempts;
        this.#reconnectDelay = this.options.reconnectDelay;
    }

    /**
     * Connect to SSE endpoint
     * @param {string} url - The SSE endpoint URL
     * @param {Object} params - Query parameters
     */
    connect(url, params = {}) {
        this.#url = url;

        // Build URL with parameters
        const fullUrl = new URL(url, window.location.origin);
        Object.entries(params).forEach(([key, value]) => {
            fullUrl.searchParams.append(key, value);
        });

        try {
            this.#eventSource = new EventSource(fullUrl.toString(), {
                withCredentials: this.options.withCredentials
            });

            this.#eventSource.onopen = () => {
                this.#isConnected = true;
                this.#reconnectAttempts = 0;
                this.#emit('connected', { url: fullUrl.toString() });
            };

            this.#eventSource.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.#emit('message', data);
                } catch (error) {
                    console.warn('SSEClient: Failed to parse message data, using raw data:', error);
                    this.#emit('message', event.data);
                }
            };

            this.#eventSource.onerror = (event) => {
                this.#isConnected = false;

                if (this.#eventSource.readyState === EventSource.CLOSED) {
                    this.#emit('disconnected', { event });
                    this.#handleReconnection();
                } else {
                    this.#emit('error', { event });
                }
            };

        } catch (error) {
            console.error('SSEClient connection error:', error);
            this.#emit('error', { error });
        }
    }

    /**
     * Handle automatic reconnection
     */
    #handleReconnection() {
        if (this.#reconnectAttempts < this.#maxReconnectAttempts) {
            const delay = this.#reconnectDelay * Math.pow(2, this.#reconnectAttempts);

            setTimeout(() => {
                this.#reconnectAttempts++;
                this.#emit('reconnecting', {
                    attempt: this.#reconnectAttempts,
                    maxAttempts: this.#maxReconnectAttempts
                });
                this.connect(this.#url);
            }, delay);
        } else {
            this.#emit('failed', {
                reason: 'Max reconnection attempts reached',
                attempts: this.#reconnectAttempts
            });
        }
    }

    /**
     * Add event listener for a specific event type
     * @param {string} eventType - The event type from the server
     * @param {Function} callback - Callback function
     */
    addEventListener(eventType, callback) {
        if (!this.#listeners.has(eventType)) {
            this.#listeners.set(eventType, new Set());
        }
        this.#listeners.get(eventType).add(callback);

        // Add to EventSource if it's a custom event type
        if (this.#eventSource && !['connected', 'disconnected', 'error', 'message', 'reconnecting', 'failed'].includes(eventType)) {
            this.#eventSource.addEventListener(eventType, (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.#emit(eventType, data);
                } catch (error) {
                    console.warn(`SSEClient: Failed to parse ${eventType} data, using raw data:`, error);
                    this.#emit(eventType, event.data);
                }
            });
        }
    }

    /**
     * Remove event listener
     * @param {string} eventType - The event type
     * @param {Function} callback - The callback to remove
     */
    removeEventListener(eventType, callback) {
        if (this.#listeners.has(eventType)) {
            this.#listeners.get(eventType).delete(callback);
        }
    }

    /**
     * Emit event to all listeners
     * @param {string} eventType - The event type
     * @param {*} data - The event data
     */
    #emit(eventType, data) {
        if (this.#listeners.has(eventType)) {
            this.#listeners.get(eventType).forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`SSEClient: Error in ${eventType} listener`, error);
                }
            });
        }
    }

    /**
     * Close the SSE connection
     */
    disconnect() {
        if (this.#eventSource) {
            this.#eventSource.close();
            this.#eventSource = null;
        }
        this.#isConnected = false;
        this.#listeners.clear();
        this.#emit('disconnected', { reason: 'Manual disconnect' });
    }

    /**
     * Check if connected
     */
    get connected() {
        return this.#isConnected;
    }

    /**
     * Get connection state info
     */
    get state() {
        return {
            connected: this.#isConnected,
            url: this.#url,
            reconnectAttempts: this.#reconnectAttempts,
            readyState: this.#eventSource?.readyState
        };
    }

    /**
     * Internal method for testing - expose emit functionality
     * @private
     */
    _emit(eventType, data) {
        this.#emit(eventType, data);
    }

    /**
     * Internal method for testing - get event source
     * @private
     */
    _getEventSource() {
        return this.#eventSource;
    }
}

/**
 * Utility function to create a simple progress tracker
 * @param {string} taskId - The task ID to track
 * @param {Object} callbacks - Callback functions
 */
export function trackProgress(taskId, { onProgress, onComplete, onError, onConnect } = {}) {
    const client = new SSEClient();

    // Set up event listeners
    if (onConnect) client.addEventListener('connected', onConnect);
    if (onProgress) client.addEventListener('progress', onProgress);
    if (onComplete) client.addEventListener('completed', onComplete);
    if (onError) client.addEventListener('error', onError);

    // Connect to progress endpoint
    client.connect(`/api/llm/score-progress/${taskId}`);

    return {
        client,
        stop: () => client.disconnect()
    };
}

/**
 * Utility function to monitor feed health
 * @param {Object} callbacks - Callback functions
 */
export function monitorFeedHealth({ onHealthUpdate, onConnect, onError } = {}) {
    const client = new SSEClient();

    // Set up event listeners
    if (onConnect) client.addEventListener('connected', onConnect);
    if (onHealthUpdate) client.addEventListener('feed_health', onHealthUpdate);
    if (onError) client.addEventListener('error', onError);

    // Connect to feed health stream
    client.connect('/api/admin/feeds/health/stream');

    return {
        client,
        stop: () => client.disconnect()
    };
}

export default SSEClient;

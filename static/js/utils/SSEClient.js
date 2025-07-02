/**
 * Simple Server-Sent Events (SSE) Client
 * Lightweight wrapper for basic SSE functionality with automatic reconnection
 */

/**
 * Simple SSE client with automatic reconnection
 */
export class SSEClient {
    _eventSource = null;
    _url = null;
    _listeners = new Map();
    _reconnectAttempts = 0;
    _maxReconnectAttempts = 3;
    _reconnectDelay = 1000;
    _isConnected = false;

    constructor(options = {}) {
        this.options = {
            maxReconnectAttempts: 3,
            reconnectDelay: 1000,
            withCredentials: false,
            ...options
        };
        this._maxReconnectAttempts = this.options.maxReconnectAttempts;
        this._reconnectDelay = this.options.reconnectDelay;
    }

    /**
     * Connect to SSE endpoint
     * @param {string} url - The SSE endpoint URL
     * @param {Object} params - Query parameters
     */
    connect(url, params = {}) {
        this._url = url;

        // Build URL with parameters
        const fullUrl = new URL(url, window.location.origin);
        Object.entries(params).forEach(([key, value]) => {
            fullUrl.searchParams.append(key, value);
        });

        try {
            console.log('SSEClient: Connecting to', fullUrl.toString());

            this._eventSource = new EventSource(fullUrl.toString(), {
                withCredentials: this.options.withCredentials
            });

            // Register custom event listeners BEFORE setting up built-in handlers
            this._registerCustomEventListeners();

            this._eventSource.onopen = () => {
                console.log('SSEClient: Connection opened');
                this._isConnected = true;
                this._reconnectAttempts = 0;
                this._emit('connected', { url: fullUrl.toString() });
            };

            this._eventSource.onmessage = (event) => {
                console.log('SSEClient: Received message:', event.data);
                try {
                    const data = JSON.parse(event.data);
                    this._emit('message', data);
                } catch (error) {
                    console.warn('SSEClient: Failed to parse message data, using raw data:', error);
                    this._emit('message', event.data);
                }
            };

            this._eventSource.onerror = (event) => {
                console.error('SSEClient: Error event:', event);
                this._isConnected = false;

                // Check readyState to determine the type of error
                if (this._eventSource.readyState === EventSource.CLOSED) {
                    console.log('SSEClient: Connection closed');
                    this._emit('disconnected', { event, reason: 'Connection closed' });
                    this._handleReconnection();
                } else if (this._eventSource.readyState === EventSource.CONNECTING) {
                    console.log('SSEClient: Connection failed, will retry');
                    this._emit('error', { event, reason: 'Connection failed' });
                } else {
                    console.log('SSEClient: Unknown error state');
                    this._emit('error', { event, reason: 'Unknown error' });
                }
            };

        } catch (error) {
            console.error('SSEClient connection error:', error);
            this._emit('error', { error });
        }

        // Attach any custom event listeners that were registered before connection
        this._attachPendingEventListeners();
    }

    /**
     * Attach pending custom event listeners to the EventSource
     * This handles the timing issue where listeners are registered before connection
     */
    _attachPendingEventListeners() {
        if (!this._eventSource) {
            return;
        }

        } catch (error) {
            console.error('SSEClient: Failed to create EventSource:', error);
            this._emit('error', { error: error.message });
            throw error;
        }
    }

    /**
     * Register custom event listeners on the EventSource
     * This is called before setting up the connection to ensure events are captured
     */
    _registerCustomEventListeners() {
        if (!this._eventSource) return;

        // Register all custom event types that have listeners
        this._listeners.forEach((callbacks, eventType) => {
            // Skip built-in event types that are handled by onopen, onmessage, onerror
            if (!['connected', 'disconnected', 'error', 'message', 'reconnecting', 'failed'].includes(eventType)) {
                console.log(`SSEClient: Registering listener for event type: ${eventType}`);
                this._eventSource.addEventListener(eventType, (event) => {
                    console.log(`SSEClient: Received ${eventType} event:`, event.data);
                    try {
                        const data = JSON.parse(event.data);
                        this._emit(eventType, data);
                    } catch (error) {
                        console.warn(`SSEClient: Failed to parse ${eventType} data, using raw data:`, error);
                        this._emit(eventType, event.data);
                    }
                });
            }
        });
    }

    /**
     * Handle automatic reconnection
     */
    _handleReconnection() {
        if (this._reconnectAttempts < this._maxReconnectAttempts) {
            const delay = this._reconnectDelay * Math.pow(2, this._reconnectAttempts);

            setTimeout(() => {
                this._reconnectAttempts++;
                this._emit('reconnecting', {
                    attempt: this._reconnectAttempts,
                    maxAttempts: this._maxReconnectAttempts
                });
                this.connect(this._url);
            }, delay);
        } else {
            this._emit('failed', {
                reason: 'Max reconnection attempts reached',
                attempts: this._reconnectAttempts
            });
        }
    }

    /**
     * Add event listener for a specific event type
     * @param {string} eventType - The event type from the server
     * @param {Function} callback - Callback function
     */
    addEventListener(eventType, callback) {
        if (!this._listeners.has(eventType)) {
            this._listeners.set(eventType, new Set());
        }
        this._listeners.get(eventType).add(callback);

        // If EventSource already exists and this is a custom event type, attach immediately
        if (this._eventSource && !['connected', 'disconnected', 'error', 'message', 'reconnecting', 'failed'].includes(eventType)) {
            console.log(`SSEClient: Late-registering listener for event type: ${eventType}`);
            this._eventSource.addEventListener(eventType, (event) => {
                console.log(`SSEClient: Received ${eventType} event:`, event.data);
                try {
                    const data = JSON.parse(event.data);
                    this._emit(eventType, data);
                } catch (error) {
                    console.warn(`SSEClient: Failed to parse ${eventType} data, using raw data:`, error);
                    this._emit(eventType, event.data);
                }
            });
        }
        // If EventSource doesn't exist yet, the listener will be attached in _attachPendingEventListeners()
    }

    /**
     * Remove event listener
     * @param {string} eventType - The event type
     * @param {Function} callback - The callback to remove
     */
    removeEventListener(eventType, callback) {
        if (this._listeners.has(eventType)) {
            this._listeners.get(eventType).delete(callback);
        }
    }

    /**
     * Emit event to all listeners
     * @param {string} eventType - The event type
     * @param {*} data - The event data
     */
    _emit(eventType, data) {
        if (this._listeners.has(eventType)) {
            this._listeners.get(eventType).forEach(callback => {
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
        if (this._eventSource) {
            this._eventSource.close();
            this._eventSource = null;
        }
        this._isConnected = false;
        this._listeners.clear();
        this._emit('disconnected', { reason: 'Manual disconnect' });
    }

    /**
     * Check if connected
     */
    get connected() {
        return this._isConnected;
    }

    /**
     * Get connection state info
     */
    get state() {
        return {
            connected: this._isConnected,
            url: this._url,
            reconnectAttempts: this._reconnectAttempts,
            readyState: this._eventSource?.readyState
        };
    }

    /**
     * Internal method for testing - get event source
     * @private
     */
    _getEventSource() {
        return this._eventSource;
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

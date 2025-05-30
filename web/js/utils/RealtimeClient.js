/**
 * Real-time Communication Client
 * Provides SSE (Server-Sent Events) with WebSocket fallback for real-time features
 * Handles connection resilience, automatic reconnection, and error recovery
 */

import { ErrorHandler } from './ErrorHandler.js';

/**
 * Event types for real-time communication
 */
export const EVENT_TYPES = {
    PROGRESS: 'progress',
    COMPLETED: 'completed',
    ERROR: 'error',
    FEED_HEALTH: 'feed_health',
    BIAS_UPDATE: 'bias_update',
    SYSTEM_STATUS: 'system_status'
};

/**
 * Connection states
 */
export const CONNECTION_STATES = {
    DISCONNECTED: 'disconnected',
    CONNECTING: 'connecting',
    CONNECTED: 'connected',
    RECONNECTING: 'reconnecting',
    FAILED: 'failed'
};

/**
 * Enhanced real-time client with SSE and WebSocket fallback
 */
export class RealtimeClient {
    #eventSource = null;
    #websocket = null;
    #useWebSocket = false;
    #endpoint = null;
    #listeners = new Map();
    #connectionState = CONNECTION_STATES.DISCONNECTED;
    #reconnectAttempts = 0;
    #maxReconnectAttempts = 5;
    #reconnectTimer = null;
    #heartbeatInterval = null;
    #lastHeartbeat = null;

    constructor(options = {}) {
        this.options = {
            heartbeatInterval: 30000, // 30 seconds
            maxReconnectAttempts: 5,
            reconnectDelay: 1000, // Start with 1 second
            maxReconnectDelay: 30000, // Max 30 seconds
            forceWebSocket: false,
            ...options
        };

        this.#maxReconnectAttempts = this.options.maxReconnectAttempts;
        this.#useWebSocket = this.options.forceWebSocket;
    }

    /**
     * Connect to a real-time endpoint
     * @param {string} endpoint - The endpoint URL (HTTP for SSE, WS for WebSocket)
     * @param {Object} params - Additional connection parameters
     */
    async connect(endpoint, params = {}) {
        this.#endpoint = endpoint;
        this.#setConnectionState(CONNECTION_STATES.CONNECTING);

        try {
            if (this.#useWebSocket) {
                await this.#connectWebSocket(endpoint, params);
            } else {
                await this.#connectSSE(endpoint, params);
            }
        } catch (error) {
            console.error('RealtimeClient: Connection failed', error);
            this.#handleConnectionError(error);
        }
    }

    /**
     * Connect using Server-Sent Events
     */
    async #connectSSE(endpoint, params) {
        // Build URL with parameters
        const url = new URL(endpoint, window.location.origin);
        Object.entries(params).forEach(([key, value]) => {
            url.searchParams.append(key, value);
        });

        this.#eventSource = new EventSource(url.toString());

        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                reject(new Error('SSE connection timeout'));
            }, 10000);

            this.#eventSource.onopen = () => {
                clearTimeout(timeout);
                this.#setConnectionState(CONNECTION_STATES.CONNECTED);
                this.#reconnectAttempts = 0;
                this.#startHeartbeat();
                resolve();
            };

            this.#eventSource.onmessage = (event) => {
                this.#handleMessage(event.data);
            };

            this.#eventSource.onerror = (event) => {
                clearTimeout(timeout);
                if (this.#eventSource.readyState === EventSource.CLOSED) {
                    this.#handleConnectionError(new Error('SSE connection closed'));
                } else {
                    // Temporary error, SSE will auto-reconnect
                    console.warn('RealtimeClient: SSE temporary error', event);
                }
            };

            // Handle specific event types
            Object.values(EVENT_TYPES).forEach(eventType => {
                this.#eventSource.addEventListener(eventType, (event) => {
                    this.#handleMessage(event.data, eventType);
                });
            });
        });
    }

    /**
     * Connect using WebSocket
     */
    async #connectWebSocket(endpoint, params) {
        // Convert HTTP endpoint to WebSocket URL
        const wsUrl = endpoint.replace(/^https?:/, window.location.protocol === 'https:' ? 'wss:' : 'ws:');
        const url = new URL(wsUrl, window.location.origin);

        // Add parameters as query string for WebSocket
        Object.entries(params).forEach(([key, value]) => {
            url.searchParams.append(key, value);
        });

        this.#websocket = new WebSocket(url.toString());

        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                reject(new Error('WebSocket connection timeout'));
            }, 10000);

            this.#websocket.onopen = () => {
                clearTimeout(timeout);
                this.#setConnectionState(CONNECTION_STATES.CONNECTED);
                this.#reconnectAttempts = 0;
                this.#startHeartbeat();
                resolve();
            };

            this.#websocket.onmessage = (event) => {
                this.#handleMessage(event.data);
            };

            this.#websocket.onclose = (event) => {
                clearTimeout(timeout);
                if (event.code !== 1000) { // Not a normal closure
                    this.#handleConnectionError(new Error(`WebSocket closed with code ${event.code}`));
                }
            };

            this.#websocket.onerror = (event) => {
                clearTimeout(timeout);
                this.#handleConnectionError(new Error('WebSocket error'));
            };
        });
    }

    /**
     * Handle incoming messages
     */
    #handleMessage(data, eventType = null) {
        try {
            const message = JSON.parse(data);
            const type = eventType || message.type || EVENT_TYPES.PROGRESS;

            // Update last heartbeat for connection health
            this.#lastHeartbeat = Date.now();

            // Emit to listeners
            this.#emit(type, message);

            // Handle special message types
            if (type === EVENT_TYPES.ERROR) {
                this.#handleServerError(message);
            }
        } catch (error) {
            console.error('RealtimeClient: Failed to parse message', error, data);
        }
    }

    /**
     * Handle server-sent errors
     */
    #handleServerError(errorMessage) {
        const error = new Error(errorMessage.error?.message || 'Server error');
        error.code = errorMessage.error?.code;
        error.retryable = errorMessage.error?.retryable !== false;

        ErrorHandler.handleError(error, { context: 'RealtimeClient' });

        if (!error.retryable) {
            this.disconnect();
        }
    }

    /**
     * Handle connection errors and implement reconnection logic
     */
    #handleConnectionError(error) {
        console.error('RealtimeClient: Connection error', error);

        this.#setConnectionState(CONNECTION_STATES.FAILED);
        this.#cleanup();

        // Try WebSocket fallback if SSE failed
        if (!this.#useWebSocket && this.#reconnectAttempts === 0) {
            console.log('RealtimeClient: Trying WebSocket fallback');
            this.#useWebSocket = true;
            this.#reconnect();
            return;
        }

        // Implement exponential backoff reconnection
        if (this.#reconnectAttempts < this.#maxReconnectAttempts) {
            const delay = Math.min(
                this.options.reconnectDelay * Math.pow(2, this.#reconnectAttempts),
                this.options.maxReconnectDelay
            );

            console.log(`RealtimeClient: Reconnecting in ${delay}ms (attempt ${this.#reconnectAttempts + 1})`);

            this.#setConnectionState(CONNECTION_STATES.RECONNECTING);
            this.#reconnectTimer = setTimeout(() => {
                this.#reconnect();
            }, delay);
        } else {
            console.error('RealtimeClient: Max reconnection attempts reached');
            this.#emit('connection_failed', { error, attempts: this.#reconnectAttempts });
        }
    }

    /**
     * Reconnect to the endpoint
     */
    async #reconnect() {
        this.#reconnectAttempts++;
        try {
            await this.connect(this.#endpoint);
        } catch (error) {
            this.#handleConnectionError(error);
        }
    }

    /**
     * Start heartbeat monitoring
     */
    #startHeartbeat() {
        this.#lastHeartbeat = Date.now();
        this.#heartbeatInterval = setInterval(() => {
            const timeSinceLastHeartbeat = Date.now() - this.#lastHeartbeat;

            if (timeSinceLastHeartbeat > this.options.heartbeatInterval * 2) {
                console.warn('RealtimeClient: Heartbeat timeout, connection may be stale');
                this.#handleConnectionError(new Error('Heartbeat timeout'));
            }
        }, this.options.heartbeatInterval);
    }

    /**
     * Set connection state and emit state change events
     */
    #setConnectionState(state) {
        if (this.#connectionState !== state) {
            const previousState = this.#connectionState;
            this.#connectionState = state;
            this.#emit('connection_state_changed', {
                state,
                previousState,
                reconnectAttempts: this.#reconnectAttempts
            });
        }
    }

    /**
     * Clean up connections and timers
     */
    #cleanup() {
        if (this.#eventSource) {
            this.#eventSource.close();
            this.#eventSource = null;
        }

        if (this.#websocket) {
            this.#websocket.close();
            this.#websocket = null;
        }

        if (this.#reconnectTimer) {
            clearTimeout(this.#reconnectTimer);
            this.#reconnectTimer = null;
        }

        if (this.#heartbeatInterval) {
            clearInterval(this.#heartbeatInterval);
            this.#heartbeatInterval = null;
        }
    }

    /**
     * Add event listener
     * @param {string} eventType - The event type to listen for
     * @param {Function} callback - The callback function
     */
    addEventListener(eventType, callback) {
        if (!this.#listeners.has(eventType)) {
            this.#listeners.set(eventType, new Set());
        }
        this.#listeners.get(eventType).add(callback);
    }

    /**
     * Remove event listener
     * @param {string} eventType - The event type
     * @param {Function} callback - The callback function to remove
     */
    removeEventListener(eventType, callback) {
        if (this.#listeners.has(eventType)) {
            this.#listeners.get(eventType).delete(callback);
        }
    }

    /**
     * Emit event to listeners
     */
    #emit(eventType, data) {
        if (this.#listeners.has(eventType)) {
            this.#listeners.get(eventType).forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`RealtimeClient: Error in ${eventType} listener`, error);
                }
            });
        }
    }

    /**
     * Send message through WebSocket (if connected via WebSocket)
     * @param {Object} message - Message to send
     */
    send(message) {
        if (this.#websocket && this.#websocket.readyState === WebSocket.OPEN) {
            this.#websocket.send(JSON.stringify(message));
        } else {
            throw new Error('Cannot send message: WebSocket not connected');
        }
    }

    /**
     * Disconnect from the real-time endpoint
     */
    disconnect() {
        this.#setConnectionState(CONNECTION_STATES.DISCONNECTED);
        this.#cleanup();
        this.#listeners.clear();
    }

    /**
     * Get current connection state
     */
    get connectionState() {
        return this.#connectionState;
    }

    /**
     * Check if currently connected
     */
    get isConnected() {
        return this.#connectionState === CONNECTION_STATES.CONNECTED;
    }

    /**
     * Get connection info
     */
    get connectionInfo() {
        return {
            state: this.#connectionState,
            transportType: this.#useWebSocket ? 'websocket' : 'sse',
            reconnectAttempts: this.#reconnectAttempts,
            endpoint: this.#endpoint,
            lastHeartbeat: this.#lastHeartbeat
        };
    }
}

/**
 * Progress tracking client specifically for article analysis
 */
export class ProgressTracker extends RealtimeClient {
    constructor(options = {}) {
        super({
            ...options,
            heartbeatInterval: 10000 // More frequent heartbeat for progress tracking
        });
    }

    /**
     * Track progress for an article analysis task
     * @param {string} taskId - The analysis task ID
     * @param {Function} onProgress - Progress callback
     * @param {Function} onComplete - Completion callback
     * @param {Function} onError - Error callback
     */
    async trackProgress(taskId, { onProgress, onComplete, onError } = {}) {
        // Set up event listeners
        if (onProgress) {
            this.addEventListener(EVENT_TYPES.PROGRESS, onProgress);
        }
        if (onComplete) {
            this.addEventListener(EVENT_TYPES.COMPLETED, onComplete);
        }
        if (onError) {
            this.addEventListener(EVENT_TYPES.ERROR, onError);
        }

        // Connect to progress endpoint
        await this.connect(`/api/llm/score-progress/${taskId}`);
    }

    /**
     * Stop tracking progress and clean up
     */
    stopTracking() {
        this.disconnect();
    }
}

/**
 * Feed health monitoring client
 */
export class FeedHealthMonitor extends RealtimeClient {
    constructor(options = {}) {
        super({
            ...options,
            heartbeatInterval: 30000 // Standard heartbeat for feed monitoring
        });
    }

    /**
     * Start monitoring feed health
     * @param {Function} onHealthUpdate - Health update callback
     */
    async startMonitoring(onHealthUpdate) {
        if (onHealthUpdate) {
            this.addEventListener(EVENT_TYPES.FEED_HEALTH, onHealthUpdate);
        }

        // Connect to feed health stream
        await this.connect('/api/admin/feeds/health/stream');
    }

    /**
     * Stop monitoring feed health
     */
    stopMonitoring() {
        this.disconnect();
    }
}

// Export singleton instances for common use cases
export const progressTracker = new ProgressTracker();
export const feedHealthMonitor = new FeedHealthMonitor();

// Export default RealtimeClient for custom implementations
export default RealtimeClient;

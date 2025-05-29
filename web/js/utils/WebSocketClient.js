/**
 * WebSocket Client
 * Enhanced WebSocket wrapper with automatic reconnection and message handling
 */

import { ErrorHandler } from './ErrorHandler.js';

/**
 * Connection states for WebSocket
 */
export const WS_STATES = {
    CONNECTING: 'connecting',
    CONNECTED: 'connected',
    DISCONNECTED: 'disconnected',
    RECONNECTING: 'reconnecting',
    FAILED: 'failed'
};

/**
 * Enhanced WebSocket client with reconnection and message management
 */
export class WebSocketClient {
    #websocket = null;
    #url = null;
    #protocols = null;
    #listeners = new Map();
    #reconnectAttempts = 0;
    #maxReconnectAttempts = 5;
    #reconnectDelay = 1000;
    #maxReconnectDelay = 30000;
    #reconnectTimer = null;
    #heartbeatInterval = null;
    #heartbeatTimer = null;
    #state = WS_STATES.DISCONNECTED;
    #messageQueue = [];
    #queueMessages = true;

    constructor(options = {}) {
        this.options = {
            maxReconnectAttempts: 5,
            reconnectDelay: 1000,
            maxReconnectDelay: 30000,
            heartbeatInterval: 30000,
            queueMessages: true,
            maxQueueSize: 100,
            ...options
        };

        this.#maxReconnectAttempts = this.options.maxReconnectAttempts;
        this.#reconnectDelay = this.options.reconnectDelay;
        this.#maxReconnectDelay = this.options.maxReconnectDelay;
        this.#queueMessages = this.options.queueMessages;
    }

    /**
     * Connect to WebSocket endpoint
     * @param {string} url - The WebSocket URL
     * @param {string|string[]} protocols - WebSocket protocols
     */
    connect(url, protocols = []) {
        this.#url = url;
        this.#protocols = protocols;
        this.#setState(WS_STATES.CONNECTING);

        try {
            this.#websocket = new WebSocket(url, protocols);

            this.#websocket.onopen = (event) => {
                this.#setState(WS_STATES.CONNECTED);
                this.#reconnectAttempts = 0;
                this.#startHeartbeat();
                this.#processMessageQueue();
                this.#emit('connected', { event, url });
            };

            this.#websocket.onmessage = (event) => {
                this.#handleMessage(event);
            };

            this.#websocket.onclose = (event) => {
                this.#setState(WS_STATES.DISCONNECTED);
                this.#stopHeartbeat();
                this.#emit('disconnected', { event, code: event.code, reason: event.reason });

                // Attempt reconnection unless it was a clean close
                if (event.code !== 1000 && event.code !== 1001) {
                    this.#handleReconnection();
                }
            };

            this.#websocket.onerror = (event) => {
                this.#emit('error', { event });
                ErrorHandler.handleError(new Error('WebSocket error'), { 
                    context: 'WebSocketClient',
                    url: this.#url
                });
            };

        } catch (error) {
            this.#setState(WS_STATES.FAILED);
            this.#emit('error', { error });
            ErrorHandler.handleError(error, { context: 'WebSocketClient.connect' });
        }
    }

    /**
     * Handle incoming messages
     * @param {MessageEvent} event - The message event
     */
    #handleMessage(event) {
        try {
            // Try to parse as JSON
            let data;
            try {
                data = JSON.parse(event.data);
            } catch {
                data = event.data;
            }

            // Handle heartbeat/pong messages
            if (data?.type === 'pong' || data === 'pong') {
                this.#resetHeartbeat();
                return;
            }

            // Emit message events
            this.#emit('message', { data, originalEvent: event });

            // Emit specific event types if the message has a type
            if (data?.type) {
                this.#emit(data.type, data);
            }

        } catch (error) {
            console.error('WebSocketClient: Error handling message', error);
            this.#emit('message_error', { error, event });
        }
    }

    /**
     * Handle reconnection logic
     */
    #handleReconnection() {
        if (this.#reconnectAttempts < this.#maxReconnectAttempts) {
            const delay = Math.min(
                this.#reconnectDelay * Math.pow(2, this.#reconnectAttempts),
                this.#maxReconnectDelay
            );

            this.#setState(WS_STATES.RECONNECTING);
            this.#emit('reconnecting', { 
                attempt: this.#reconnectAttempts + 1, 
                maxAttempts: this.#maxReconnectAttempts,
                delay 
            });

            this.#reconnectTimer = setTimeout(() => {
                this.#reconnectAttempts++;
                this.connect(this.#url, this.#protocols);
            }, delay);

        } else {
            this.#setState(WS_STATES.FAILED);
            this.#emit('failed', { 
                reason: 'Max reconnection attempts reached',
                attempts: this.#reconnectAttempts
            });
        }
    }

    /**
     * Set connection state
     */
    #setState(state) {
        if (this.#state !== state) {
            const previousState = this.#state;
            this.#state = state;
            this.#emit('state_changed', { state, previousState });
        }
    }

    /**
     * Start heartbeat monitoring
     */
    #startHeartbeat() {
        if (this.options.heartbeatInterval > 0) {
            this.#heartbeatTimer = setInterval(() => {
                if (this.#websocket?.readyState === WebSocket.OPEN) {
                    this.send({ type: 'ping' });
                }
            }, this.options.heartbeatInterval);
        }
    }

    /**
     * Stop heartbeat monitoring
     */
    #stopHeartbeat() {
        if (this.#heartbeatTimer) {
            clearInterval(this.#heartbeatTimer);
            this.#heartbeatTimer = null;
        }
    }

    /**
     * Reset heartbeat timer
     */
    #resetHeartbeat() {
        // Heartbeat received, connection is alive
        // Could implement connection health tracking here
    }

    /**
     * Process queued messages when connection is established
     */
    #processMessageQueue() {
        if (this.#messageQueue.length > 0) {
            const queue = [...this.#messageQueue];
            this.#messageQueue = [];
            
            queue.forEach(message => {
                this.send(message, false); // Don't queue again
            });
        }
    }

    /**
     * Send a message through the WebSocket
     * @param {*} message - The message to send
     * @param {boolean} queue - Whether to queue if not connected
     */
    send(message, queue = true) {
        const data = typeof message === 'string' ? message : JSON.stringify(message);

        if (this.#websocket?.readyState === WebSocket.OPEN) {
            try {
                this.#websocket.send(data);
                this.#emit('message_sent', { message });
            } catch (error) {
                this.#emit('send_error', { error, message });
                ErrorHandler.handleError(error, { context: 'WebSocketClient.send' });
            }
        } else if (queue && this.#queueMessages && this.#state !== WS_STATES.FAILED) {
            // Queue message for when connection is restored
            if (this.#messageQueue.length < this.options.maxQueueSize) {
                this.#messageQueue.push(message);
                this.#emit('message_queued', { message, queueLength: this.#messageQueue.length });
            } else {
                this.#emit('queue_full', { message, queueLength: this.#messageQueue.length });
            }
        } else {
            throw new Error(`Cannot send message: WebSocket not connected (state: ${this.#state})`);
        }
    }

    /**
     * Add event listener
     * @param {string} eventType - The event type
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
     * @param {Function} callback - The callback to remove
     */
    removeEventListener(eventType, callback) {
        if (this.#listeners.has(eventType)) {
            this.#listeners.get(eventType).delete(callback);
        }
    }

    /**
     * Emit event to listeners
     * @param {string} eventType - The event type
     * @param {*} data - The event data
     */
    #emit(eventType, data) {
        if (this.#listeners.has(eventType)) {
            this.#listeners.get(eventType).forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`WebSocketClient: Error in ${eventType} listener`, error);
                }
            });
        }
    }

    /**
     * Disconnect from WebSocket
     * @param {number} code - Close code
     * @param {string} reason - Close reason
     */
    disconnect(code = 1000, reason = 'Client disconnect') {
        // Clear reconnection timer
        if (this.#reconnectTimer) {
            clearTimeout(this.#reconnectTimer);
            this.#reconnectTimer = null;
        }

        // Stop heartbeat
        this.#stopHeartbeat();

        // Close WebSocket
        if (this.#websocket) {
            try {
                this.#websocket.close(code, reason);
            } catch (error) {
                console.error('WebSocketClient: Error closing connection', error);
            }
            this.#websocket = null;
        }

        this.#setState(WS_STATES.DISCONNECTED);
        this.#listeners.clear();
        this.#messageQueue = [];
    }

    /**
     * Force reconnection
     */
    reconnect() {
        this.disconnect(1000, 'Manual reconnect');
        setTimeout(() => {
            this.#reconnectAttempts = 0;
            this.connect(this.#url, this.#protocols);
        }, 100);
    }

    /**
     * Get connection state
     */
    get state() {
        return this.#state;
    }

    /**
     * Check if connected
     */
    get connected() {
        return this.#state === WS_STATES.CONNECTED;
    }

    /**
     * Get connection info
     */
    get info() {
        return {
            state: this.#state,
            url: this.#url,
            protocols: this.#protocols,
            reconnectAttempts: this.#reconnectAttempts,
            queueLength: this.#messageQueue.length,
            readyState: this.#websocket?.readyState
        };
    }

    /**
     * Get WebSocket ready state constants
     */
    static get READY_STATES() {
        return {
            CONNECTING: WebSocket.CONNECTING,
            OPEN: WebSocket.OPEN,
            CLOSING: WebSocket.CLOSING,
            CLOSED: WebSocket.CLOSED
        };
    }
}

export default WebSocketClient;

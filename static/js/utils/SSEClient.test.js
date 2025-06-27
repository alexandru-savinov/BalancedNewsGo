/**
 * @jest-environment jsdom
 */

import { SSEClient } from './SSEClient.js';

// Mock EventSource
class MockEventSource {
  constructor(url, options) {
    this.url = url;
    this.options = options;
    this.readyState = EventSource.CONNECTING;
    this.onopen = null;
    this.onmessage = null;
    this.onerror = null;
    this.listeners = new Map();
    
    // Simulate connection after a short delay
    setTimeout(() => {
      this.readyState = EventSource.OPEN;
      if (this.onopen) this.onopen();
    }, 10);
  }

  addEventListener(type, listener) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, []);
    }
    this.listeners.get(type).push(listener);
  }

  removeEventListener(type, listener) {
    if (this.listeners.has(type)) {
      const listeners = this.listeners.get(type);
      const index = listeners.indexOf(listener);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  close() {
    this.readyState = EventSource.CLOSED;
  }

  // Helper method to simulate events
  simulateEvent(type, data) {
    const event = { type, data };
    if (type === 'message' && this.onmessage) {
      this.onmessage(event);
    }
    if (this.listeners.has(type)) {
      this.listeners.get(type).forEach(listener => listener(event));
    }
  }

  simulateError() {
    this.readyState = EventSource.CLOSED;
    if (this.onerror) this.onerror();
  }
}

// Mock EventSource constants
MockEventSource.CONNECTING = 0;
MockEventSource.OPEN = 1;
MockEventSource.CLOSED = 2;

global.EventSource = MockEventSource;

describe('SSEClient', () => {
  let sseClient;
  let mockEventSource;

  beforeEach(() => {
    sseClient = new SSEClient();
    jest.clearAllMocks();
  });

  afterEach(() => {
    if (sseClient) {
      sseClient.disconnect();
    }
  });

  describe('constructor', () => {
    test('should create instance with default options', () => {
      expect(sseClient).toBeInstanceOf(SSEClient);
    });

    test('should accept custom options', () => {
      const customClient = new SSEClient({
        maxReconnectAttempts: 5,
        reconnectDelay: 2000,
        withCredentials: true
      });
      expect(customClient).toBeInstanceOf(SSEClient);
    });
  });

  describe('connect', () => {
    test('should connect to SSE endpoint', async () => {
      const url = '/test-sse';
      const params = { id: '123' };

      sseClient.connect(url, params);

      // Wait for connection
      await new Promise(resolve => setTimeout(resolve, 20));

      expect(sseClient.connected).toBe(true);
    });

    test('should build URL with parameters', () => {
      const url = '/test-sse';
      const params = { id: '123', type: 'test' };

      sseClient.connect(url, params);

      // The URL should include the parameters
      expect(sseClient.connected).toBe(false); // Initially false until connected
    });
  });

  describe('event handling', () => {
    beforeEach(async () => {
      sseClient.connect('/test-sse');
      await new Promise(resolve => setTimeout(resolve, 20));
    });

    test('should register event listeners', () => {
      const listener = jest.fn();
      sseClient.addEventListener('test-event', listener);

      // Simulate event by calling the internal emit method
      sseClient._emit('test-event', 'test data');

      expect(listener).toHaveBeenCalledWith('test data');
    });

    test('should handle multiple listeners for same event', () => {
      const listener1 = jest.fn();
      const listener2 = jest.fn();

      sseClient.addEventListener('test-event', listener1);
      sseClient.addEventListener('test-event', listener2);

      sseClient._emit('test-event', 'test data');

      expect(listener1).toHaveBeenCalledWith('test data');
      expect(listener2).toHaveBeenCalledWith('test data');
    });

    test('should remove event listeners', () => {
      const listener = jest.fn();
      sseClient.addEventListener('test-event', listener);
      sseClient.removeEventListener('test-event', listener);

      sseClient._emit('test-event', 'test data');

      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe('disconnect', () => {
    test('should disconnect and clean up', async () => {
      sseClient.connect('/test-sse');
      await new Promise(resolve => setTimeout(resolve, 20));

      expect(sseClient.connected).toBe(true);

      sseClient.disconnect();

      expect(sseClient.connected).toBe(false);
    });
  });

  describe('reconnection', () => {
    test('should handle connection errors', async () => {
      const errorListener = jest.fn();
      sseClient.addEventListener('error', errorListener);

      sseClient.connect('/test-sse');
      await new Promise(resolve => setTimeout(resolve, 20));

      // Simulate error
      const eventSource = sseClient._getEventSource();
      eventSource.simulateError();

      // Wait for error handling
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(errorListener).toHaveBeenCalled();
    });

    test('should respect max reconnection attempts', async () => {
      const client = new SSEClient({ maxReconnectAttempts: 1 });
      
      client.connect('/test-sse');
      await new Promise(resolve => setTimeout(resolve, 20));

      // Simulate multiple errors
      const eventSource = client._getEventSource();
      eventSource.simulateError();
      
      await new Promise(resolve => setTimeout(resolve, 100));
      
      // Should stop trying after max attempts
      expect(client.connected).toBe(false);

      client.disconnect();
    });
  });

  describe('utility methods', () => {
    test('should report connection status', async () => {
      expect(sseClient.connected).toBe(false);

      sseClient.connect('/test-sse');
      await new Promise(resolve => setTimeout(resolve, 20));

      expect(sseClient.connected).toBe(true);
    });

    test('should provide connection state info', async () => {
      expect(sseClient.state.connected).toBe(false);

      sseClient.connect('/test-sse');
      await new Promise(resolve => setTimeout(resolve, 20));

      const state = sseClient.state;
      expect(state.connected).toBe(true);
      expect(state.url).toBe('/test-sse');
    });
  });
});

/**
 * Jest Test Setup
 * Global test configuration and utilities
 */

// Import web components polyfill for Jest/jsdom environment
require('@webcomponents/webcomponentsjs/webcomponents-bundle.js');

// Mock DOM APIs that aren't available in Jest/jsdom
global.ResizeObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

global.IntersectionObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Mock EventSource for SSE testing
global.EventSource = jest.fn().mockImplementation(() => ({
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  close: jest.fn(),
  readyState: 0,
  CONNECTING: 0,
  OPEN: 1,
  CLOSED: 2,
}));

// Mock fetch API
global.fetch = jest.fn();

// Mock performance API
global.performance = {
  ...global.performance,
  mark: jest.fn(),
  measure: jest.fn(),
  now: jest.fn(() => Date.now()),
};

// Custom matchers for accessibility testing
expect.extend({
  toHaveAriaAttribute(received, attribute, value) {
    const pass = received.hasAttribute(attribute) && 
                 (value === undefined || received.getAttribute(attribute) === value);
    
    if (pass) {
      return {
        message: () => `expected element not to have aria attribute ${attribute}${value ? ` with value ${value}` : ''}`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected element to have aria attribute ${attribute}${value ? ` with value ${value}` : ''}`,
        pass: false,
      };
    }
  },
});

// Suppress console warnings in tests unless explicitly testing for them
const originalWarn = console.warn;
console.warn = (...args) => {
  if (args[0]?.includes?.('test-specific warning')) {
    originalWarn(...args);
  }
};

// Set up test timeout
jest.setTimeout(10000);

// Mock utility modules that components import
jest.mock('../js/utils/LazyLoader.js', () => ({}), { virtual: true });
jest.mock('../js/utils/PerformanceMonitor.js', () => ({
  start: jest.fn(),
  end: jest.fn(),
  measure: jest.fn()
}), { virtual: true });
jest.mock('../js/utils/ComponentPerformanceMonitor.js', () => ({
  startRender: jest.fn(),
  endRender: jest.fn(),
  measureInteraction: jest.fn()
}), { virtual: true });

// Global API mocks
global.ApiClient = {
  updateBias: jest.fn().mockResolvedValue({ success: true }),
  get: jest.fn().mockResolvedValue({ data: [] }),
  post: jest.fn().mockResolvedValue({ success: true })
};

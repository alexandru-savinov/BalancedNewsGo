// Jest setup file to provide DOM environment for web components
global.EventSource = class EventSource {
  constructor(url) {
    this.url = url;
    this.readyState = 1;
    this.onopen = null;
    this.onmessage = null;
    this.onerror = null;
  }
  
  addEventListener(event, callback) {
    this[`on${event}`] = callback;
  }
  
  removeEventListener() {}
  
  close() {
    this.readyState = 2;
  }
};

// Mock fetch for API calls
global.fetch = jest.fn(() =>
  Promise.resolve({
    ok: true,
    json: () => Promise.resolve({}),
  })
);

// Create global window APIs if needed
global.ResizeObserver = class ResizeObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
};

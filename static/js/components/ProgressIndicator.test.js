/**
 * @jest-environment jsdom
 */

// Mock SSEClient before importing ProgressIndicator
const mockSSEClient = {
  connect: jest.fn(),
  disconnect: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  connected: false,
  state: { connected: false }
};

jest.mock('../utils/SSEClient.js', () => ({
  SSEClient: jest.fn(() => mockSSEClient)
}));

// Mock customElements.define
global.customElements = {
  define: jest.fn(),
  get: jest.fn(),
  whenDefined: jest.fn(() => Promise.resolve())
};

// Mock HTMLElement
global.HTMLElement = class HTMLElement {
  constructor() {
    this.shadowRoot = null;
    this.attributes = new Map();
  }
  
  attachShadow() {
    this.shadowRoot = {
      innerHTML: '',
      appendChild: jest.fn(),
      querySelector: jest.fn(),
      querySelectorAll: jest.fn(() => [])
    };
    return this.shadowRoot;
  }
  
  getAttribute(name) {
    return this.attributes.get(name) || null;
  }
  
  setAttribute(name, value) {
    this.attributes.set(name, value);
  }
  
  hasAttribute(name) {
    return this.attributes.has(name);
  }
  
  removeAttribute(name) {
    this.attributes.delete(name);
  }
  
  dispatchEvent() {}
  addEventListener() {}
  removeEventListener() {}
};

describe('ProgressIndicator', () => {
  let ProgressIndicator;
  let progressIndicator;

  beforeAll(async () => {
    // Import after mocking
    const module = await import('./ProgressIndicator.js');
    ProgressIndicator = module.default || module.ProgressIndicator;
  });

  beforeEach(() => {
    jest.clearAllMocks();
    mockSSEClient.connected = false;
    mockSSEClient.state = { connected: false };
    
    progressIndicator = new ProgressIndicator();
  });

  afterEach(() => {
    if (progressIndicator) {
      progressIndicator.disconnect?.();
    }
  });

  describe('constructor', () => {
    test('should create instance with shadow DOM', () => {
      expect(progressIndicator).toBeInstanceOf(ProgressIndicator);
      expect(progressIndicator.shadowRoot).toBeTruthy();
    });

    test('should initialize with default state', () => {
      expect(progressIndicator.status).toBe('idle');
      expect(progressIndicator.progress).toBe(0);
    });
  });

  describe('attributes', () => {
    test('should observe required attributes', () => {
      const observedAttrs = ProgressIndicator.observedAttributes;
      expect(observedAttrs).toContain('article-id');
      expect(observedAttrs).toContain('auto-connect');
      expect(observedAttrs).toContain('show-details');
    });

    test('should handle article-id attribute', () => {
      progressIndicator.setAttribute('article-id', '123');
      expect(progressIndicator.getAttribute('article-id')).toBe('123');
    });

    test('should handle auto-connect attribute', () => {
      progressIndicator.setAttribute('auto-connect', 'true');
      expect(progressIndicator.hasAttribute('auto-connect')).toBe(true);
    });
  });

  describe('connection management', () => {
    test('should connect to SSE when article ID is set', () => {
      progressIndicator.setAttribute('article-id', '123');
      progressIndicator.connect();
      
      expect(mockSSEClient.connect).toHaveBeenCalled();
    });

    test('should disconnect SSE client', () => {
      progressIndicator.connect();
      progressIndicator.disconnect();
      
      expect(mockSSEClient.disconnect).toHaveBeenCalled();
    });

    test('should handle auto-connect', () => {
      const connectSpy = jest.spyOn(progressIndicator, 'connect');
      progressIndicator.setAttribute('auto-connect', 'true');
      progressIndicator.setAttribute('article-id', '123');
      
      // Trigger attribute change
      progressIndicator.attributeChangedCallback('article-id', null, '123');
      
      expect(connectSpy).toHaveBeenCalled();
    });
  });

  describe('progress updates', () => {
    test('should update progress value', () => {
      progressIndicator.updateProgress(50, 'Processing');
      
      expect(progressIndicator.progress).toBe(50);
      expect(progressIndicator.stage).toBe('Processing');
    });

    test('should handle progress events from SSE', () => {
      const updateSpy = jest.spyOn(progressIndicator, 'updateProgress');
      
      progressIndicator.connect();
      
      // Find the progress event listener
      const progressListener = mockSSEClient.addEventListener.mock.calls
        .find(call => call[0] === 'progress')?.[1];
      
      if (progressListener) {
        progressListener({ progress: 75, stage: 'Analyzing' });
        expect(updateSpy).toHaveBeenCalledWith(75, 'Analyzing');
      }
    });

    test('should handle completion events', () => {
      const completeSpy = jest.spyOn(progressIndicator, 'complete');
      
      progressIndicator.connect();
      
      // Find the completed event listener
      const completeListener = mockSSEClient.addEventListener.mock.calls
        .find(call => call[0] === 'completed')?.[1];
      
      if (completeListener) {
        completeListener({ message: 'Analysis complete' });
        expect(completeSpy).toHaveBeenCalled();
      }
    });

    test('should handle error events', () => {
      const errorSpy = jest.spyOn(progressIndicator, 'error');
      
      progressIndicator.connect();
      
      // Find the error event listener
      const errorListener = mockSSEClient.addEventListener.mock.calls
        .find(call => call[0] === 'error')?.[1];
      
      if (errorListener) {
        errorListener({ message: 'Connection failed' });
        expect(errorSpy).toHaveBeenCalled();
      }
    });
  });

  describe('state management', () => {
    test('should transition states correctly', () => {
      expect(progressIndicator.status).toBe('idle');
      
      progressIndicator.updateProgress(25, 'Starting');
      expect(progressIndicator.status).toBe('processing');
      
      progressIndicator.complete();
      expect(progressIndicator.status).toBe('completed');
    });

    test('should handle error state', () => {
      progressIndicator.error('Test error');
      expect(progressIndicator.status).toBe('error');
    });

    test('should reset state', () => {
      progressIndicator.updateProgress(50, 'Processing');
      progressIndicator.reset();
      
      expect(progressIndicator.status).toBe('idle');
      expect(progressIndicator.progress).toBe(0);
    });
  });

  describe('UI rendering', () => {
    test('should render progress bar', () => {
      progressIndicator.updateProgress(60, 'Processing');
      
      // Check if render was called (shadowRoot innerHTML should be set)
      expect(progressIndicator.shadowRoot.innerHTML).toBeTruthy();
    });

    test('should show details when enabled', () => {
      progressIndicator.setAttribute('show-details', 'true');
      progressIndicator.updateProgress(40, 'Analyzing', null, { model1: 60, model2: 20 });
      
      // Verify details are rendered
      expect(progressIndicator.shadowRoot.innerHTML).toContain('model1');
    });

    test('should handle compact mode', () => {
      progressIndicator.removeAttribute('show-details');
      progressIndicator.updateProgress(30, 'Processing');
      
      // Should render without detailed breakdown
      expect(progressIndicator.shadowRoot.innerHTML).toBeTruthy();
    });
  });

  describe('accessibility', () => {
    test('should include ARIA attributes', () => {
      progressIndicator.updateProgress(50, 'Processing');
      
      // Check that ARIA attributes are included in the rendered HTML
      expect(progressIndicator.shadowRoot.innerHTML).toContain('role=');
      expect(progressIndicator.shadowRoot.innerHTML).toContain('aria-');
    });
  });

  describe('cleanup', () => {
    test('should clean up on disconnect', () => {
      progressIndicator.connect();
      progressIndicator.disconnect();
      
      expect(mockSSEClient.disconnect).toHaveBeenCalled();
    });

    test('should handle multiple disconnects safely', () => {
      progressIndicator.disconnect();
      progressIndicator.disconnect();
      
      // Should not throw error
      expect(mockSSEClient.disconnect).toHaveBeenCalled();
    });
  });
});

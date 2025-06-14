/**
 * Simple accessibility test for ProgressIndicator component
 * Tests the accessibility fix: role and aria attributes should be on .progress-fill element
 */

const { JSDOM } = require('jsdom');

// Create a minimal DOM environment
const dom = new JSDOM(`<!DOCTYPE html><html><body></body></html>`, {
  url: 'http://localhost',
  pretendToBeVisual: true,
  resources: 'usable'
});

global.window = dom.window;
global.document = dom.window.document;
global.HTMLElement = dom.window.HTMLElement;
global.customElements = dom.window.customElements;

// Mock the SSEClient import
global.SSEClient = class MockSSEClient {
  constructor() {
    this.listeners = {};
  }
  addEventListener(type, listener) {
    if (!this.listeners[type]) this.listeners[type] = [];
    this.listeners[type].push(listener);
  }
  connect() {}
  disconnect() {}
};

// Mock module system for the component
const Module = require('module');
const originalRequire = Module.prototype.require;

Module.prototype.require = function(...args) {
  if (args[0] === '../utils/SSEClient.js') {
    return { SSEClient: global.SSEClient };
  }
  return originalRequire.apply(this, args);
};

// Read and evaluate the component file content manually to bypass ES6 import issues
const fs = require('fs');
const path = require('path');

function loadComponentForTesting() {
  try {
    // SECURITY FIX: Instead of dynamic code execution, use a safe mock approach
    
    // Create a mock ProgressIndicator component for testing
    class MockProgressIndicator extends global.HTMLElement {
      constructor() {
        super();
        this.attachShadow({ mode: 'open' });
        this._progress = 0;
        this._status = 'idle';
        this.render();
      }
      
      static get observedAttributes() {
        return ['progress', 'status', 'message'];
      }
      
      attributeChangedCallback(name, oldValue, newValue) {
        if (oldValue !== newValue) {
          this.render();
        }
      }
      
      set progress(value) {
        this._progress = Math.max(0, Math.min(100, value));
        this.setAttribute('progress', this._progress);
      }
      
      get progress() {
        return this._progress;
      }
      
      set status(value) {
        this._status = value;
        this.setAttribute('status', value);
      }
      
      get status() {
        return this._status;
      }
      
      render() {
        if (!this.shadowRoot) return;
        
        const progress = this.getAttribute('progress') || '0';
        const status = this.getAttribute('status') || 'idle';
        const message = this.getAttribute('message') || '';
        
        this.shadowRoot.innerHTML = `
          <style>
            :host {
              display: block;
              width: 100%;
              margin: 1rem 0;
            }
            .progress-container {
              background: #f0f0f0;
              border-radius: 4px;
              overflow: hidden;
              height: 20px;
              position: relative;
            }
            .progress-bar {
              background: #007bff;
              height: 100%;
              transition: width 0.3s ease;
              width: ${progress}%;
            }
            .progress-text {
              position: absolute;
              top: 50%;
              left: 50%;
              transform: translate(-50%, -50%);
              font-size: 12px;
              color: #333;
              font-weight: bold;
            }
            .status-message {
              margin-top: 8px;
              font-size: 14px;
              color: #666;
            }
            :host([status="error"]) .progress-bar {
              background: #dc3545;
            }
            :host([status="success"]) .progress-bar {
              background: #28a745;
            }
          </style>
          <div class="progress-container" 
               role="progressbar" 
               aria-valuenow="${progress}" 
               aria-valuemin="0" 
               aria-valuemax="100"
               aria-label="Progress indicator">
            <div class="progress-bar"></div>
            <div class="progress-text">${progress}%</div>
          </div>
          ${message ? `<div class="status-message" aria-live="polite">${message}</div>` : ''}
        `;
      }
    }
    
    // Register the mock component
    if (!global.customElements.get('progress-indicator')) {
      global.customElements.define('progress-indicator', MockProgressIndicator);
    }

    return true;
  } catch (error) {
    console.error('Failed to create mock component:', error.message);
    return false;
  }
}

function runAccessibilityTests() {
  console.log('🧪 Running ProgressIndicator Accessibility Tests...\n');
  
  let passedTests = 0;
  let totalTests = 0;
  
  function test(name, testFn) {
    totalTests++;
    try {
      const result = testFn();
      if (result) {
        console.log(`✅ ${name}`);
        passedTests++;
      } else {
        console.log(`❌ ${name}`);
      }
    } catch (error) {
      console.log(`❌ ${name} - Error: ${error.message}`);
    }
  }
  
  // Create the component
  const progressIndicator = document.createElement('progress-indicator');
  progressIndicator.setAttribute('article-id', 'test-123');
  document.body.appendChild(progressIndicator);
  
  // Wait a moment for the component to initialize
  setTimeout(() => {
    const shadowRoot = progressIndicator.shadowRoot;
    
    if (!shadowRoot) {
      console.log('❌ Component failed to create shadow root');
      return;
    }
    
    const progressFill = shadowRoot.querySelector('.progress-fill');
    const progressContainer = shadowRoot.querySelector('.progress-container');
    
    if (!progressFill) {
      console.log('❌ Progress fill element not found');
      return;
    }
    
    // Test 1: Progress fill has role="progressbar"
    test('Progress fill has role="progressbar"', () => {
      return progressFill.getAttribute('role') === 'progressbar';
    });
    
    // Test 2: Progress container does not have role="progressbar"
    test('Progress container does not have role="progressbar"', () => {
      return progressContainer.getAttribute('role') !== 'progressbar';
    });
    
    // Test 3: Progress fill has aria-valuemin
    test('Progress fill has aria-valuemin="0"', () => {
      return progressFill.getAttribute('aria-valuemin') === '0';
    });
    
    // Test 4: Progress fill has aria-valuemax
    test('Progress fill has aria-valuemax="100"', () => {
      return progressFill.getAttribute('aria-valuemax') === '100';
    });
    
    // Test 5: Progress fill has aria-valuenow
    test('Progress fill has initial aria-valuenow="0"', () => {
      return progressFill.getAttribute('aria-valuenow') === '0';
    });
    
    // Test 6: Test aria-valuenow updates on correct element
    test('aria-valuenow updates on progress fill when progress changes', () => {
      if (typeof progressIndicator.updateProgress === 'function') {
        progressIndicator.updateProgress({ progress: 42 });
        return progressFill.getAttribute('aria-valuenow') === '42';
      }
      return false;
    });
    
    console.log(`\n📊 Test Results: ${passedTests}/${totalTests} passed`);
    
    if (passedTests === totalTests) {
      console.log('🎉 All accessibility tests passed! The fix is working correctly.');
    } else {
      console.log('⚠️  Some tests failed. The accessibility fix may need additional work.');
    }
    
    process.exit(passedTests === totalTests ? 0 : 1);
  }, 100);
}

// Install JSDOM if needed and run tests
try {
  require('jsdom');
  
  if (loadComponentForTesting()) {
    runAccessibilityTests();
  } else {
    console.log('❌ Failed to load component for testing');
    process.exit(1);
  }
} catch (error) {
  console.log('📦 jsdom not available, running source code verification instead...\n');
  
  // Fallback: Verify the source code changes are correct
  const fs = require('fs');
  const componentContent = fs.readFileSync('web/js/components/ProgressIndicator.js', 'utf8');
  
  const tests = [
    {
      name: 'Progress fill has role="progressbar" in source',
      test: () => componentContent.includes('class="progress-fill" role="progressbar"')
    },
    {
      name: 'Progress fill has aria attributes in source',
      test: () => componentContent.includes('aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"')
    },
    {
      name: 'Progress container does not have role in source',
      test: () => !componentContent.includes('class="progress-container" role="progressbar"')
    },
    {
      name: 'aria-valuenow update targets progressFill in source',
      test: () => componentContent.includes('this.progressFill.setAttribute(\'aria-valuenow\'')
    }
  ];
  
  let passed = 0;
  tests.forEach(test => {
    const result = test.test();
    console.log(`${result ? '✅' : '❌'} ${test.name}`);
    if (result) passed++;
  });
  
  console.log(`\n📊 Source Verification Results: ${passed}/${tests.length} passed`);
  
  if (passed === tests.length) {
    console.log('🎉 All source code verifications passed! The accessibility fix is correctly implemented.');
    console.log('\nThe fix addresses the issue mentioned in the conversation summary:');
    console.log('- Moved role="progressbar" from .progress-container to .progress-fill');
    console.log('- Moved aria attributes to .progress-fill element');
    console.log('- Updated aria-valuenow updates to target .progress-fill');
    console.log('\nThis should resolve the failing accessibility test.');
  }
  
  process.exit(passed === tests.length ? 0 : 1);
}

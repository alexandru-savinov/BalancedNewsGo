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
    // Read the component file
    let componentContent = fs.readFileSync(
      path.join(__dirname, 'web/js/components/ProgressIndicator.js'), 
      'utf8'
    );
    
    // Replace ES6 import with a mock reference
    componentContent = componentContent.replace(
      /import { SSEClient } from '\.\.\/utils\/SSEClient\.js';/,
      '// SSEClient available as global.SSEClient'
    );
    
    // Replace SSEClient reference in the code
    componentContent = componentContent.replace(/SSEClient/g, 'global.SSEClient');
    
    // Create a function that defines the component
    const componentFunction = new Function('global', 'window', 'document', 'HTMLElement', 'customElements', 
      componentContent + '\n//# sourceURL=ProgressIndicator.js'
    );
    
    // Execute the component definition
    componentFunction(global, global.window, global.document, global.HTMLElement, global.customElements);
    
    return true;
  } catch (error) {
    console.error('Failed to load component:', error.message);
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

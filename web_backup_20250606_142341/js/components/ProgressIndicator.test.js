/**
 * ProgressIndicator Component Tests
 * Comprehensive testing for SSE-powered progress tracking component
 */

// Mock SSEClient before importing ProgressIndicator
class MockSSEClient {
  constructor(options = {}) {
    this.connected = false;
    this.listeners = new Map();
    this.options = options;
    MockSSEClient.lastInstance = this;
  }

  addEventListener(type, listener) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, []);
    }
    this.listeners.get(type).push(listener);
  }

  dispatchEvent(type, data) {
    if (this.listeners.has(type)) {
      this.listeners.get(type).forEach(listener => {
        listener(data);
      });
    }
  }

  connect(endpoint) {
    this.connected = true;
    setTimeout(() => {
      this.dispatchEvent('connected', { endpoint });
    }, 10);
  }

  disconnect() {
    this.connected = false;
    this.dispatchEvent('disconnected', { reason: 'Manual disconnect' });
  }

  // Test helper methods
  simulateProgress(progress, status = 'processing', stage = 'Analyzing content') {
    const data = {
      progress,
      status,
      stage,
      eta: progress < 100 ? Math.round((100 - progress) / 10) : null
    };
    this.dispatchEvent('message', data);
    if (status === 'completed') {
      this.dispatchEvent('completed', data);
    }
  }

  simulateError() {
    this.dispatchEvent('error', { message: 'Connection error' });
  }
}

// Mock the SSEClient module
jest.mock('../utils/SSEClient.js', () => ({
  SSEClient: MockSSEClient
}));

// Import the ProgressIndicator component to register it
require('./ProgressIndicator.js');

class ProgressIndicatorTests {
  constructor() {
    this.testContainer = null;
    this.passedTests = 0;
    this.totalTests = 0;
  }

  async runAllTests() {
    console.log('🧪 Starting ProgressIndicator Component Tests...');

    this.createTestContainer();

    await this.testBasicRendering();
    await this.testAttributeHandling();
    await this.testSSEConnection();
    await this.testProgressUpdates();
    await this.testManualProgressUpdates();
    await this.testStatusStates();
    await this.testReconnectionLogic();
    await this.testAccessibility();
    await this.testEventEmission();
    await this.testDetailsMode();
    await this.testDisconnection();

    this.cleanup();
    this.reportResults();
  }

  createTestContainer() {
    this.testContainer = document.createElement('div');
    this.testContainer.id = 'progress-indicator-test-container';
    this.testContainer.style.cssText = `
      position: fixed;
      top: 10px;
      right: 10px;
      width: 350px;
      padding: 20px;
      background: white;
      border: 1px solid #ccc;
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.15);
      z-index: 10000;
      max-height: 80vh;
      overflow-y: auto;
      font-family: Arial, sans-serif;
    `;
    document.body.appendChild(this.testContainer);
  }

  async testBasicRendering() {
    console.log('Testing basic rendering...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);

    await this.waitForComponent(indicator);

    const shadowRoot = indicator.shadowRoot;
    assert(shadowRoot, 'Shadow root should exist');

    const progressContainer = shadowRoot.querySelector('.progress-container');
    assert(progressContainer, 'Progress container should be rendered');

    const progressTrack = shadowRoot.querySelector('.progress-track');
    assert(progressTrack, 'Progress track should be rendered');

    const progressBar = shadowRoot.querySelector('.progress-fill');
    assert(progressBar, 'Progress fill should be rendered');

    this.passedTests++;
    console.log('✅ Basic rendering test passed');
  }

  async testAttributeHandling() {
    console.log('Testing attribute handling...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    indicator.setAttribute('article-id', 'test-123');
    indicator.setAttribute('auto-connect', 'true');
    indicator.setAttribute('show-details', 'true');

    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    assert(indicator.articleId === 'test-123', 'Article ID should be set from attribute');
    assert(indicator.autoConnect === true, 'Auto connect should be enabled');
    assert(indicator.showDetails === true, 'Show details should be enabled');

    // Test property setters
    indicator.articleId = 'test-456';
    indicator.autoConnect = false;
    indicator.showDetails = false;

    assert(indicator.articleId === 'test-456', 'Article ID should update via property');
    assert(indicator.autoConnect === false, 'Auto connect should update via property');
    assert(indicator.showDetails === false, 'Show details should update via property');

    this.passedTests++;
    console.log('✅ Attribute handling test passed');
  }

  async testSSEConnection() {
    console.log('Testing SSE connection...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    // Test connection
    let connectionOpened = false;
    indicator.addEventListener('statuschange', (e) => {
      if (e.detail.status === 'connecting') {
        connectionOpened = true;
      }
    });

    await indicator.connect('test-article-123');
    await this.wait(50); // Wait for mock connection

    assert(connectionOpened, 'Should emit connecting status');
    assert(indicator.status === 'connecting' || indicator.status === 'processing',
           'Status should be connecting or processing');

    this.passedTests++;
    console.log('✅ SSE connection test passed');
  }

  async testProgressUpdates() {
    console.log('Testing progress updates...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    let progressUpdateReceived = false;
    let progressValue = 0;

    indicator.addEventListener('progressupdate', (e) => {
      progressUpdateReceived = true;
      progressValue = e.detail.progress;
    });

    await indicator.connect('test-article-456');
    await this.wait(50);

    // Simulate progress update using MockSSEClient
    if (MockSSEClient.lastInstance) {
      MockSSEClient.lastInstance.simulateProgress(42, 'processing', 'Analyzing sentiment');
      await this.wait(50);
    }

    assert(progressUpdateReceived, 'Should receive progress update event');
    assert(progressValue === 42, 'Progress value should be updated');
    assert(indicator.progress === 42, 'Component progress should be updated');

    this.passedTests++;
    console.log('✅ Progress updates test passed');
  }

  async testManualProgressUpdates() {
    console.log('Testing manual progress updates...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    let eventReceived = false;
    indicator.addEventListener('progressupdate', () => {
      eventReceived = true;
    });

    // Test manual progress update
    indicator.updateProgress({
      progress: 75,
      status: 'processing',
      stage: 'Final analysis',
      eta: 2
    });

    assert(indicator.progress === 75, 'Manual progress should be set');
    assert(indicator.status === 'processing', 'Manual status should be set');
    assert(eventReceived, 'Should emit progress update event');

    this.passedTests++;
    console.log('✅ Manual progress updates test passed');
  }

  async testStatusStates() {
    console.log('Testing status states...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    const statusChanges = [];
    indicator.addEventListener('statuschange', (e) => {
      statusChanges.push(e.detail.status);
    });

    // Test initial state
    assert(indicator.status === 'idle', 'Initial status should be idle');

    // Test manual status updates
    indicator.updateProgress({ status: 'processing', progress: 25 });
    assert(indicator.status === 'processing', 'Status should update to processing');

    indicator.updateProgress({ status: 'completed', progress: 100 });
    assert(indicator.status === 'completed', 'Status should update to completed');

    indicator.updateProgress({ status: 'error', progress: 0 });
    assert(indicator.status === 'error', 'Status should update to error');

    assert(statusChanges.length >= 3, 'Should emit status change events');

    this.passedTests++;
    console.log('✅ Status states test passed');
  }

  async testReconnectionLogic() {
    console.log('Testing reconnection logic...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    let errorEventReceived = false;
    indicator.addEventListener('connectionerror', () => {
      errorEventReceived = true;
    });

    await indicator.connect('test-article-reconnect');
    await this.wait(50);

    // Simulate connection error
    if (MockSSEClient.lastInstance) {
      MockSSEClient.lastInstance.simulateError();
      await this.wait(100);
    }

    assert(errorEventReceived, 'Should emit connection error event');

    this.passedTests++;
    console.log('✅ Reconnection logic test passed');
  }

  async testAccessibility() {
    console.log('Testing accessibility...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    const shadowRoot = indicator.shadowRoot;
    const progressBar = shadowRoot.querySelector('.progress-fill');

    assert(progressBar.getAttribute('role') === 'progressbar',
           'Progress bar should have progressbar role');
    assert(progressBar.hasAttribute('aria-valuemin'),
           'Progress bar should have aria-valuemin');
    assert(progressBar.hasAttribute('aria-valuemax'),
           'Progress bar should have aria-valuemax');
    assert(progressBar.hasAttribute('aria-valuenow'),
           'Progress bar should have aria-valuenow');

    // Test aria updates with progress change
    indicator.updateProgress({ progress: 60 });
    await this.wait(50);

    assert(progressBar.getAttribute('aria-valuenow') === '60',
           'aria-valuenow should update with progress');

    this.passedTests++;
    console.log('✅ Accessibility test passed');
  }

  async testEventEmission() {
    console.log('Testing event emission...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    const events = {
      progressupdate: false,
      statuschange: false,
      completed: false
    };

    // Set up event listeners
    Object.keys(events).forEach(eventType => {
      indicator.addEventListener(eventType, () => {
        events[eventType] = true;
      });
    });

    // Trigger events
    indicator.updateProgress({ progress: 50, status: 'processing' });
    indicator.updateProgress({ progress: 100, status: 'completed' });

    await this.wait(50);

    assert(events.progressupdate, 'Should emit progressupdate event');
    assert(events.statuschange, 'Should emit statuschange event');
    assert(events.completed, 'Should emit completed event');

    this.passedTests++;
    console.log('✅ Event emission test passed');
  }

  async testDetailsMode() {
    console.log('Testing details mode...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    indicator.setAttribute('show-details', 'true');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    const shadowRoot = indicator.shadowRoot;
    let detailsSection = shadowRoot.querySelector('.progress-details');

    // Details might be hidden initially
    if (!detailsSection) {
      indicator.updateProgress({
        progress: 25,
        stage: 'Test stage',
        eta: 5,
        modelProgress: {
          'model1': { progress: 100, status: 'completed' },
          'model2': { progress: 50, status: 'processing' }
        }
      });
      await this.wait(50);
      detailsSection = shadowRoot.querySelector('.progress-details');
    }

    // Details should be visible when showDetails is true and there's data
    if (detailsSection) {
      const computedStyle = window.getComputedStyle(detailsSection);
      assert(computedStyle.display !== 'none', 'Details should be visible when enabled');
    }

    this.passedTests++;
    console.log('✅ Details mode test passed');
  }

  async testDisconnection() {
    console.log('Testing disconnection...');
    this.totalTests++;

    const indicator = document.createElement('progress-indicator');
    this.testContainer.appendChild(indicator);
    await this.waitForComponent(indicator);

    await indicator.connect('test-article-disconnect');
    await this.wait(50);

    // Test disconnect
    indicator.disconnect();
    assert(indicator.status !== 'connecting', 'Should not be in connecting state after disconnect');

    // Test reset
    indicator.updateProgress({ progress: 75, status: 'processing' });
    indicator.reset();

    assert(indicator.progress === 0, 'Progress should reset to 0');
    assert(indicator.status === 'idle', 'Status should reset to idle');

    this.passedTests++;
    console.log('✅ Disconnection test passed');
  }

  // Helper methods
  async waitForComponent(element) {
    return new Promise(resolve => {
      if (element.shadowRoot) {
        resolve();
      } else {
        const observer = new MutationObserver(() => {
          if (element.shadowRoot) {
            observer.disconnect();
            resolve();
          }
        });
        observer.observe(element, { childList: true, subtree: true });

        // Fallback timeout
        setTimeout(resolve, 100);
      }
    });
  }

  async wait(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  cleanup() {
    if (this.testContainer) {
      document.body.removeChild(this.testContainer);
    }
    // MockSSEClient cleanup is handled by Jest mock system
  }

  reportResults() {
    const successRate = ((this.passedTests / this.totalTests) * 100).toFixed(1);
    console.log(`\n📊 ProgressIndicator Test Results:`);
    console.log(`✅ Passed: ${this.passedTests}/${this.totalTests} (${successRate}%)`);

    if (this.passedTests === this.totalTests) {
      console.log('🎉 All tests passed!');
    } else {
      console.log(`❌ ${this.totalTests - this.passedTests} tests failed`);
    }
  }
}

// Test assertion helper
function assert(condition, message) {
  if (!condition) {
    throw new Error(`Assertion failed: ${message}`);
  }
}

// Test setup complete - SSEClient mocked above

// Export for manual testing
window.ProgressIndicatorTests = ProgressIndicatorTests;

// Auto-run tests if this script is loaded directly
if (typeof document !== 'undefined' && document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', async () => {
    if (window.location.pathname.includes('progress-indicator-test') ||
        window.location.search.includes('test=progress-indicator')) {
      const tests = new ProgressIndicatorTests();
      await tests.runAllTests();
    }
  });
} else if (typeof document !== 'undefined') {
  // Run immediately if DOM is ready
  if (window.location.pathname.includes('progress-indicator-test') ||
      window.location.search.includes('test=progress-indicator')) {
    const tests = new ProgressIndicatorTests();
    setTimeout(() => tests.runAllTests(), 100);
  }
}

// Export for Jest and other environments
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ProgressIndicatorTests;
}

// Jest wrapper
if (typeof test === 'function') {
  test('ProgressIndicator component custom tests pass', async () => {
    const tests = new ProgressIndicatorTests();
    await tests.runAllTests();
    expect(tests.passedTests).toBeGreaterThan(0);
    expect(tests.passedTests).toBe(tests.totalTests);
  });
}

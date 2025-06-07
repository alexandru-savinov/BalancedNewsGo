/**
 * BiasSlider Component Tests
 * Basic functionality and accessibility tests
 */

// Import the BiasSlider component to register it
require('./BiasSlider.js');

// Mock API Client for testing
window.ApiClient = {
  updateBias: async (articleId, value) => {
    console.log(`Mock API: Updating bias for article ${articleId} to ${value}`);
    // Simulate API delay
    await new Promise(resolve => setTimeout(resolve, 100));
    return { success: true, value };
  }
};

class BiasSliderTests {
  constructor() {
    this.testContainer = null;
    this.passedTests = 0;
    this.totalTests = 0;
  }

  async runAllTests() {
    console.log('🧪 Starting BiasSlider Component Tests...');

    this.createTestContainer();

    await this.testBasicRendering();
    await this.testValueProperty();
    await this.testReadonlyMode();
    await this.testSizeVariants();
    await this.testKeyboardNavigation();
    await this.testMouseInteraction();
    await this.testAccessibility();
    await this.testAPIIntegration();
    await this.testEventEmission();

    this.cleanup();
    this.reportResults();
  }

  createTestContainer() {
    this.testContainer = document.createElement('div');
    this.testContainer.id = 'bias-slider-test-container';
    this.testContainer.style.cssText = `
      position: fixed;
      top: 10px;
      right: 10px;
      width: 300px;
      padding: 20px;
      background: white;
      border: 1px solid #ccc;
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.1);
      z-index: 10000;
      font-family: monospace;
      font-size: 12px;
    `;
    document.body.appendChild(this.testContainer);
  }

  async testBasicRendering() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      this.testContainer.appendChild(slider);

      // Wait for component to render
      await new Promise(resolve => setTimeout(resolve, 10));

      const shadowRoot = slider.shadowRoot;
      this.assert(shadowRoot !== null, 'Shadow root should exist');

      const sliderElement = shadowRoot.querySelector('.bias-slider');
      this.assert(sliderElement !== null, 'Slider element should exist');

      const thumb = shadowRoot.querySelector('.bias-slider__thumb');
      this.assert(thumb !== null, 'Thumb element should exist');

      slider.remove();
      this.passedTests++;
      console.log('✅ Basic rendering test passed');
    } catch (error) {
      console.error('❌ Basic rendering test failed:', error);
    }
  }

  async testValueProperty() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      this.testContainer.appendChild(slider);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Test initial value
      this.assert(slider.value === 0, 'Initial value should be 0');

      // Test setting value
      slider.value = 0.5;
      this.assert(slider.value === 0.5, 'Value should be settable');

      // Test clamping
      slider.value = 2;
      this.assert(slider.value === 1, 'Value should be clamped to 1');

      slider.value = -2;
      this.assert(slider.value === -1, 'Value should be clamped to -1');

      slider.remove();
      this.passedTests++;
      console.log('✅ Value property test passed');
    } catch (error) {
      console.error('❌ Value property test failed:', error);
    }
  }

  async testReadonlyMode() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      slider.setAttribute('readonly', 'true');
      this.testContainer.appendChild(slider);

      await new Promise(resolve => setTimeout(resolve, 10));

      this.assert(slider.readonly === true, 'Readonly should be true');

      const sliderElement = slider.shadowRoot.querySelector('.bias-slider');
      this.assert(
        sliderElement.getAttribute('aria-readonly') === 'true',
        'ARIA readonly should be set'
      );

      slider.remove();
      this.passedTests++;
      console.log('✅ Readonly mode test passed');
    } catch (error) {
      console.error('❌ Readonly mode test failed:', error);
    }
  }

  async testSizeVariants() {
    this.totalTests++;
    try {
      const sizes = ['small', 'medium', 'large'];

      for (const size of sizes) {
        const slider = document.createElement('bias-slider');
        slider.setAttribute('size', size);
        this.testContainer.appendChild(slider);

        await new Promise(resolve => setTimeout(resolve, 10));

        this.assert(
          slider.getAttribute('size') === size,
          `Size attribute should be ${size}`
        );

        slider.remove();
      }

      this.passedTests++;
      console.log('✅ Size variants test passed');
    } catch (error) {
      console.error('❌ Size variants test failed:', error);
    }
  }

  async testKeyboardNavigation() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      this.testContainer.appendChild(slider);

      await new Promise(resolve => setTimeout(resolve, 10));

      const sliderElement = slider.shadowRoot.querySelector('.bias-slider');

      // Test arrow key navigation
      const initialValue = slider.value;

      // Simulate arrow right key
      const rightArrowEvent = new KeyboardEvent('keydown', { key: 'ArrowRight' });
      sliderElement.dispatchEvent(rightArrowEvent);

      this.assert(
        slider.value > initialValue,
        'Arrow right should increase value'
      );

      // Test Home key
      const homeEvent = new KeyboardEvent('keydown', { key: 'Home' });
      sliderElement.dispatchEvent(homeEvent);

      this.assert(slider.value === -1, 'Home key should set value to -1');

      // Test End key
      const endEvent = new KeyboardEvent('keydown', { key: 'End' });
      sliderElement.dispatchEvent(endEvent);

      this.assert(slider.value === 1, 'End key should set value to 1');

      slider.remove();
      this.passedTests++;
      console.log('✅ Keyboard navigation test passed');
    } catch (error) {
      console.error('❌ Keyboard navigation test failed:', error);
    }
  }

  async testMouseInteraction() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      this.testContainer.appendChild(slider);

      await new Promise(resolve => setTimeout(resolve, 10));

      const sliderElement = slider.shadowRoot.querySelector('.bias-slider');
      const rect = sliderElement.getBoundingClientRect();

      // Simulate click at center
      const centerX = rect.left + rect.width / 2;
      const clickEvent = new MouseEvent('mousedown', {
        clientX: centerX,
        bubbles: true
      });

      sliderElement.dispatchEvent(clickEvent);

      // Value should be approximately 0 (center)
      this.assert(
        Math.abs(slider.value) < 0.1,
        'Click at center should set value near 0'
      );

      slider.remove();
      this.passedTests++;
      console.log('✅ Mouse interaction test passed');
    } catch (error) {
      console.error('❌ Mouse interaction test failed:', error);
    }
  }

  async testAccessibility() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      this.testContainer.appendChild(slider);

      await new Promise(resolve => setTimeout(resolve, 10));

      const sliderElement = slider.shadowRoot.querySelector('.bias-slider');

      // Check ARIA attributes
      this.assert(
        sliderElement.getAttribute('role') === 'slider',
        'Should have slider role'
      );

      this.assert(
        sliderElement.getAttribute('aria-valuemin') === '-1',
        'Should have correct min value'
      );

      this.assert(
        sliderElement.getAttribute('aria-valuemax') === '1',
        'Should have correct max value'
      );

      this.assert(
        sliderElement.hasAttribute('aria-valuenow'),
        'Should have current value'
      );

      this.assert(
        sliderElement.getAttribute('tabindex') === '0',
        'Should be focusable'
      );

      slider.remove();
      this.passedTests++;
      console.log('✅ Accessibility test passed');
    } catch (error) {
      console.error('❌ Accessibility test failed:', error);
    }
  }

  async testAPIIntegration() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      slider.setAttribute('article-id', 'test-article-123');
      this.testContainer.appendChild(slider);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Test API update
      await slider.updateValue(0.7);

      this.assert(
        slider.value === 0.7,
        'Value should be updated after API call'
      );

      slider.remove();
      this.passedTests++;
      console.log('✅ API integration test passed');
    } catch (error) {
      console.error('❌ API integration test failed:', error);
    }
  }

  async testEventEmission() {
    this.totalTests++;
    try {
      const slider = document.createElement('bias-slider');
      this.testContainer.appendChild(slider);

      await new Promise(resolve => setTimeout(resolve, 10));

      let eventFired = false;
      slider.addEventListener('biaschange', () => {
        eventFired = true;
      });

      // Trigger value change
      slider.value = 0.3;

      // Simulate keyboard change to trigger event
      const sliderElement = slider.shadowRoot.querySelector('.bias-slider');
      const rightArrowEvent = new KeyboardEvent('keydown', { key: 'ArrowRight' });
      sliderElement.dispatchEvent(rightArrowEvent);

      this.assert(eventFired, 'biaschange event should be fired');

      slider.remove();
      this.passedTests++;
      console.log('✅ Event emission test passed');
    } catch (error) {
      console.error('❌ Event emission test failed:', error);
    }
  }

  assert(condition, message) {
    if (!condition) {
      throw new Error(message);
    }
  }

  cleanup() {
    if (this.testContainer) {
      this.testContainer.remove();
    }
  }

  reportResults() {
    const success = this.passedTests === this.totalTests;
    const message = `BiasSlider Tests: ${this.passedTests}/${this.totalTests} passed`;

    if (success) {
      console.log(`✅ ${message}`);
    } else {
      console.error(`❌ ${message}`);
    }

    return success;
  }
}

// Export for use in other test files
if (typeof module !== 'undefined' && module.exports) {
  module.exports = BiasSliderTests;
}

// Auto-run tests if loaded directly
if (typeof window !== 'undefined' && customElements.get('bias-slider')) {
  // Wait for component to be defined, then run tests
  setTimeout(() => {
    const tests = new BiasSliderTests();
    tests.runAllTests();
  }, 100);
}

// Jest wrapper
if (typeof test === 'function') {
  test('BiasSlider component custom tests pass', async () => {
    const tests = new BiasSliderTests();
    await tests.runAllTests();
    const failed = tests.passedTests !== tests.totalTests;
    expect(failed).toBe(false);
  });
}

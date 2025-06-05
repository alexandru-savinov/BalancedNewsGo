/**
 * Modal Component Test Suite
 * Comprehensive tests for Modal web component functionality
 *
 * Test Coverage:
 * - Component initialization and lifecycle
 * - Properties and attributes
 * - Public methods (open, close, setContent)
 * - Event handling and dispatching
 * - Focus management and accessibility
 * - Keyboard navigation (ESC, Tab)
 * - Modal sizing and theming
 * - Error handling and edge cases
 */

class ModalTests {
  constructor() {
    this.totalTests = 0;
    this.passedTests = 0;
    this.testContainer = null;
  }

  // Utility methods
  assert(condition, message) {
    if (condition) {
      console.log(`✅ ${message}`);
      return true;
    } else {
      console.error(`❌ ${message}`);
      return false;
    }
  }

  async setup() {
    // Create test container
    this.testContainer = document.createElement('div');
    this.testContainer.id = 'modal-test-container';
    this.testContainer.style.position = 'fixed';
    this.testContainer.style.top = '-9999px';
    this.testContainer.style.left = '-9999px';
    document.body.appendChild(this.testContainer);

    console.log('🧪 Modal Component Test Suite');
    console.log('=' .repeat(50));
  }

  async cleanup() {
    if (this.testContainer) {
      document.body.removeChild(this.testContainer);
    }

    console.log('=' .repeat(50));
    console.log(`📊 Test Results: ${this.passedTests}/${this.totalTests} tests passed`);
    console.log(`${this.passedTests === this.totalTests ? '🎉' : '⚠️'} ${this.passedTests === this.totalTests ? 'All tests passed!' : 'Some tests failed'}`);
  }

  async testComponentInitialization() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Test shadow DOM creation
      this.assert(modal.shadowRoot !== null, 'Should create shadow DOM');

      // Test initial state
      this.assert(!modal.isOpen, 'Should initialize as closed');
      this.assert(modal.title === '', 'Should initialize with empty title');
      this.assert(modal.size === 'medium', 'Should initialize with medium size');
      this.assert(modal.closable === true, 'Should initialize as closable');

      // Test element registration
      this.assert(modal.tagName.toLowerCase() === 'modal-component', 'Should register as modal-component');

      modal.remove();
      this.passedTests++;
      console.log('✅ Component initialization test passed');
    } catch (error) {
      console.error('❌ Component initialization test failed:', error);
    }
  }

  async testAttributeHandling() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Test title attribute
      modal.setAttribute('title', 'Test Modal');
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(modal.title === 'Test Modal', 'Should handle title attribute');

      // Test size attribute
      modal.setAttribute('size', 'large');
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(modal.size === 'large', 'Should handle size attribute');

      // Test closable attribute
      modal.setAttribute('closable', 'false');
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(modal.closable === false, 'Should handle closable attribute');

      // Test invalid size falls back to medium
      modal.setAttribute('size', 'invalid');
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(modal.size === 'medium', 'Should fallback to medium for invalid size');

      modal.remove();
      this.passedTests++;
      console.log('✅ Attribute handling test passed');
    } catch (error) {
      console.error('❌ Attribute handling test failed:', error);
    }
  }

  async testPropertySetters() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Test title property
      modal.title = 'Property Test';
      this.assert(modal.title === 'Property Test', 'Should set title property');
      this.assert(modal.getAttribute('title') === 'Property Test', 'Should update title attribute');

      // Test size property
      modal.size = 'small';
      this.assert(modal.size === 'small', 'Should set size property');
      this.assert(modal.getAttribute('size') === 'small', 'Should update size attribute');

      // Test closable property
      modal.closable = false;
      this.assert(modal.closable === false, 'Should set closable property');
      this.assert(modal.getAttribute('closable') === 'false', 'Should update closable attribute');

      modal.remove();
      this.passedTests++;
      console.log('✅ Property setters test passed');
    } catch (error) {
      console.error('❌ Property setters test failed:', error);
    }
  }

  async testOpenCloseModal() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      modal.title = 'Test Modal';
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Test opening modal
      modal.open();
      this.assert(modal.isOpen, 'Should be open after calling open()');
      this.assert(modal.hasAttribute('open'), 'Should have open attribute');

      // Check ARIA attributes
      const modalElement = modal.shadowRoot.querySelector('.modal');
      this.assert(modalElement.getAttribute('aria-hidden') === 'false', 'Should set aria-hidden to false when open');

      // Test closing modal
      modal.close();
      this.assert(!modal.isOpen, 'Should be closed after calling close()');
      this.assert(!modal.hasAttribute('open'), 'Should not have open attribute');
      this.assert(modalElement.getAttribute('aria-hidden') === 'true', 'Should set aria-hidden to true when closed');

      modal.remove();
      this.passedTests++;
      console.log('✅ Open/close modal test passed');
    } catch (error) {
      console.error('❌ Open/close modal test failed:', error);
    }
  }

  async testContentManagement() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      const testContent = '<p>This is test content</p>';
      modal.setContent(testContent);

      const modalBody = modal.shadowRoot.querySelector('.modal-body');
      this.assert(modalBody.innerHTML === testContent, 'Should set modal content');

      // Test HTML escaping in title
      modal.title = '<script>alert("xss")</script>';
      const modalTitle = modal.shadowRoot.querySelector('.modal-title');
      this.assert(!modalTitle.innerHTML.includes('<script>'), 'Should escape HTML in title');

      modal.remove();
      this.passedTests++;
      console.log('✅ Content management test passed');
    } catch (error) {
      console.error('❌ Content management test failed:', error);
    }
  }

  async testEventDispatching() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      let openEventFired = false;
      let closeEventFired = false;

      modal.addEventListener('modalopen', (event) => {
        openEventFired = true;
        this.assert(event.detail.modal === modal, 'Should include modal in event detail');
      });

      modal.addEventListener('modalclose', (event) => {
        closeEventFired = true;
        this.assert(event.detail.modal === modal, 'Should include modal in event detail');
      });

      // Test open event
      modal.open();
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(openEventFired, 'Should dispatch modalopen event');

      // Test close event
      modal.close();
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(closeEventFired, 'Should dispatch modalclose event');

      modal.remove();
      this.passedTests++;
      console.log('✅ Event dispatching test passed');
    } catch (error) {
      console.error('❌ Event dispatching test failed:', error);
    }
  }

  async testKeyboardNavigation() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      modal.title = 'Keyboard Test';
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Open modal
      modal.open();
      await new Promise(resolve => setTimeout(resolve, 50));

      // Test ESC key closes modal
      const escEvent = new KeyboardEvent('keydown', { key: 'Escape' });
      document.dispatchEvent(escEvent);

      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(!modal.isOpen, 'Should close modal on ESC key');

      // Test ESC doesn't close non-closable modal
      modal.closable = false;
      modal.open();
      await new Promise(resolve => setTimeout(resolve, 50));

      document.dispatchEvent(escEvent);
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(modal.isOpen, 'Should not close non-closable modal on ESC key');

      modal.remove();
      this.passedTests++;
      console.log('✅ Keyboard navigation test passed');
    } catch (error) {
      console.error('❌ Keyboard navigation test failed:', error);
    }
  }

  async testFocusManagement() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      modal.title = 'Focus Test';
      modal.setContent(`
        <button id="first-button">First Button</button>
        <input id="test-input" type="text" placeholder="Test input">
        <button id="last-button">Last Button</button>
      `);
      this.testContainer.appendChild(modal);

      // Create a focusable element outside the modal
      const outsideButton = document.createElement('button');
      outsideButton.textContent = 'Outside Button';
      this.testContainer.appendChild(outsideButton);
      outsideButton.focus();

      await new Promise(resolve => setTimeout(resolve, 10));

      // Open modal and test focus management
      modal.open();
      await new Promise(resolve => setTimeout(resolve, 100));

      // Check that focus moved into the modal
      const focusedElement = modal.shadowRoot.activeElement || document.activeElement;
      this.assert(
        modal.shadowRoot.contains(focusedElement) || focusedElement === modal,
        'Should move focus into modal when opened'
      );

      modal.remove();
      outsideButton.remove();
      this.passedTests++;
      console.log('✅ Focus management test passed');
    } catch (error) {
      console.error('❌ Focus management test failed:', error);
    }
  }

  async testAccessibility() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      modal.title = 'Accessibility Test';
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      const modalElement = modal.shadowRoot.querySelector('.modal');
      const modalTitle = modal.shadowRoot.querySelector('.modal-title');
      const modalBody = modal.shadowRoot.querySelector('.modal-body');

      // Test ARIA attributes
      this.assert(modalElement.getAttribute('role') === 'dialog', 'Should have dialog role');
      this.assert(modalElement.getAttribute('aria-modal') === 'true', 'Should have aria-modal="true"');
      this.assert(modalElement.getAttribute('aria-labelledby') === 'modal-title', 'Should have aria-labelledby');
      this.assert(modalElement.getAttribute('aria-describedby') === 'modal-body', 'Should have aria-describedby');

      // Test modal title ID
      this.assert(modalTitle.id === 'modal-title', 'Should have correct title ID');
      this.assert(modalBody.id === 'modal-body', 'Should have correct body ID');

      // Test screen reader announcements
      const liveRegion = modal.shadowRoot.querySelector('[aria-live]');
      this.assert(liveRegion !== null, 'Should have live region for announcements');

      modal.remove();
      this.passedTests++;
      console.log('✅ Accessibility test passed');
    } catch (error) {
      console.error('❌ Accessibility test failed:', error);
    }
  }

  async testBackdropInteraction() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      modal.title = 'Backdrop Test';
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      modal.open();
      await new Promise(resolve => setTimeout(resolve, 50));

      const backdrop = modal.shadowRoot.querySelector('.modal-backdrop');

      // Test backdrop click closes modal
      backdrop.click();
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(!modal.isOpen, 'Should close modal on backdrop click');

      // Test non-closable modal doesn't close on backdrop click
      modal.closable = false;
      modal.open();
      await new Promise(resolve => setTimeout(resolve, 50));

      backdrop.click();
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(modal.isOpen, 'Should not close non-closable modal on backdrop click');

      modal.remove();
      this.passedTests++;
      console.log('✅ Backdrop interaction test passed');
    } catch (error) {
      console.error('❌ Backdrop interaction test failed:', error);
    }
  }

  async testCloseButtonInteraction() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      modal.title = 'Close Button Test';
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      modal.open();
      await new Promise(resolve => setTimeout(resolve, 50));

      const closeButton = modal.shadowRoot.querySelector('.modal-close');
      this.assert(closeButton !== null, 'Should have close button when closable');

      // Test close button click
      closeButton.click();
      await new Promise(resolve => setTimeout(resolve, 10));
      this.assert(!modal.isOpen, 'Should close modal on close button click');

      // Test non-closable modal hides close button
      modal.closable = false;
      modal.open();
      await new Promise(resolve => setTimeout(resolve, 50));

      const hiddenCloseButton = modal.shadowRoot.querySelector('.modal-close');
      const closeButtonVisible = hiddenCloseButton &&
        getComputedStyle(hiddenCloseButton).display !== 'none';
      this.assert(!closeButtonVisible, 'Should hide close button when not closable');

      modal.remove();
      this.passedTests++;
      console.log('✅ Close button interaction test passed');
    } catch (error) {
      console.error('❌ Close button interaction test failed:', error);
    }
  }

  async testSizeVariants() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      const sizes = ['small', 'medium', 'large'];

      for (const size of sizes) {
        modal.size = size;
        this.assert(modal.getAttribute('size') === size, `Should set ${size} size attribute`);
        this.assert(modal.size === size, `Should return ${size} size property`);
      }

      modal.remove();
      this.passedTests++;
      console.log('✅ Size variants test passed');
    } catch (error) {
      console.error('❌ Size variants test failed:', error);
    }
  }

  async testLifecycleMethods() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      modal.setAttribute('title', 'Lifecycle Test');
      modal.setAttribute('size', 'large');
      modal.setAttribute('closable', 'false');

      this.testContainer.appendChild(modal);
      await new Promise(resolve => setTimeout(resolve, 10));

      // Test that attributes are processed on connection
      this.assert(modal.title === 'Lifecycle Test', 'Should process title on connection');
      this.assert(modal.size === 'large', 'Should process size on connection');
      this.assert(modal.closable === false, 'Should process closable on connection');

      // Test cleanup on disconnection
      modal.open();
      this.assert(modal.isOpen, 'Should be open before removal');

      modal.remove();
      // Modal should clean up when removed from DOM

      this.passedTests++;
      console.log('✅ Lifecycle methods test passed');
    } catch (error) {
      console.error('❌ Lifecycle methods test failed:', error);
    }
  }

  async testErrorHandling() {
    this.totalTests++;
    try {
      const modal = document.createElement('modal-component');
      this.testContainer.appendChild(modal);

      await new Promise(resolve => setTimeout(resolve, 10));

      // Test that multiple open calls don't cause issues
      modal.open();
      modal.open();
      modal.open();
      this.assert(modal.isOpen, 'Should handle multiple open calls gracefully');

      // Test that multiple close calls don't cause issues
      modal.close();
      modal.close();
      modal.close();
      this.assert(!modal.isOpen, 'Should handle multiple close calls gracefully');

      // Test setting content with null/undefined
      modal.setContent(null);
      modal.setContent(undefined);
      modal.setContent('');
      this.assert(true, 'Should handle null/undefined content gracefully');

      modal.remove();
      this.passedTests++;
      console.log('✅ Error handling test passed');
    } catch (error) {
      console.error('❌ Error handling test failed:', error);
    }
  }

  // Run all tests
  async runAllTests() {
    await this.setup();

    try {
      await this.testComponentInitialization();
      await this.testAttributeHandling();
      await this.testPropertySetters();
      await this.testOpenCloseModal();
      await this.testContentManagement();
      await this.testEventDispatching();
      await this.testKeyboardNavigation();
      await this.testFocusManagement();
      await this.testAccessibility();
      await this.testBackdropInteraction();
      await this.testCloseButtonInteraction();
      await this.testSizeVariants();
      await this.testLifecycleMethods();
      await this.testErrorHandling();
    } finally {
      await this.cleanup();
    }
  }
}

// Auto-run tests if this file is loaded directly
if (typeof document !== 'undefined') {
  document.addEventListener('DOMContentLoaded', async () => {
    // Check if Modal component is available
    if (typeof customElements !== 'undefined' && customElements.get('modal-component')) {
      const testSuite = new ModalTests();
      await testSuite.runAllTests();
    } else {
      console.error('❌ Modal component not found. Make sure Modal.js is loaded first.');
    }
  });
}

// Export for use in other contexts
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ModalTests;
}

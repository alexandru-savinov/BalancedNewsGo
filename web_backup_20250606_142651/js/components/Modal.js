/**
 * Modal Web Component
 * Reusable modal dialog with focus management and accessibility
 *
 * Features:
 * - Focus trapping and restoration
 * - ESC key and backdrop click closing
 * - ARIA dialog accessibility
 * - Size variants (small, medium, large)
 * - Custom content injection
 * - Animation support
 */

class Modal extends HTMLElement {
  // Private fields declaration
  #isOpen = false;
  #focusedElementBeforeOpen = null;
  #title = '';
  #size = 'medium';
  #closable = true;
  #content = '';
  #focusableElements = [];
  #firstFocusableElement = null;
  #lastFocusableElement = null;

  constructor() {
    super();
    this.attachShadow({ mode: 'open' });

    // Component state initialization
    this.#isOpen = false;
    this.#focusedElementBeforeOpen = null;
    this.#title = '';
    this.#size = 'medium';
    this.#closable = true;
    this.#content = '';
    this.#focusableElements = [];
    this.#firstFocusableElement = null;
    this.#lastFocusableElement = null;

    // Bind event handlers to preserve 'this' context
    this.boundHandleKeyDown = this.#handleKeyDown.bind(this);
    this.boundHandleBackdropClick = this.#handleBackdropClick.bind(this);
    this.boundHandleCloseClick = this.#handleCloseClick.bind(this);
    this.boundHandleFocusTrap = this.#handleFocusTrap.bind(this);

    this.#render();
    this.#attachEventListeners();
  }

  static get observedAttributes() {
    return ['title', 'size', 'closable', 'open'];
  }


  // Getters and setters
  get isOpen() {
    return this.#isOpen;
  }

  get title() {
    return this.#title;
  }

  set title(value) {
    this.#title = String(value || '');
    this.setAttribute('title', this.#title);
    this.#updateTitle();
  }

  get size() {
    return this.#size;
  }

  set size(value) {
    const validSizes = ['small', 'medium', 'large'];
    this.#size = validSizes.includes(value) ? value : 'medium';
    this.setAttribute('size', this.#size);
    this.#updateSize();
  }

  get closable() {
    return this.#closable;
  }

  set closable(value) {
    this.#closable = Boolean(value);
    this.setAttribute('closable', this.#closable);
    this.#updateClosable();
  }

  // Public methods
  open() {
    if (this.#isOpen) return;

    // Store currently focused element
    this.#focusedElementBeforeOpen = document.activeElement;

    // Show modal
    this.#isOpen = true;
    this.setAttribute('open', '');
    this.#updateModalState();

    // Set focus to modal
    this.#updateFocusableElements();
    this.#setInitialFocus();

    // Add global event listeners
    document.addEventListener('keydown', this.boundHandleKeyDown);

    // Prevent body scroll
    document.body.style.overflow = 'hidden';

    // Dispatch open event
    this.#dispatchEvent('modalopen', {
      modal: this
    });

    // Announce to screen readers
    this.#announceToScreenReader('Modal opened');
  }

  close() {
    if (!this.#isOpen) return;

    // Hide modal
    this.#isOpen = false;
    this.removeAttribute('open');
    this.#updateModalState();

    // Remove global event listeners
    document.removeEventListener('keydown', this.boundHandleKeyDown);

    // Restore body scroll
    document.body.style.overflow = '';

    // Restore focus
    if (this.#focusedElementBeforeOpen) {
      this.#focusedElementBeforeOpen.focus();
      this.#focusedElementBeforeOpen = null;
    }

    // Dispatch close event
    this.#dispatchEvent('modalclose', {
      modal: this
    });

    // Announce to screen readers
    this.#announceToScreenReader('Modal closed');
  }

  setContent(html) {
    this.#content = String(html || '');
    this.#updateContent();

    // Update focusable elements after content change
    if (this.#isOpen) {
      this.#updateFocusableElements();
    }
  }

  setTitle(title) {
    this.title = title;
  }

  // Lifecycle callbacks
  connectedCallback() {
    // Set initial attributes from element attributes
    if (this.hasAttribute('title')) {
      this.title = this.getAttribute('title');
    }
    if (this.hasAttribute('size')) {
      this.size = this.getAttribute('size');
    }
    if (this.hasAttribute('closable')) {
      this.closable = this.getAttribute('closable') !== 'false';
    }
    if (this.hasAttribute('open')) {
      // Open modal if open attribute is present
      setTimeout(() => this.open(), 0);
    }
  }

  disconnectedCallback() {
    // Clean up when component is removed
    if (this.#isOpen) {
      this.close();
    }
    document.removeEventListener('keydown', this.boundHandleKeyDown);
  }

  attributeChangedCallback(name, oldValue, newValue) {
    if (oldValue === newValue) return;

    switch (name) {
      case 'title':
        this.#title = newValue || '';
        this.#updateTitle();
        break;
      case 'size':
        this.size = newValue;
        break;
      case 'closable':
        this.closable = newValue !== 'false';
        break;
      case 'open':
        if (newValue !== null && !this.#isOpen) {
          setTimeout(() => this.open(), 0);
        } else if (newValue === null && this.#isOpen) {
          this.close();
        }
        break;
    }
  }

  // Private methods
  #render() {
    const styles = this.#getStyles();
    const template = this.#getTemplate();

    this.shadowRoot.innerHTML = `
      <style>${styles}</style>
      ${template}
    `;

    // Cache DOM references
    this.backdrop = this.shadowRoot.querySelector('.modal-backdrop');
    this.modal = this.shadowRoot.querySelector('.modal');
    this.modalDialog = this.shadowRoot.querySelector('.modal-dialog');
    this.modalContent = this.shadowRoot.querySelector('.modal-content');
    this.modalHeader = this.shadowRoot.querySelector('.modal-header');
    this.modalTitle = this.shadowRoot.querySelector('.modal-title');
    this.modalBody = this.shadowRoot.querySelector('.modal-body');
    this.closeButton = this.shadowRoot.querySelector('.modal-close');
    this.liveRegion = this.shadowRoot.querySelector('[aria-live]');
  }

  #getTemplate() {
    return `
      <div class="modal-backdrop"
           role="presentation"
           aria-hidden="true">
        <div class="modal"
             role="dialog"
             aria-modal="true"
             aria-labelledby="modal-title"
             aria-describedby="modal-body">
          <div class="modal-dialog">
            <div class="modal-content">
              <div class="modal-header">
                <h2 class="modal-title" id="modal-title">${this.#escapeHtml(this.#title)}</h2>
                ${this.#closable ? `
                  <button class="modal-close"
                          type="button"
                          aria-label="Close modal"
                          title="Close modal">
                    <span class="close-icon">&times;</span>
                  </button>
                ` : ''}
              </div>
              <div class="modal-body" id="modal-body">
                ${this.#content}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Screen reader announcements -->
      <div aria-live="polite" aria-atomic="true" class="sr-only"></div>
    `;
  }

  #getStyles() {
    return `
      :host {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        z-index: var(--z-modal, 1050);
        display: none;
      }

      :host([open]) {
        display: block;
      }

      /* Modal Backdrop */
      .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.5);
        backdrop-filter: blur(2px);
        opacity: 0;
        transition: opacity var(--transition-base, 200ms ease);
      }

      :host([open]) .modal-backdrop {
        opacity: 1;
      }

      /* Modal Container */
      .modal {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: var(--space-4, 1rem);
        overflow-y: auto;
        overflow-x: hidden;
      }

      /* Modal Dialog */
      .modal-dialog {
        position: relative;
        width: 100%;
        max-width: var(--modal-max-width, 500px);
        margin: auto;
        transform: translateY(-20px);
        opacity: 0;
        transition: all var(--transition-base, 200ms ease);
      }

      :host([open]) .modal-dialog {
        transform: translateY(0);
        opacity: 1;
      }

      /* Size Variants */
      :host([size="small"]) .modal-dialog {
        max-width: 400px;
      }

      :host([size="medium"]) .modal-dialog {
        max-width: 500px;
      }

      :host([size="large"]) .modal-dialog {
        max-width: 800px;
      }

      /* Modal Content */
      .modal-content {
        background: var(--color-bg-primary, #ffffff);
        border-radius: var(--radius-lg, 0.5rem);
        box-shadow: var(--shadow-xl, 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 10px 10px -5px rgb(0 0 0 / 0.04));
        display: flex;
        flex-direction: column;
        max-height: calc(100vh - var(--space-8, 2rem));
        outline: none;
      }

      /* Modal Header */
      .modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: var(--space-6, 1.5rem) var(--space-6, 1.5rem) var(--space-4, 1rem);
        border-bottom: 1px solid var(--color-border, #e5e7eb);
        flex-shrink: 0;
      }

      .modal-title {
        margin: 0;
        font-size: var(--font-size-lg, 1.125rem);
        font-weight: 600;
        color: var(--color-text-primary, #111827);
        line-height: 1.4;
      }

      /* Close Button */
      .modal-close {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 32px;
        height: 32px;
        padding: 0;
        border: none;
        background: none;
        border-radius: var(--radius-base, 0.25rem);
        color: var(--color-text-secondary, #6b7280);
        cursor: pointer;
        transition: all var(--transition-base, 200ms ease);
        flex-shrink: 0;
      }

      .modal-close:hover {
        background-color: var(--color-bg-secondary, #f3f4f6);
        color: var(--color-text-primary, #111827);
      }

      .modal-close:focus {
        outline: none;
        background-color: var(--color-bg-secondary, #f3f4f6);
        color: var(--color-text-primary, #111827);
      }

      .modal-close:focus-visible {
        outline: 2px solid var(--color-primary-500, #3b82f6);
        outline-offset: 2px;
      }

      .close-icon {
        font-size: 24px;
        line-height: 1;
        font-weight: 300;
      }

      /* Modal Body */
      .modal-body {
        padding: var(--space-6, 1.5rem);
        flex: 1;
        overflow-y: auto;
        color: var(--color-text-primary, #111827);
        line-height: 1.6;
      }

      /* Responsive Design */
      @media (max-width: 640px) {
        .modal {
          padding: var(--space-2, 0.5rem);
          align-items: flex-start;
        }

        .modal-dialog {
          margin-top: var(--space-4, 1rem);
        }

        .modal-content {
          max-height: calc(100vh - var(--space-4, 1rem));
          border-radius: var(--radius-base, 0.25rem);
        }

        .modal-header,
        .modal-body {
          padding: var(--space-4, 1rem);
        }

        :host([size="small"]) .modal-dialog,
        :host([size="medium"]) .modal-dialog,
        :host([size="large"]) .modal-dialog {
          max-width: none;
          width: 100%;
        }
      }

      /* Dark Mode Support */
      @media (prefers-color-scheme: dark) {
        .modal-backdrop {
          background-color: rgba(0, 0, 0, 0.7);
        }

        .modal-content {
          background: var(--color-bg-primary, #1f2937);
          border: 1px solid var(--color-border, #374151);
        }

        .modal-header {
          border-bottom-color: var(--color-border, #374151);
        }

        .modal-title {
          color: var(--color-text-primary, #f9fafb);
        }

        .modal-body {
          color: var(--color-text-primary, #f9fafb);
        }

        .modal-close {
          color: var(--color-text-secondary, #d1d5db);
        }

        .modal-close:hover,
        .modal-close:focus {
          background-color: var(--color-bg-secondary, #374151);
          color: var(--color-text-primary, #f9fafb);
        }
      }

      /* Reduced Motion Support */
      @media (prefers-reduced-motion: reduce) {
        .modal-backdrop,
        .modal-dialog {
          transition-duration: 0.01ms !important;
        }
      }

      /* High Contrast Mode */
      @media (prefers-contrast: high) {
        .modal-content {
          border: 2px solid currentColor;
        }

        .modal-close:focus-visible {
          outline-width: 3px;
        }
      }

      /* Hidden content for screen readers */
      .sr-only {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
      }

      /* Focus Trap Styles */
      .modal-content:focus {
        outline: none;
      }

      /* Hide modal when not open */
      :host(:not([open])) {
        display: none !important;
      }
    `;
  }

  #attachEventListeners() {
    // Backdrop click to close
    if (this.backdrop) {
      this.backdrop.addEventListener('click', this.boundHandleBackdropClick);
    }

    // Close button click
    if (this.closeButton) {
      this.closeButton.addEventListener('click', this.boundHandleCloseClick);
    }

    // Focus trap on modal content
    if (this.modalContent) {
      this.modalContent.addEventListener('keydown', this.boundHandleFocusTrap);
    }
  }

  #handleKeyDown(event) {
    if (!this.#isOpen) return;

    switch (event.key) {
      case 'Escape':
        if (this.#closable) {
          event.preventDefault();
          this.close();
        }
        break;
    }
  }

  #handleBackdropClick(event) {
    // Only close if clicking directly on the backdrop
    if (event.target === this.backdrop && this.#closable) {
      this.close();
    }
  }

  #handleCloseClick(event) {
    event.preventDefault();
    this.close();
  }

  #handleFocusTrap(event) {
    if (event.key !== 'Tab') return;

    if (!this.#firstFocusableElement || !this.#lastFocusableElement) {
      this.#updateFocusableElements();
    }

    if (event.shiftKey) {
      // Shift + Tab
      if (document.activeElement === this.#firstFocusableElement) {
        event.preventDefault();
        this.#lastFocusableElement.focus();
      }
    } else {
      // Tab
      if (document.activeElement === this.#lastFocusableElement) {
        event.preventDefault();
        this.#firstFocusableElement.focus();
      }
    }
  }

  #updateModalState() {
    if (this.#isOpen) {
      this.setAttribute('open', '');
      this.modal?.setAttribute('aria-hidden', 'false');
    } else {
      this.removeAttribute('open');
      this.modal?.setAttribute('aria-hidden', 'true');
    }
  }

  #updateTitle() {
    if (this.modalTitle) {
      this.modalTitle.textContent = this.#title;
    }
  }

  #updateContent() {
    if (this.modalBody) {
      this.modalBody.innerHTML = this.#content;
    }
  }

  #updateSize() {
    // Size is handled via CSS attribute selectors
  }

  #updateClosable() {
    if (this.closeButton) {
      this.closeButton.style.display = this.#closable ? 'flex' : 'none';
    }
    if (!this.#closable) {
      this.modalHeader?.classList.add('no-close-button');
    } else {
      this.modalHeader?.classList.remove('no-close-button');
    }
  }

  #updateFocusableElements() {
    const focusableSelector = [
      'button:not([disabled])',
      'input:not([disabled])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      'a[href]',
      '[tabindex]:not([tabindex="-1"])',
      '[contenteditable="true"]'
    ].join(', ');

    this.#focusableElements = Array.from(
      this.modalContent?.querySelectorAll(focusableSelector) || []
    );

    this.#firstFocusableElement = this.#focusableElements[0] || this.closeButton || this.modalContent;
    this.#lastFocusableElement = this.#focusableElements[this.#focusableElements.length - 1] || this.closeButton || this.modalContent;
  }

  #setInitialFocus() {
    // Focus the first focusable element or the modal content itself
    const focusTarget = this.#firstFocusableElement || this.modalContent;

    if (focusTarget) {
      // Use setTimeout to ensure the modal is visible before focusing
      setTimeout(() => {
        focusTarget.focus();
      }, 50);
    }
  }

  #announceToScreenReader(message) {
    if (this.liveRegion) {
      this.liveRegion.textContent = message;
      // Clear the message after announcement
      setTimeout(() => {
        this.liveRegion.textContent = '';
      }, 1000);
    }
  }

  #escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  #dispatchEvent(type, detail) {
    const event = new CustomEvent(type, {
      detail,
      bubbles: true,
      cancelable: true
    });

    this.dispatchEvent(event);
    return event;
  }
}

// Register the custom element
customElements.define('modal-component', Modal);

export default Modal;

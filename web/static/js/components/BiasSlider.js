/**
 * BiasSlider Web Component
 * Interactive bias score component with accessibility support
 *
 * Features:
 * - Visual slider with color gradient (red → gray → blue)
 * - Mouse, touch, and keyboard interaction
 * - API integration with error handling
 * - Full accessibility (ARIA, screen reader support)
 * - Size variants and read-only mode
 */

class BiasSlider extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: 'open' });

    // Component state
    this.#value = 0;          // Current bias score (-1 to +1)
    this.#readonly = false;   // Edit mode toggle
    this.#articleId = null;   // Associated article ID
    this.#size = 'medium';    // Component size variant
    this.#isDragging = false; // Drag state
    this.#startValue = 0;     // Value at start of drag

    // Bind event handlers
    this.#handleMouseDown = this.#handleMouseDown.bind(this);
    this.#handleMouseMove = this.#handleMouseMove.bind(this);
    this.#handleMouseUp = this.#handleMouseUp.bind(this);
    this.#handleKeyDown = this.#handleKeyDown.bind(this);
    this.#handleFocus = this.#handleFocus.bind(this);
    this.#handleBlur = this.#handleBlur.bind(this);

    this.#render();
    this.#attachEventListeners();
  }

  static get observedAttributes() {
    return ['value', 'readonly', 'article-id', 'size'];
  }

  // Private properties
  #value = 0;
  #readonly = false;
  #articleId = null;
  #size = 'medium';
  #isDragging = false;
  #startValue = 0;

  // Getters and setters
  get value() {
    return this.#value;
  }

  set value(val) {
    const newValue = this.#clampValue(parseFloat(val) || 0);
    if (newValue !== this.#value) {
      this.#value = newValue;
      this.#updateDisplay();
      this.#announceValueChange();
    }
  }

  get readonly() {
    return this.#readonly;
  }

  set readonly(val) {
    this.#readonly = val === true || val === 'true';
    this.#updateReadonlyState();
  }

  get articleId() {
    return this.#articleId;
  }

  set articleId(val) {
    this.#articleId = val;
  }

  get size() {
    return this.#size;
  }

  set size(val) {
    if (['small', 'medium', 'large'].includes(val)) {
      this.#size = val;
      this.#updateSize();
    }
  }

  // Lifecycle callbacks
  attributeChangedCallback(name, oldValue, newValue) {
    if (oldValue === newValue) return;

    switch (name) {
      case 'value':
        this.value = newValue;
        break;
      case 'readonly':
        this.readonly = newValue;
        break;
      case 'article-id':
        this.articleId = newValue;
        break;
      case 'size':
        this.size = newValue;
        break;
    }
  }

  connectedCallback() {
    // Set initial attributes from element attributes
    if (this.hasAttribute('value')) {
      this.value = this.getAttribute('value');
    }
    if (this.hasAttribute('readonly')) {
      this.readonly = this.getAttribute('readonly');
    }
    if (this.hasAttribute('article-id')) {
      this.articleId = this.getAttribute('article-id');
    }
    if (this.hasAttribute('size')) {
      this.size = this.getAttribute('size');
    }
  }

  disconnectedCallback() {
    this.#removeEventListeners();
  }

  // Public methods
  async updateValue(newValue) {
    const clampedValue = this.#clampValue(newValue);
    const oldValue = this.#value;

    // Optimistic update
    this.value = clampedValue;

    if (this.#articleId && window.ApiClient) {
      try {
        await window.ApiClient.updateBias(this.#articleId, clampedValue);
        this.#dispatchEvent('biasupdate', {
          value: clampedValue,
          articleId: this.#articleId,
          success: true
        });
      } catch (error) {
        // Rollback on error
        this.value = oldValue;
        this.#dispatchEvent('apierror', {
          error: error.message,
          articleId: this.#articleId,
          attemptedValue: clampedValue
        });
        throw error;
      }
    }
  }

  enableEditMode() {
    this.readonly = false;
  }

  disableEditMode() {
    this.readonly = true;
  }

  // Private methods
  #render() {
    const template = document.createElement('template');
    template.innerHTML = `
      <style>
        :host {
          display: block;
          --slider-height: 8px;
          --thumb-size: 20px;
          --track-radius: 4px;
          --color-bias-left: #dc2626;
          --color-bias-center: #6b7280;
          --color-bias-right: #2563eb;
          --transition-base: 200ms ease;
        }

        :host([size="small"]) {
          --slider-height: 6px;
          --thumb-size: 16px;
        }

        :host([size="large"]) {
          --slider-height: 10px;
          --thumb-size: 24px;
        }

        .bias-slider {
          position: relative;
          width: 100%;
          height: var(--thumb-size);
          margin: 8px 0;
          cursor: pointer;
        }

        .bias-slider[readonly] {
          cursor: default;
        }

        .bias-slider__track {
          position: absolute;
          top: 50%;
          left: 0;
          right: 0;
          height: var(--slider-height);
          transform: translateY(-50%);
          background: linear-gradient(
            to right,
            var(--color-bias-left) 0%,
            var(--color-bias-center) 50%,
            var(--color-bias-right) 100%
          );
          border-radius: var(--track-radius);
          overflow: hidden;
        }

        .bias-slider__thumb {
          position: absolute;
          top: 50%;
          width: var(--thumb-size);
          height: var(--thumb-size);
          background: white;
          border: 2px solid #374151;
          border-radius: 50%;
          transform: translate(-50%, -50%);
          transition: var(--transition-base);
          cursor: grab;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        }

        .bias-slider__thumb:hover {
          border-color: #1f2937;
          box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
        }

        .bias-slider__thumb:focus {
          outline: 2px solid #3b82f6;
          outline-offset: 2px;
        }

        .bias-slider__thumb[readonly] {
          cursor: default;
          opacity: 0.7;
        }

        .bias-slider__thumb:active,
        .bias-slider__thumb.dragging {
          cursor: grabbing;
          transform: translate(-50%, -50%) scale(1.1);
        }

        .bias-slider__value {
          position: absolute;
          top: 100%;
          left: 50%;
          transform: translateX(-50%);
          font-size: 0.75rem;
          color: #6b7280;
          margin-top: 4px;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }

        .bias-slider__labels {
          display: flex;
          justify-content: space-between;
          margin-top: 20px;
          font-size: 0.75rem;
          color: #9ca3af;
        }

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
      </style>

      <div class="bias-slider" role="slider"
           aria-valuemin="-1"
           aria-valuemax="1"
           aria-valuenow="0"
           aria-label="Bias score slider"
           tabindex="0">
        <div class="bias-slider__track"></div>
        <div class="bias-slider__thumb" tabindex="-1"></div>
        <div class="bias-slider__value">0.0</div>
        <div class="bias-slider__labels">
          <span>Left</span>
          <span>Center</span>
          <span>Right</span>
        </div>
        <div class="sr-only" aria-live="polite" aria-atomic="true"></div>
      </div>
    `;

    this.shadowRoot.appendChild(template.content.cloneNode(true));

    // Cache DOM elements
    this.slider = this.shadowRoot.querySelector('.bias-slider');
    this.thumb = this.shadowRoot.querySelector('.bias-slider__thumb');
    this.valueDisplay = this.shadowRoot.querySelector('.bias-slider__value');
    this.announcer = this.shadowRoot.querySelector('.sr-only');
  }

  #attachEventListeners() {
    this.slider.addEventListener('mousedown', this.#handleMouseDown);
    this.slider.addEventListener('keydown', this.#handleKeyDown);
    this.slider.addEventListener('focus', this.#handleFocus);
    this.slider.addEventListener('blur', this.#handleBlur);

    // Touch events for mobile
    this.slider.addEventListener('touchstart', this.#handleTouchStart.bind(this));
    this.slider.addEventListener('touchmove', this.#handleTouchMove.bind(this));
    this.slider.addEventListener('touchend', this.#handleTouchEnd.bind(this));
  }

  #removeEventListeners() {
    document.removeEventListener('mousemove', this.#handleMouseMove);
    document.removeEventListener('mouseup', this.#handleMouseUp);
  }

  #handleMouseDown(event) {
    if (this.#readonly) return;

    event.preventDefault();
    this.#isDragging = true;
    this.#startValue = this.#value;

    document.addEventListener('mousemove', this.#handleMouseMove);
    document.addEventListener('mouseup', this.#handleMouseUp);

    this.thumb.classList.add('dragging');

    // Update value based on click position
    this.#updateValueFromPosition(event.clientX);
  }

  #handleMouseMove(event) {
    if (!this.#isDragging || this.#readonly) return;

    event.preventDefault();
    this.#updateValueFromPosition(event.clientX);
  }

  #handleMouseUp(event) {
    if (!this.#isDragging) return;

    this.#isDragging = false;
    this.thumb.classList.remove('dragging');

    document.removeEventListener('mousemove', this.#handleMouseMove);
    document.removeEventListener('mouseup', this.#handleMouseUp);

    // Trigger API update if value changed
    if (this.#value !== this.#startValue) {
      this.#dispatchEvent('biaschange', {
        value: this.#value,
        oldValue: this.#startValue,
        articleId: this.#articleId
      });

      if (this.#articleId) {
        this.updateValue(this.#value).catch(console.error);
      }
    }
  }

  #handleTouchStart(event) {
    if (this.#readonly) return;

    event.preventDefault();
    const touch = event.touches[0];
    this.#isDragging = true;
    this.#startValue = this.#value;
    this.#updateValueFromPosition(touch.clientX);
  }

  #handleTouchMove(event) {
    if (!this.#isDragging || this.#readonly) return;

    event.preventDefault();
    const touch = event.touches[0];
    this.#updateValueFromPosition(touch.clientX);
  }

  #handleTouchEnd(event) {
    if (!this.#isDragging) return;

    this.#isDragging = false;

    if (this.#value !== this.#startValue) {
      this.#dispatchEvent('biaschange', {
        value: this.#value,
        oldValue: this.#startValue,
        articleId: this.#articleId
      });

      if (this.#articleId) {
        this.updateValue(this.#value).catch(console.error);
      }
    }
  }

  #handleKeyDown(event) {
    if (this.#readonly) return;

    let newValue = this.#value;
    const step = 0.1;

    switch (event.key) {
      case 'ArrowLeft':
      case 'ArrowDown':
        newValue = this.#value - step;
        break;
      case 'ArrowRight':
      case 'ArrowUp':
        newValue = this.#value + step;
        break;
      case 'Home':
        newValue = -1;
        break;
      case 'End':
        newValue = 1;
        break;
      case 'PageDown':
        newValue = this.#value - 0.5;
        break;
      case 'PageUp':
        newValue = this.#value + 0.5;
        break;
      default:
        return; // Don't prevent default for other keys
    }

    event.preventDefault();
    const oldValue = this.#value;
    this.value = newValue;

    if (this.#value !== oldValue) {
      this.#dispatchEvent('biaschange', {
        value: this.#value,
        oldValue: oldValue,
        articleId: this.#articleId
      });

      if (this.#articleId) {
        this.updateValue(this.#value).catch(console.error);
      }
    }
  }

  #handleFocus() {
    this.thumb.style.transform = 'translate(-50%, -50%) scale(1.05)';
  }

  #handleBlur() {
    this.thumb.style.transform = 'translate(-50%, -50%)';
  }

  #updateValueFromPosition(clientX) {
    const rect = this.slider.getBoundingClientRect();
    const percentage = (clientX - rect.left) / rect.width;
    const newValue = (percentage * 2) - 1; // Convert to -1 to 1 range
    this.value = newValue;

    // Emit live update event during drag
    this.#dispatchEvent('biasupdate', {
      value: this.#value,
      articleId: this.#articleId,
      live: true
    });
  }

  #updateDisplay() {
    const percentage = ((this.#value + 1) / 2) * 100;
    this.thumb.style.left = `${percentage}%`;
    this.valueDisplay.textContent = this.#value.toFixed(1);
    this.slider.setAttribute('aria-valuenow', this.#value.toFixed(2));
  }

  #updateReadonlyState() {
    if (this.#readonly) {
      this.slider.setAttribute('readonly', '');
      this.thumb.setAttribute('readonly', '');
      this.slider.setAttribute('aria-readonly', 'true');
    } else {
      this.slider.removeAttribute('readonly');
      this.thumb.removeAttribute('readonly');
      this.slider.setAttribute('aria-readonly', 'false');
    }
  }

  #updateSize() {
    this.setAttribute('size', this.#size);
  }

  #clampValue(value) {
    return Math.max(-1, Math.min(1, value));
  }

  #announceValueChange() {
    const biasText = this.#getBiasText(this.#value);
    this.announcer.textContent = `Bias score: ${this.#value.toFixed(1)}, ${biasText}`;
  }

  #getBiasText(value) {
    if (value < -0.3) return 'left leaning';
    if (value > 0.3) return 'right leaning';
    return 'center';
  }

  #dispatchEvent(eventName, detail) {
    this.dispatchEvent(new CustomEvent(eventName, {
      detail,
      bubbles: true,
      composed: true
    }));
  }
}

// Register the custom element
customElements.define('bias-slider', BiasSlider);

export default BiasSlider;

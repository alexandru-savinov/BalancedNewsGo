/**
 * ArticleCard Web Component
 * Reusable article preview component with BiasSlider integration
 *
 * Features:
 * - Article data display with metadata
 * - Embedded BiasSlider component
 * - Compact and regular layout modes
 * - Click navigation handling
 * - Accessibility support
 * - Responsive design
 */

class ArticleCard extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: 'open' });

    // Component state
    this.#article = null;
    this.#biasSlider = null;
    this.#showBiasSlider = true;
    this.#compact = false;
    this.#clickable = true;

    // Bind event handlers
    this.#handleCardClick = this.#handleCardClick.bind(this);
    this.#handleActionClick = this.#handleActionClick.bind(this);
    this.#handleBiasChange = this.#handleBiasChange.bind(this);

    this.#render();
    this.#attachEventListeners();
  }

  static get observedAttributes() {
    return ['article-data', 'show-bias-slider', 'compact', 'clickable'];
  }

  // Private properties
  #article = null;
  #biasSlider = null;
  #showBiasSlider = true;
  #compact = false;
  #clickable = true;

  // Getters and setters
  get article() {
    return this.#article;
  }

  set article(data) {
    if (typeof data === 'string') {
      try {
        this.#article = JSON.parse(data);
      } catch (error) {
        console.error('Invalid article data JSON:', error);
        return;
      }
    } else {
      this.#article = data;
    }
    this.#updateContent();
  }

  get showBiasSlider() {
    return this.#showBiasSlider;
  }

  set showBiasSlider(value) {
    this.#showBiasSlider = value === true || value === 'true';
    this.#updateBiasSliderVisibility();
  }

  get compact() {
    return this.#compact;
  }

  set compact(value) {
    this.#compact = value === true || value === 'true';
    this.#updateLayout();
  }

  get clickable() {
    return this.#clickable;
  }

  set clickable(value) {
    this.#clickable = value === true || value === 'true';
    this.#updateClickableState();
  }

  // Lifecycle callbacks
  attributeChangedCallback(name, oldValue, newValue) {
    if (oldValue === newValue) return;

    switch (name) {
      case 'article-data':
        this.article = newValue;
        break;
      case 'show-bias-slider':
        this.showBiasSlider = newValue;
        break;
      case 'compact':
        this.compact = newValue;
        break;
      case 'clickable':
        this.clickable = newValue;
        break;
    }
  }

  connectedCallback() {
    // Set initial attributes from element attributes
    if (this.hasAttribute('article-data')) {
      this.article = this.getAttribute('article-data');
    }
    if (this.hasAttribute('show-bias-slider')) {
      this.showBiasSlider = this.getAttribute('show-bias-slider');
    }
    if (this.hasAttribute('compact')) {
      this.compact = this.getAttribute('compact');
    }
    if (this.hasAttribute('clickable')) {
      this.clickable = this.getAttribute('clickable');
    }
  }

  disconnectedCallback() {
    this.#removeEventListeners();
  }

  // Public methods
  updateArticle(articleData) {
    this.article = articleData;
  }

  toggleCompactMode() {
    this.compact = !this.compact;
  }

  // Private methods
  #render() {
    const template = document.createElement('template');
    template.innerHTML = `
      <style>
        :host {
          display: block;
          --card-padding: 1.5rem;
          --card-radius: 8px;
          --transition-base: 200ms ease;
          --color-primary: #3b82f6;
          --color-secondary: #6b7280;
          --color-text-primary: #1f2937;
          --color-text-secondary: #4b5563;
          --color-border: #e5e7eb;
          --color-bg-primary: white;
          --color-bg-secondary: #f9fafb;
        }

        :host([compact]) {
          --card-padding: 1rem;
        }

        .article-card {
          background: var(--color-bg-primary);
          border: 1px solid var(--color-border);
          border-radius: var(--card-radius);
          padding: var(--card-padding);
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
          transition: box-shadow var(--transition-base), transform var(--transition-base);
          display: flex;
          flex-direction: column;
          height: 100%;
          cursor: pointer;
          position: relative;
        }

        :host(:not([clickable])) .article-card {
          cursor: default;
        }

        .article-card:hover {
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
          transform: translateY(-2px);
        }

        :host(:not([clickable])) .article-card:hover {
          transform: none;
        }

        .article-card__header {
          margin-bottom: 1rem;
        }

        :host([compact]) .article-card__header {
          margin-bottom: 0.75rem;
        }

        .article-card__title {
          font-size: 1.125rem;
          font-weight: 600;
          line-height: 1.4;
          margin: 0 0 0.5rem 0;
          color: var(--color-text-primary);
        }

        :host([compact]) .article-card__title {
          font-size: 1rem;
          margin-bottom: 0.25rem;
        }

        .article-card__link {
          color: inherit;
          text-decoration: none;
          transition: color var(--transition-base);
        }

        .article-card__link:hover {
          color: var(--color-primary);
        }

        .article-card__meta {
          display: flex;
          justify-content: space-between;
          align-items: center;
          font-size: 0.875rem;
          color: var(--color-secondary);
          margin-bottom: 0.5rem;
          flex-wrap: wrap;
          gap: 0.5rem;
        }

        :host([compact]) .article-card__meta {
          font-size: 0.8125rem;
          margin-bottom: 0.25rem;
        }

        .article-card__source {
          font-weight: 500;
          color: var(--color-text-secondary);
        }

        .article-card__date {
          color: var(--color-secondary);
        }

        .article-card__content {
          flex: 1;
          margin-bottom: 1.5rem;
        }

        :host([compact]) .article-card__content {
          margin-bottom: 1rem;
        }

        .article-card__excerpt {
          color: var(--color-text-secondary);
          line-height: 1.5;
          margin: 0;
          font-size: 0.9375rem;
        }

        :host([compact]) .article-card__excerpt {
          font-size: 0.875rem;
          line-height: 1.4;
        }

        .article-card__footer {
          margin-top: auto;
        }

        .bias-slider-container {
          margin: 1rem 0;
          padding: 0.75rem;
          background: var(--color-bg-secondary);
          border-radius: 6px;
          border: 1px solid var(--color-border);
        }

        :host([compact]) .bias-slider-container {
          margin: 0.75rem 0;
          padding: 0.5rem;
        }

        .bias-slider-container[hidden] {
          display: none;
        }

        .bias-slider-label {
          display: block;
          font-size: 0.875rem;
          font-weight: 500;
          color: var(--color-text-secondary);
          margin-bottom: 0.5rem;
        }

        :host([compact]) .bias-slider-label {
          font-size: 0.8125rem;
          margin-bottom: 0.25rem;
        }

        .bias-slider-description {
          font-size: 0.75rem;
          color: var(--color-secondary);
          margin-top: 0.5rem;
          text-align: center;
        }

        :host([compact]) .bias-slider-description {
          font-size: 0.6875rem;
          margin-top: 0.25rem;
        }

        .article-card__actions {
          display: flex;
          gap: 0.5rem;
          margin-top: 1rem;
        }

        :host([compact]) .article-card__actions {
          margin-top: 0.75rem;
          gap: 0.375rem;
        }

        .btn {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          padding: 0.5rem 1rem;
          border-radius: 0.375rem;
          font-size: 0.875rem;
          font-weight: 500;
          text-decoration: none;
          border: none;
          cursor: pointer;
          transition: all var(--transition-base);
          flex: 1;
        }

        :host([compact]) .btn {
          padding: 0.375rem 0.75rem;
          font-size: 0.8125rem;
        }

        .btn--primary {
          background: var(--color-primary);
          color: white;
        }

        .btn--primary:hover {
          background: #2563eb;
        }

        .btn--secondary {
          background: var(--color-bg-primary);
          color: var(--color-text-secondary);
          border: 1px solid var(--color-border);
        }

        .btn--secondary:hover {
          background: var(--color-bg-secondary);
          border-color: var(--color-secondary);
        }

        /* Loading state */
        .loading-state {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 2rem;
          color: var(--color-secondary);
        }

        .loading-spinner {
          width: 20px;
          height: 20px;
          border: 2px solid var(--color-border);
          border-top-color: var(--color-primary);
          border-radius: 50%;
          animation: spin 1s linear infinite;
          margin-right: 0.5rem;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        /* Dark mode support */
        @media (prefers-color-scheme: dark) {
          :host {
            --color-bg-primary: #1f2937;
            --color-bg-secondary: #374151;
            --color-text-primary: #f9fafb;
            --color-text-secondary: #d1d5db;
            --color-border: #374151;
            --color-secondary: #9ca3af;
          }
        }

        /* Reduced motion support */
        @media (prefers-reduced-motion: reduce) {
          :host {
            --transition-base: 0ms;
          }

          .article-card:hover {
            transform: none;
          }
        }

        /* Mobile responsive */
        @media (max-width: 768px) {
          .article-card__actions {
            flex-direction: column;
          }

          .article-card__meta {
            flex-direction: column;
            align-items: flex-start;
          }
        }
      </style>

      <article class="article-card" role="article">
        <div class="article-card__header">
          <h3 class="article-card__title">
            <a href="#" class="article-card__link" role="button" tabindex="0">
              Article Title
            </a>
          </h3>
          <div class="article-card__meta">
            <span class="article-card__source">Source</span>
            <time class="article-card__date" datetime="">Date</time>
          </div>
        </div>

        <div class="article-card__content">
          <p class="article-card__excerpt">Article excerpt will appear here...</p>
        </div>

        <div class="article-card__footer">
          <div class="bias-slider-container">
            <label class="bias-slider-label">Bias Score</label>
            <bias-slider value="0" size="small"></bias-slider>
            <div class="bias-slider-description">
              Confidence: 0% • Source: N/A
            </div>
          </div>

          <div class="article-card__actions">
            <a href="#" class="btn btn--primary" data-action="read-more">
              Read More
            </a>
            <a href="#" class="btn btn--secondary" data-action="original-source"
               target="_blank" rel="noopener noreferrer">
              Original
            </a>
          </div>
        </div>

        <div class="loading-state" hidden>
          <div class="loading-spinner"></div>
          <span>Loading...</span>
        </div>
      </article>
    `;

    this.shadowRoot.appendChild(template.content.cloneNode(true));

    // Cache DOM elements
    this.card = this.shadowRoot.querySelector('.article-card');
    this.titleElement = this.shadowRoot.querySelector('.article-card__title');
    this.linkElement = this.shadowRoot.querySelector('.article-card__link');
    this.sourceElement = this.shadowRoot.querySelector('.article-card__source');
    this.dateElement = this.shadowRoot.querySelector('.article-card__date');
    this.excerptElement = this.shadowRoot.querySelector('.article-card__excerpt');
    this.biasSliderContainer = this.shadowRoot.querySelector('.bias-slider-container');
    this.biasSliderElement = this.shadowRoot.querySelector('bias-slider');
    this.biasDescriptionElement = this.shadowRoot.querySelector('.bias-slider-description');
    this.actionsContainer = this.shadowRoot.querySelector('.article-card__actions');
    this.readMoreButton = this.shadowRoot.querySelector('[data-action="read-more"]');
    this.originalSourceButton = this.shadowRoot.querySelector('[data-action="original-source"]');
    this.loadingState = this.shadowRoot.querySelector('.loading-state');
  }

  #attachEventListeners() {
    // Card click for navigation
    this.card.addEventListener('click', this.#handleCardClick);

    // Action button clicks
    this.readMoreButton.addEventListener('click', this.#handleActionClick);
    this.originalSourceButton.addEventListener('click', this.#handleActionClick);

    // Bias slider events
    this.biasSliderElement.addEventListener('biaschange', this.#handleBiasChange);

    // Keyboard navigation for card
    this.card.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        this.#handleCardClick(event);
      }
    });
  }

  #removeEventListeners() {
    // Event listeners are automatically removed when element is removed from DOM
  }

  #handleCardClick(event) {
    if (!this.#clickable || !this.#article) return;

    // Don't navigate if clicking on action buttons or bias slider
    if (event.target.closest('.article-card__actions') ||
        event.target.closest('.bias-slider-container')) {
      return;
    }

    event.preventDefault();

    this.#dispatchEvent('articleclick', {
      article: this.#article,
      target: event.target
    });
  }

  #handleActionClick(event) {
    event.stopPropagation(); // Prevent card click

    const action = event.target.dataset.action;

    this.#dispatchEvent('articleaction', {
      action,
      article: this.#article,
      target: event.target
    });
  }

  #handleBiasChange(event) {
    // Forward bias change events
    this.#dispatchEvent('biaschange', {
      ...event.detail,
      article: this.#article
    });
  }

  #updateContent() {
    if (!this.#article) {
      this.#showLoadingState();
      return;
    }

    this.#hideLoadingState();

    // Update title and link
    const title = this.#escapeHtml(this.#article.title || 'Untitled Article');
    this.titleElement.querySelector('.article-card__link').textContent = title;
    this.linkElement.href = `/article/${this.#article.id}`;

    // Update meta information
    this.sourceElement.textContent = this.#escapeHtml(this.#article.source || 'Unknown Source');

    const date = new Date(this.#article.pub_date || this.#article.publishedAt);
    this.dateElement.textContent = date.toLocaleDateString();
    this.dateElement.setAttribute('datetime', date.toISOString());

    // Update excerpt
    const content = this.#article.content || this.#article.summary || '';
    const excerpt = this.#truncateText(content, this.#compact ? 100 : 150);
    this.excerptElement.textContent = this.#escapeHtml(excerpt);

    // Update bias slider
    const biasScore = this.#article.composite_score || this.#article.bias?.score || 0;
    this.biasSliderElement.value = biasScore;
    this.biasSliderElement.setAttribute('article-id', this.#article.id);

    // Update bias description
    const confidence = this.#article.confidence || this.#article.bias?.confidence || 0;
    const scoreSource = this.#article.score_source || this.#article.bias?.modelScores?.[0]?.model || 'N/A';
    this.biasDescriptionElement.innerHTML = `
      Confidence: ${Math.round(confidence * 100)}% • Source: ${this.#escapeHtml(scoreSource)}
    `;

    // Update action button links
    this.readMoreButton.href = `/article/${this.#article.id}`;
    this.originalSourceButton.href = this.#article.url || '#';
  }

  #updateBiasSliderVisibility() {
    if (this.biasSliderContainer) {
      this.biasSliderContainer.hidden = !this.#showBiasSlider;
    }
  }

  #updateLayout() {
    // Layout changes are handled via CSS :host([compact]) selectors
  }

  #updateClickableState() {
    if (this.card) {
      this.card.style.cursor = this.#clickable ? 'pointer' : 'default';
    }
  }

  #showLoadingState() {
    if (this.loadingState) {
      this.loadingState.hidden = false;
    }
  }

  #hideLoadingState() {
    if (this.loadingState) {
      this.loadingState.hidden = true;
    }
  }

  #escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  #truncateText(text, maxLength) {
    if (text.length <= maxLength) return text;
    return text.slice(0, maxLength).trim() + '...';
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
customElements.define('article-card', ArticleCard);

export default ArticleCard;

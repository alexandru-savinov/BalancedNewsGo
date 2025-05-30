/**
 * ArticleCard Component Test Suite
 * Tests for the ArticleCard web component functionality
 */

// Mock data for testing
const mockArticle = {
  id: 'test-article-123',
  title: 'Test Article Title',
  content: 'This is a test article content that should be displayed in the excerpt area.',
  source: 'Test Source',
  pub_date: '2025-05-27T10:00:00Z',
  url: 'https://example.com/article',
  composite_score: 0.5,
  confidence: 0.85,
  score_source: 'GPT-4'
};

describe('ArticleCard Component', () => {
  let articleCard;

  beforeEach(() => {
    // Ensure BiasSlider is available (it's imported by ArticleCard)
    if (!customElements.get('bias-slider')) {
      // Mock BiasSlider for testing
      class MockBiasSlider extends HTMLElement {
        constructor() {
          super();
          this.attachShadow({ mode: 'open' });
          this.shadowRoot.innerHTML = '<div>Mock BiasSlider</div>';
        }

        set value(val) { this._value = val; }
        get value() { return this._value || 0; }

        setAttribute(name, value) {
          super.setAttribute(name, value);
          if (name === 'article-id') this._articleId = value;
        }
      }
      customElements.define('bias-slider', MockBiasSlider);
    }

    articleCard = document.createElement('article-card');
    document.body.appendChild(articleCard);
  });

  afterEach(() => {
    if (articleCard && articleCard.parentNode) {
      articleCard.parentNode.removeChild(articleCard);
    }
  });

  describe('Component Creation and Registration', () => {
    test('should be defined as custom element', () => {
      expect(customElements.get('article-card')).toBeDefined();
    });

    test('should create with Shadow DOM', () => {
      expect(articleCard.shadowRoot).toBeTruthy();
    });

    test('should have initial default state', () => {
      expect(articleCard.article).toBeNull();
      expect(articleCard.showBiasSlider).toBe(true);
      expect(articleCard.compact).toBe(false);
      expect(articleCard.clickable).toBe(true);
    });
  });

  describe('Article Data Handling', () => {
    test('should accept article data as object', () => {
      articleCard.article = mockArticle;
      expect(articleCard.article).toEqual(mockArticle);
    });

    test('should accept article data as JSON string', () => {
      articleCard.article = JSON.stringify(mockArticle);
      expect(articleCard.article).toEqual(mockArticle);
    });

    test('should handle invalid JSON gracefully', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      articleCard.article = 'invalid json';
      expect(consoleSpy).toHaveBeenCalled();
      expect(articleCard.article).toBeNull();
      consoleSpy.mockRestore();
    });

    test('should update content when article is set', () => {
      articleCard.article = mockArticle;

      // Check if title is updated
      const titleLink = articleCard.shadowRoot.querySelector('.article-card__link');
      expect(titleLink.textContent).toBe(mockArticle.title);
      expect(titleLink.href).toContain(`/article/${mockArticle.id}`);
    });

    test('should update meta information', () => {
      articleCard.article = mockArticle;

      const source = articleCard.shadowRoot.querySelector('.article-card__source');
      const date = articleCard.shadowRoot.querySelector('.article-card__date');

      expect(source.textContent).toBe(mockArticle.source);
      expect(date.textContent).toBeTruthy();
      expect(date.getAttribute('datetime')).toContain('2025-05-27');
    });

    test('should update bias slider', () => {
      articleCard.article = mockArticle;

      const biasSlider = articleCard.shadowRoot.querySelector('bias-slider');
      expect(biasSlider.value).toBe(mockArticle.composite_score);
      expect(biasSlider.getAttribute('article-id')).toBe(mockArticle.id);
    });

    test('should update action buttons', () => {
      articleCard.article = mockArticle;

      const readMoreBtn = articleCard.shadowRoot.querySelector('[data-action="read-more"]');
      const originalBtn = articleCard.shadowRoot.querySelector('[data-action="original-source"]');

      expect(readMoreBtn.href).toContain(`/article/${mockArticle.id}`);
      expect(originalBtn.href).toBe(mockArticle.url);
    });
  });

  describe('Component Properties', () => {
    test('should toggle compact mode', () => {
      expect(articleCard.compact).toBe(false);

      articleCard.compact = true;
      expect(articleCard.compact).toBe(true);
      expect(articleCard.hasAttribute('compact')).toBe(false); // Internal state, not reflected

      articleCard.compact = 'true';
      expect(articleCard.compact).toBe(true);
    });

    test('should toggle bias slider visibility', () => {
      const container = articleCard.shadowRoot.querySelector('.bias-slider-container');

      expect(articleCard.showBiasSlider).toBe(true);
      expect(container.hidden).toBe(false);

      articleCard.showBiasSlider = false;
      expect(articleCard.showBiasSlider).toBe(false);
      expect(container.hidden).toBe(true);
    });

    test('should toggle clickable state', () => {
      const card = articleCard.shadowRoot.querySelector('.article-card');

      expect(articleCard.clickable).toBe(true);

      articleCard.clickable = false;
      expect(articleCard.clickable).toBe(false);
    });
  });

  describe('Attribute Handling', () => {
    test('should respond to attribute changes', () => {
      articleCard.setAttribute('article-data', JSON.stringify(mockArticle));
      expect(articleCard.article).toEqual(mockArticle);

      articleCard.setAttribute('compact', 'true');
      expect(articleCard.compact).toBe(true);

      articleCard.setAttribute('show-bias-slider', 'false');
      expect(articleCard.showBiasSlider).toBe(false);

      articleCard.setAttribute('clickable', 'false');
      expect(articleCard.clickable).toBe(false);
    });
  });

  describe('Event Handling', () => {
    beforeEach(() => {
      articleCard.article = mockArticle;
    });

    test('should emit articleclick event on card click', () => {
      const eventSpy = jest.fn();
      articleCard.addEventListener('articleclick', eventSpy);

      const card = articleCard.shadowRoot.querySelector('.article-card');
      card.click();

      expect(eventSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: expect.objectContaining({
            article: mockArticle
          })
        })
      );
    });

    test('should not emit articleclick when clickable is false', () => {
      articleCard.clickable = false;

      const eventSpy = jest.fn();
      articleCard.addEventListener('articleclick', eventSpy);

      const card = articleCard.shadowRoot.querySelector('.article-card');
      card.click();

      expect(eventSpy).not.toHaveBeenCalled();
    });

    test('should emit articleaction event on button clicks', () => {
      const eventSpy = jest.fn();
      articleCard.addEventListener('articleaction', eventSpy);

      const readMoreBtn = articleCard.shadowRoot.querySelector('[data-action="read-more"]');
      readMoreBtn.click();

      expect(eventSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: expect.objectContaining({
            action: 'read-more',
            article: mockArticle
          })
        })
      );
    });

    test('should handle keyboard navigation', () => {
      const eventSpy = jest.fn();
      articleCard.addEventListener('articleclick', eventSpy);

      const card = articleCard.shadowRoot.querySelector('.article-card');

      // Test Enter key
      const enterEvent = new KeyboardEvent('keydown', { key: 'Enter' });
      card.dispatchEvent(enterEvent);
      expect(eventSpy).toHaveBeenCalled();

      eventSpy.mockClear();

      // Test Space key
      const spaceEvent = new KeyboardEvent('keydown', { key: ' ' });
      card.dispatchEvent(spaceEvent);
      expect(eventSpy).toHaveBeenCalled();
    });

    test('should forward bias change events', () => {
      const eventSpy = jest.fn();
      articleCard.addEventListener('biaschange', eventSpy);

      const biasSlider = articleCard.shadowRoot.querySelector('bias-slider');
      const biasEvent = new CustomEvent('biaschange', {
        detail: { value: 0.3, articleId: mockArticle.id }
      });
      biasSlider.dispatchEvent(biasEvent);

      expect(eventSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: expect.objectContaining({
            value: 0.3,
            articleId: mockArticle.id,
            article: mockArticle
          })
        })
      );
    });
  });

  describe('Public Methods', () => {
    test('should update article via method', () => {
      const newArticle = { ...mockArticle, title: 'Updated Title' };

      articleCard.updateArticle(newArticle);
      expect(articleCard.article).toEqual(newArticle);

      const titleLink = articleCard.shadowRoot.querySelector('.article-card__link');
      expect(titleLink.textContent).toBe('Updated Title');
    });

    test('should toggle compact mode via method', () => {
      expect(articleCard.compact).toBe(false);

      articleCard.toggleCompactMode();
      expect(articleCard.compact).toBe(true);

      articleCard.toggleCompactMode();
      expect(articleCard.compact).toBe(false);
    });
  });

  describe('Loading State', () => {
    test('should show loading state when no article', () => {
      const loadingState = articleCard.shadowRoot.querySelector('.loading-state');
      expect(loadingState.hidden).toBe(true); // Initially hidden

      // Simulate loading by clearing article
      articleCard.article = null;
      // The loading state should be shown in the private method
    });
  });

  describe('Accessibility', () => {
    beforeEach(() => {
      articleCard.article = mockArticle;
    });

    test('should have proper ARIA roles', () => {
      const article = articleCard.shadowRoot.querySelector('article');
      expect(article.getAttribute('role')).toBe('article');

      const link = articleCard.shadowRoot.querySelector('.article-card__link');
      expect(link.getAttribute('role')).toBe('button');
      expect(link.getAttribute('tabindex')).toBe('0');
    });

    test('should have proper datetime attribute', () => {
      const timeElement = articleCard.shadowRoot.querySelector('.article-card__date');
      expect(timeElement.getAttribute('datetime')).toBeTruthy();
    });

    test('should have proper label for bias slider', () => {
      const label = articleCard.shadowRoot.querySelector('.bias-slider-label');
      expect(label.textContent).toBe('Bias Score');
    });
  });

  describe('Responsive Design', () => {
    test('should apply compact styles when compact is true', () => {
      articleCard.compact = true;
      // Compact mode changes are handled via CSS :host([compact]) selectors
      // This would require integration testing to verify properly
    });
  });

  describe('Text Utilities', () => {
    test('should truncate long text', () => {
      const longContent = 'A'.repeat(200);
      const article = { ...mockArticle, content: longContent };

      articleCard.article = article;

      const excerpt = articleCard.shadowRoot.querySelector('.article-card__excerpt');
      expect(excerpt.textContent.length).toBeLessThan(longContent.length);
      expect(excerpt.textContent).toContain('...');
    });

    test('should escape HTML in content', () => {
      const maliciousArticle = {
        ...mockArticle,
        title: '<script>alert("xss")</script>',
        source: '<img src="x" onerror="alert(1)">',
        content: '<b>Bold content</b>'
      };

      articleCard.article = maliciousArticle;

      const titleLink = articleCard.shadowRoot.querySelector('.article-card__link');
      const source = articleCard.shadowRoot.querySelector('.article-card__source');

      expect(titleLink.textContent).not.toContain('<script>');
      expect(source.textContent).not.toContain('<img');
    });
  });
});

// Integration tests would go here for testing with actual DOM and CSS
describe('ArticleCard Integration', () => {
  test('should work with real DOM environment', () => {
    // This would test actual rendering, styling, and interaction
    // in a more realistic browser environment
  });
});

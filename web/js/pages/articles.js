/**
 * Articles Page JavaScript
 * Handles article listing, filtering, and BiasSlider integration
 */

class ArticlesPage {
  constructor() {
    this.currentView = 'grid';
    this.currentFilters = {
      search: '',
      bias: 'all',
      date: 'all',
      sort: 'date-desc'
    };
    this.currentPage = 1;
    this.articlesPerPage = 12;
    this.isLoading = false;

    this.init();
  }
  async init() {
    this.bindEventListeners();
    await this.loadArticles();
    this.setupBiasSliderEvents();
    // setupArticleCardEvents is called after renderArticles
  }

  bindEventListeners() {
    // Search input
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
      searchInput.addEventListener('input', this.debounce(() => {
        this.currentFilters.search = searchInput.value;
        this.refreshArticles();
      }, 300));
    }

    // Filter selects
    const biasFilter = document.getElementById('bias-filter');
    if (biasFilter) {
      biasFilter.addEventListener('change', (e) => {
        this.currentFilters.bias = e.target.value;
        this.refreshArticles();
      });
    }

    const dateFilter = document.getElementById('date-filter');
    if (dateFilter) {
      dateFilter.addEventListener('change', (e) => {
        this.currentFilters.date = e.target.value;
        this.refreshArticles();
      });
    }

    const sortSelect = document.getElementById('sort-select');
    if (sortSelect) {
      sortSelect.addEventListener('change', (e) => {
        this.currentFilters.sort = e.target.value;
        this.refreshArticles();
      });
    }

    // View toggle buttons
    const viewButtons = document.querySelectorAll('.view-toggle-btn');
    viewButtons.forEach(button => {
      button.addEventListener('click', (e) => {
        this.switchView(e.target.dataset.view);
      });
    });

    // Search form submission
    const searchForm = document.getElementById('search-form');
    if (searchForm) {
      searchForm.addEventListener('submit', (e) => {
        e.preventDefault();
        this.refreshArticles();
      });
    }
  }

  setupBiasSliderEvents() {
    // Listen for bias slider changes globally
    document.addEventListener('biaschange', (event) => {
      console.log('Bias changed:', event.detail);
      this.handleBiasChange(event.detail);
    });

    document.addEventListener('apierror', (event) => {
      console.error('Bias update error:', event.detail);
      this.showErrorToast(`Failed to update bias: ${event.detail.error}`);
    });
  }

  async handleBiasChange(detail) {
    const { value, articleId } = detail;

    // Update the visual feedback immediately
    this.showSuccessToast(`Bias updated to ${value.toFixed(1)}`);

    // Optionally refresh the article data to get updated server state
    // await this.refreshSingleArticle(articleId);
  }

  async loadArticles() {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoadingState();

    try {
      const params = new URLSearchParams({
        page: this.currentPage,
        limit: this.articlesPerPage,
        search: this.currentFilters.search,
        bias: this.currentFilters.bias,
        date: this.currentFilters.date,
        sort: this.currentFilters.sort
      });

      const response = await fetch(`/api/articles?${params}`);

      if (!response.ok) {
        throw new Error(`Failed to load articles: ${response.statusText}`);
      }

      const data = await response.json();
      this.renderArticles(data.articles || []);
      this.updatePagination(data.pagination || {});

    } catch (error) {
      console.error('Error loading articles:', error);
      this.showErrorState(error.message);
    } finally {
      this.isLoading = false;
    }
  }
  renderArticles(articles) {
    const container = document.getElementById('articles-container');
    if (!container) return;

    if (articles.length === 0) {
      container.innerHTML = `
        <div class="no-articles col-span-full">
          <p class="text-center text-secondary">No articles found matching your criteria.</p>
        </div>
      `;
      return;
    }

    // Clear container and create ArticleCard components
    container.innerHTML = '';

    articles.forEach(article => {
      const articleCard = document.createElement('article-card');

      // Set article data
      articleCard.article = article;

      // Set component properties based on current view
      articleCard.compact = this.currentView === 'list';
      articleCard.showBiasSlider = true;
      articleCard.clickable = true;

      container.appendChild(articleCard);
    });

    // Set up event listeners for the new components
    this.setupArticleCardEvents();
  }

  setupArticleCardEvents() {
    const container = document.getElementById('articles-container');
    if (!container) return;

    // Listen for article card events
    container.addEventListener('articleclick', (event) => {
      const { article } = event.detail;
      window.location.href = `/article/${article.id}`;
    });

    container.addEventListener('articleaction', (event) => {
      const { action, article, target } = event.detail;

      switch (action) {
        case 'read-more':
          window.location.href = `/article/${article.id}`;
          break;
        case 'original-source':
          window.open(article.url, '_blank', 'noopener,noreferrer');
          break;
      }
    });

    container.addEventListener('biaschange', (event) => {
      this.handleBiasChange(event.detail);
    });
  }
  switchView(view) {
    this.currentView = view;

    const container = document.getElementById('articles-container');
    const buttons = document.querySelectorAll('.view-toggle-btn');

    // Update button states
    buttons.forEach(btn => {
      const isActive = btn.dataset.view === view;
      btn.classList.toggle('view-toggle-btn--active', isActive);
      btn.setAttribute('aria-pressed', isActive);
    });

    // Update container classes
    if (container) {
      container.className = view === 'list'
        ? 'articles-list space-y-4'
        : 'articles-grid grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6';

      // Update all ArticleCard components to match the new view
      const articleCards = container.querySelectorAll('article-card');
      articleCards.forEach(card => {
        card.compact = view === 'list';
      });
    }
  }

  async refreshArticles() {
    this.currentPage = 1;
    await this.loadArticles();
  }

  showLoadingState() {
    const container = document.getElementById('articles-container');
    if (container) {
      container.innerHTML = `
        <div class="loading-placeholder col-span-full" aria-live="polite">
          <div class="loading-spinner"></div>
          <p>Loading articles...</p>
        </div>
      `;
    }
  }

  showErrorState(message) {
    const container = document.getElementById('articles-container');
    if (container) {
      container.innerHTML = `
        <div class="error-state col-span-full">
          <p class="text-center text-error">
            Error loading articles: ${this.escapeHtml(message)}
          </p>
          <button onclick="window.articlesPage.refreshArticles()"
                  class="btn btn--primary mx-auto mt-4">
            Retry
          </button>
        </div>
      `;
    }
  }

  updatePagination(pagination) {
    // TODO: Implement pagination controls if needed
    console.log('Pagination:', pagination);
  }

  showSuccessToast(message) {
    this.showToast(message, 'success');
  }

  showErrorToast(message) {
    this.showToast(message, 'error');
  }

  showToast(message, type = 'info') {
    // Simple toast implementation
    const toast = document.createElement('div');
    toast.className = `toast toast--${type}`;
    toast.textContent = message;
    toast.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      padding: 12px 20px;
      border-radius: 4px;
      color: white;
      z-index: 1000;
      font-size: 14px;
      max-width: 300px;
      opacity: 0;
      transform: translateY(-20px);
      transition: all 0.3s ease;
    `;

    // Set background color based on type
    switch (type) {
      case 'success':
        toast.style.backgroundColor = '#10b981';
        break;
      case 'error':
        toast.style.backgroundColor = '#ef4444';
        break;
      default:
        toast.style.backgroundColor = '#3b82f6';
    }

    document.body.appendChild(toast);

    // Animate in
    requestAnimationFrame(() => {
      toast.style.opacity = '1';
      toast.style.transform = 'translateY(0)';
    });

    // Remove after 4 seconds
    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(-20px)';
      setTimeout(() => {
        if (toast.parentNode) {
          toast.parentNode.removeChild(toast);
        }
      }, 300);
    }, 4000);
  }

  // Utility functions
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  truncateText(text, maxLength) {
    if (text.length <= maxLength) return text;
    return text.slice(0, maxLength).trim() + '...';
  }
}

// Initialize the articles page when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  window.articlesPage = new ArticlesPage();
});

// Export for testing
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ArticlesPage;
}

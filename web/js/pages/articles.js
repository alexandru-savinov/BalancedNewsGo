/**
 * Articles Page JavaScript
 * Handles article listing, filtering, and BiasSlider integration
 */

class ArticlesPage {
  constructor() {
    this.currentView = 'grid';
    this.currentFilters = {
      search: '',
      source: 'all',
      leaning: 'all',
      dateFrom: '',
      dateTo: '',
      sortBy: 'date',
      sortOrder: 'desc'
    };
    this.currentPage = 1;
    this.articlesPerPage = 20;
    this.isLoading = false;
    this.totalArticles = 0;
    this.availableSources = [];

    // Initialize API client
    this.apiClient = new ApiClient();

    this.init();
  }

  async init() {
    this.loadStateFromURL();
    this.bindEventListeners();
    await this.loadArticles();
    this.setupBiasSliderEvents();
    this.updatePageTitle();
    // setupArticleCardEvents is called after renderArticles
  }
  bindEventListeners() {
    // Search input
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
      searchInput.addEventListener('input', this.debounce(() => {
        this.currentFilters.search = searchInput.value;
        this.currentPage = 1;
        this.refreshArticles();
      }, 300));
    }

    // Filter selects
    const sourceFilter = document.getElementById('source-filter');
    if (sourceFilter) {
      sourceFilter.addEventListener('change', (e) => {
        this.currentFilters.source = e.target.value;
        this.currentPage = 1;
        this.refreshArticles();
      });
    }

    const leaningFilter = document.getElementById('bias-filter');
    if (leaningFilter) {
      leaningFilter.addEventListener('change', (e) => {
        this.currentFilters.leaning = e.target.value;
        this.currentPage = 1;
        this.refreshArticles();
      });
    }

    const dateFilter = document.getElementById('date-filter');
    if (dateFilter) {
      dateFilter.addEventListener('change', (e) => {
        this.handleDateFilterChange(e.target.value);
      });
    }

    const sortSelect = document.getElementById('sort-select');
    if (sortSelect) {
      sortSelect.addEventListener('change', (e) => {
        this.handleSortChange(e.target.value);
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

    // Pagination event listeners will be added after pagination controls are created

    // Browser back/forward navigation
    window.addEventListener('popstate', (e) => {
      if (e.state) {
        this.loadStateFromURL();
        this.loadArticles();
      }
    });
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

    try {
      // Update bias score via API
      await this.apiClient.put(`/articles/${articleId}/bias`, {
        score: value,
        source: 'manual',
        confidence: 0.9
      });

      this.showSuccessToast(`Bias updated to ${value.toFixed(1)}`);
    } catch (error) {
      console.error('Failed to update bias:', error);
      this.showErrorToast(`Failed to update bias: ${error.message}`);
    }
  }

  // URL State Management
  loadStateFromURL() {
    const params = new URLSearchParams(window.location.search);

    this.currentFilters.search = params.get('search') || '';
    this.currentFilters.source = params.get('source') || 'all';
    this.currentFilters.leaning = params.get('leaning') || 'all';
    this.currentFilters.dateFrom = params.get('dateFrom') || '';
    this.currentFilters.dateTo = params.get('dateTo') || '';
    this.currentFilters.sortBy = params.get('sortBy') || 'date';
    this.currentFilters.sortOrder = params.get('sortOrder') || 'desc';
    this.currentPage = parseInt(params.get('page')) || 1;
    this.currentView = params.get('view') || 'grid';

    this.updateFormControls();
  }

  updateURL() {
    const params = new URLSearchParams();

    if (this.currentFilters.search) params.set('search', this.currentFilters.search);
    if (this.currentFilters.source !== 'all') params.set('source', this.currentFilters.source);
    if (this.currentFilters.leaning !== 'all') params.set('leaning', this.currentFilters.leaning);
    if (this.currentFilters.dateFrom) params.set('dateFrom', this.currentFilters.dateFrom);
    if (this.currentFilters.dateTo) params.set('dateTo', this.currentFilters.dateTo);
    if (this.currentFilters.sortBy !== 'date') params.set('sortBy', this.currentFilters.sortBy);
    if (this.currentFilters.sortOrder !== 'desc') params.set('sortOrder', this.currentFilters.sortOrder);
    if (this.currentPage > 1) params.set('page', this.currentPage.toString());
    if (this.currentView !== 'grid') params.set('view', this.currentView);

    const newURL = `${window.location.pathname}${params.toString() ? '?' + params.toString() : ''}`;
    window.history.replaceState(
      {
        filters: this.currentFilters,
        page: this.currentPage,
        view: this.currentView
      },
      '',
      newURL
    );
  }

  updateFormControls() {
    // Update form inputs to match current state
    const searchInput = document.getElementById('search-input');
    if (searchInput) searchInput.value = this.currentFilters.search;

    const sourceFilter = document.getElementById('source-filter');
    if (sourceFilter) sourceFilter.value = this.currentFilters.source;

    const leaningFilter = document.getElementById('bias-filter');
    if (leaningFilter) leaningFilter.value = this.currentFilters.leaning;

    const sortSelect = document.getElementById('sort-select');
    if (sortSelect) {
      const sortValue = `${this.currentFilters.sortBy}-${this.currentFilters.sortOrder}`;
      sortSelect.value = sortValue;
    }

    // Update view toggle
    const viewButtons = document.querySelectorAll('.view-toggle-btn');
    viewButtons.forEach(btn => {
      const isActive = btn.dataset.view === this.currentView;
      btn.classList.toggle('view-toggle-btn--active', isActive);
      btn.setAttribute('aria-pressed', isActive);
    });
  }

  // Filter and sort handlers
  handleDateFilterChange(value) {
    const now = new Date();
    let dateFrom = '';

    switch (value) {
      case 'today':
        dateFrom = now.toISOString().split('T')[0];
        break;
      case 'week':
        const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
        dateFrom = weekAgo.toISOString().split('T')[0];
        break;
      case 'month':
        const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
        dateFrom = monthAgo.toISOString().split('T')[0];
        break;
      case 'all':
      default:
        dateFrom = '';
        break;
    }

    this.currentFilters.dateFrom = dateFrom;
    this.currentFilters.dateTo = value === 'all' ? '' : now.toISOString().split('T')[0];
    this.currentPage = 1;
    this.refreshArticles();
  }

  handleSortChange(value) {
    const [sortBy, sortOrder] = value.split('-');
    this.currentFilters.sortBy = sortBy;
    this.currentFilters.sortOrder = sortOrder;
    this.currentPage = 1;
    this.refreshArticles();
  }
  async loadArticles() {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoadingState();

    try {
      // Build API query parameters
      const queryParams = {
        limit: this.articlesPerPage,
        offset: (this.currentPage - 1) * this.articlesPerPage,
        sortBy: this.currentFilters.sortBy,
        sortOrder: this.currentFilters.sortOrder
      };

      // Add optional filters
      if (this.currentFilters.search) {
        queryParams.search = this.currentFilters.search;
      }
      if (this.currentFilters.source && this.currentFilters.source !== 'all') {
        queryParams.source = [this.currentFilters.source];
      }
      if (this.currentFilters.leaning && this.currentFilters.leaning !== 'all') {
        queryParams.leaning = this.currentFilters.leaning;
      }
      if (this.currentFilters.dateFrom) {
        queryParams.dateFrom = this.currentFilters.dateFrom;
      }
      if (this.currentFilters.dateTo) {
        queryParams.dateTo = this.currentFilters.dateTo;
      }

      // Make API request using ApiClient
      const response = await this.apiClient.get('/articles', { params: queryParams });

      if (response.success) {
        const data = response.data;
        this.renderArticles(data.articles || []);
        this.updatePagination(data.pagination || {});
        this.updateAvailableFilters(data.filters || {});
        this.totalArticles = data.pagination?.total || 0;

        // Update URL to reflect current state
        this.updateURL();
        this.updatePageTitle();
      } else {
        throw new Error(response.error?.message || 'Failed to load articles');
      }

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

    // Also update the pagination to show loading state
    const paginationContainer = document.getElementById('pagination-container');
    if (paginationContainer) {
      paginationContainer.innerHTML = '';
    }
  }

  showErrorState(message) {
    const container = document.getElementById('articles-container');
    if (container) {
      container.innerHTML = `
        <div class="error-state col-span-full">
          <div class="error-icon">⚠️</div>
          <h3 class="error-title">Unable to Load Articles</h3>
          <p class="error-message">
            ${this.escapeHtml(message)}
          </p>
          <button onclick="window.articlesPage.refreshArticles()"
                  class="btn btn--primary retry-btn">
            <span class="btn-icon">🔄</span>
            Try Again
          </button>
        </div>
      `;
    }

    // Clear pagination on error
    const paginationContainer = document.getElementById('pagination-container');
    if (paginationContainer) {
      paginationContainer.innerHTML = '';
    }
  }
  updatePagination(pagination) {
    const paginationContainer = document.getElementById('pagination-container') || this.createPaginationContainer();

    if (!pagination || !pagination.total) {
      paginationContainer.innerHTML = '';
      return;
    }

    const totalPages = Math.ceil(pagination.total / this.articlesPerPage);
    const currentPage = this.currentPage;

    if (totalPages <= 1) {
      paginationContainer.innerHTML = '';
      return;
    }

    let paginationHTML = '<nav class="pagination" aria-label="Article pagination">';
    paginationHTML += '<ul class="pagination-list">';

    // Previous page
    if (pagination.hasPrev) {
      paginationHTML += `
        <li>
          <button type="button" class="pagination-btn pagination-btn--prev" data-page="${currentPage - 1}">
            <span aria-hidden="true">&laquo;</span>
            <span class="sr-only">Previous page</span>
          </button>
        </li>
      `;
    }

    // Page numbers
    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, currentPage + 2);

    if (startPage > 1) {
      paginationHTML += `<li><button type="button" class="pagination-btn" data-page="1">1</button></li>`;
      if (startPage > 2) {
        paginationHTML += '<li><span class="pagination-ellipsis">...</span></li>';
      }
    }

    for (let page = startPage; page <= endPage; page++) {
      const isActive = page === currentPage;
      paginationHTML += `
        <li>
          <button type="button"
                  class="pagination-btn ${isActive ? 'pagination-btn--active' : ''}"
                  data-page="${page}"
                  ${isActive ? 'aria-current="page"' : ''}>
            ${page}
          </button>
        </li>
      `;
    }

    if (endPage < totalPages) {
      if (endPage < totalPages - 1) {
        paginationHTML += '<li><span class="pagination-ellipsis">...</span></li>';
      }
      paginationHTML += `<li><button type="button" class="pagination-btn" data-page="${totalPages}">${totalPages}</button></li>`;
    }

    // Next page
    if (pagination.hasNext) {
      paginationHTML += `
        <li>
          <button type="button" class="pagination-btn pagination-btn--next" data-page="${currentPage + 1}">
            <span aria-hidden="true">&raquo;</span>
            <span class="sr-only">Next page</span>
          </button>
        </li>
      `;
    }

    paginationHTML += '</ul>';

    // Page info
    const startItem = (currentPage - 1) * this.articlesPerPage + 1;
    const endItem = Math.min(currentPage * this.articlesPerPage, pagination.total);
    paginationHTML += `
      <div class="pagination-info">
        Showing ${startItem}-${endItem} of ${pagination.total} articles
      </div>
    `;

    paginationHTML += '</nav>';

    paginationContainer.innerHTML = paginationHTML;
    this.bindPaginationEvents();
  }

  createPaginationContainer() {
    const container = document.createElement('div');
    container.id = 'pagination-container';
    container.className = 'pagination-wrapper mt-8';

    const articlesSection = document.querySelector('.articles-section');
    if (articlesSection) {
      articlesSection.appendChild(container);
    }

    return container;
  }

  bindPaginationEvents() {
    const paginationContainer = document.getElementById('pagination-container');
    if (!paginationContainer) return;

    paginationContainer.addEventListener('click', (e) => {
      if (e.target.classList.contains('pagination-btn') && e.target.dataset.page) {
        e.preventDefault();
        const page = parseInt(e.target.dataset.page);
        if (page !== this.currentPage) {
          this.currentPage = page;
          this.loadArticles();
          this.scrollToTop();
        }
      }
    });
  }

  updateAvailableFilters(filters) {
    if (filters.availableSources) {
      this.availableSources = filters.availableSources;
      this.updateSourceFilterOptions();
    }
  }

  updateSourceFilterOptions() {
    const sourceFilter = document.getElementById('source-filter');
    if (!sourceFilter || !this.availableSources.length) return;

    // Preserve current selection
    const currentValue = sourceFilter.value;

    // Clear existing options except "All Sources"
    sourceFilter.innerHTML = '<option value="all">All Sources</option>';

    // Add available sources
    this.availableSources.forEach(source => {
      const option = document.createElement('option');
      option.value = source;
      option.textContent = source;
      sourceFilter.appendChild(option);
    });

    // Restore selection if still valid
    if (currentValue && (currentValue === 'all' || this.availableSources.includes(currentValue))) {
      sourceFilter.value = currentValue;
    }
  }

  updatePageTitle() {
    let title = 'Articles - NewsBalancer';

    if (this.currentFilters.search) {
      title = `"${this.currentFilters.search}" - Articles - NewsBalancer`;
    } else if (this.currentFilters.source !== 'all') {
      title = `${this.currentFilters.source} - Articles - NewsBalancer`;
    } else if (this.currentFilters.leaning !== 'all') {
      const leaningNames = {
        'left': 'Left-leaning',
        'center': 'Center',
        'right': 'Right-leaning'
      };
      title = `${leaningNames[this.currentFilters.leaning]} Articles - NewsBalancer`;
    }

    if (this.currentPage > 1) {
      title = `Page ${this.currentPage} - ${title}`;
    }

    document.title = title;
  }

  scrollToTop() {
    window.scrollTo({ top: 0, behavior: 'smooth' });
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

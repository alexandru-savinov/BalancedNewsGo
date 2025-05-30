/**
 * Article Detail Page JavaScript
 * Handles individual article display, bias analysis, and real-time updates
 *
 * Features:
 * - Article content display with metadata
 * - Interactive bias slider with real-time updates
 * - Manual scoring interface (admin feature)
 * - Enhanced SSE-powered real-time analysis progress
 * - Individual model scores breakdown
 * - User feedback submission form
 * - Re-analysis trigger button
 * - Related articles section
 */

import { trackProgress } from '../utils/SSEClient.js';

class ArticleDetailPage {
  constructor() {
    this.articleId = this.getArticleIdFromURL();
    this.article = null;
    this.isAdmin = false; // Check user permissions
    this.analysisInProgress = false;
    this.progressIndicator = null;
    this.biasSlider = null;
    this.feedbackForm = null;
    
    // Initialize API client
    this.apiClient = new ApiClient();
    
    this.init();
  }

  async init() {
    try {
      this.showLoadingState();
      await this.loadArticle();
      await this.loadDOMPurify(); // Load DOMPurify dynamically
      this.setupComponents();
      this.bindEventListeners();
      this.checkUserPermissions();
      this.updatePageTitle();
      this.hideLoadingState();
    } catch (error) {
      console.error('Failed to initialize article detail page:', error);
      this.showErrorState(error);
    }
  }

  async loadDOMPurify() {
    // Dynamic import of DOMPurify for content sanitization
    if (!window.DOMPurify) {
      try {
        const DOMPurifyModule = await import('https://cdn.jsdelivr.net/npm/dompurify@3.0.5/dist/purify.es.js');
        window.DOMPurify = DOMPurifyModule.default || DOMPurifyModule;
      } catch (error) {
        console.warn('Failed to load DOMPurify, using fallback HTML escaping:', error);
        // Continue without DOMPurify - will use escapeHtml fallback
      }
    }
  }

  getArticleIdFromURL() {
    const pathParts = window.location.pathname.split('/');
    return pathParts[pathParts.length - 1] || pathParts[pathParts.length - 2];
  }

  async loadArticle() {
    try {
      const response = await this.apiClient.get(`/api/articles/${this.articleId}`);
      this.article = response.data;
      this.renderArticle();
      await this.loadRelatedArticles();
    } catch (error) {
      if (error.status === 404) {
        throw new Error('Article not found');
      }
      throw error;
    }
  }

  renderArticle() {
    if (!this.article) return;

    // Update article metadata
    this.renderArticleHeader();
    this.renderArticleContent();
    this.renderBiasAnalysis();
    this.renderModelScores();
  }

  renderArticleHeader() {
    const header = document.querySelector('.article-header');
    if (!header) return;

    const publishedDate = new Date(this.article.publishedAt).toLocaleDateString();
    const readingTime = Math.ceil(this.article.metadata.wordCount / 200); // 200 WPM

    header.innerHTML = `
      <div class="article-meta">
        <h1 class="article-title">${this.escapeHtml(this.article.title)}</h1>
        <div class="article-info">
          <span class="article-source">${this.escapeHtml(this.article.source)}</span>
          <span class="article-date">${publishedDate}</span>
          <span class="article-reading-time">${readingTime} min read</span>
          <span class="article-word-count">${this.article.metadata.wordCount} words</span>
        </div>
        <div class="article-actions">
          <a href="${this.escapeHtml(this.article.url)}" 
             target="_blank" 
             rel="noopener noreferrer"
             class="btn btn-outline">
            Read Original
            <svg class="icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path d="M7 17L17 7M17 7H7M17 7V17"/>
            </svg>
          </a>
          ${this.isAdmin ? this.renderAdminActions() : ''}
        </div>
      </div>
    `;
  }

  renderArticleContent() {
    const contentContainer = document.querySelector('.article-content');
    if (!contentContainer || !this.article.content) return;

    // Sanitize content using DOMPurify if available
    const sanitizedContent = window.DOMPurify ? 
      window.DOMPurify.sanitize(this.article.content) : 
      this.escapeHtml(this.article.content);

    contentContainer.innerHTML = `
      <div class="article-summary">
        <h2>Summary</h2>
        <p>${this.escapeHtml(this.article.summary)}</p>
      </div>
      <div class="article-body">
        <h2>Full Article</h2>
        <div class="article-text">${sanitizedContent}</div>
      </div>
    `;
  }

  renderBiasAnalysis() {
    const biasSection = document.querySelector('.bias-analysis-section');
    if (!biasSection) return;

    const confidence = this.article.bias.confidence * 100;
    const biasLabel = this.getBiasLabel(this.article.bias.score);

    biasSection.innerHTML = `
      <div class="bias-analysis-header">
        <h2>Bias Analysis</h2>
        <div class="bias-confidence">
          <span class="confidence-label">Confidence:</span>
          <span class="confidence-value">${confidence.toFixed(1)}%</span>
        </div>
      </div>
      
      <div class="main-bias-slider">
        <label for="article-bias-slider" class="bias-slider-label">
          Overall Bias Score: <strong>${biasLabel}</strong>
        </label>
        <bias-slider 
          id="article-bias-slider"
          value="${this.article.bias.score}"
          article-id="${this.articleId}"
          size="large"
          ${this.isAdmin ? '' : 'readonly'}
        ></bias-slider>
      </div>

      ${this.article.analysis ? this.renderBiasBreakdown() : ''}
      
      ${this.isAdmin ? this.renderManualScoringControls() : ''}
    `;
  }

  renderBiasBreakdown() {
    if (!this.article.analysis?.biasBreakdown) return '';

    const { political, emotional } = this.article.analysis.biasBreakdown;

    return `
      <div class="bias-breakdown">
        <h3>Bias Breakdown</h3>
        <div class="breakdown-item">
          <label>Political Bias</label>
          <div class="breakdown-bar">
            <div class="breakdown-fill" style="width: ${Math.abs(political) * 50}%; background-color: ${political < 0 ? 'var(--color-bias-left)' : 'var(--color-bias-right)'}"></div>
          </div>
          <span class="breakdown-value">${political.toFixed(2)}</span>
        </div>
        <div class="breakdown-item">
          <label>Emotional Language</label>
          <div class="breakdown-bar">
            <div class="breakdown-fill" style="width: ${Math.abs(emotional) * 50}%; background-color: var(--color-warning-500)"></div>
          </div>
          <span class="breakdown-value">${emotional.toFixed(2)}</span>
        </div>
      </div>
    `;
  }

  renderModelScores() {
    const modelSection = document.querySelector('.model-scores-section');
    if (!modelSection || !this.article.bias.modelScores) return;

    const modelScoresHtml = this.article.bias.modelScores.map(model => `
      <div class="model-score-item">
        <div class="model-info">
          <span class="model-name">${this.escapeHtml(model.modelName)}</span>
          <span class="model-timestamp">${new Date(model.timestamp).toLocaleString()}</span>
        </div>
        <div class="model-score">
          <span class="score-value">${model.score.toFixed(3)}</span>
          <span class="confidence-value">${(model.confidence * 100).toFixed(1)}%</span>
        </div>
        <div class="model-bias-indicator">
          <div class="bias-bar">
            <div class="bias-fill" 
                 style="left: ${(model.score + 1) * 50}%; 
                        background-color: ${this.getBiasColor(model.score)}">
            </div>
          </div>
        </div>
      </div>
    `).join('');

    modelSection.innerHTML = `
      <h2>Individual Model Scores</h2>
      <div class="model-scores-grid">
        ${modelScoresHtml}
      </div>
      <div class="analysis-info">
        <p>Analysis Version: ${this.article.analysis?.analysisVersion || 'Unknown'}</p>
      </div>
    `;
  }

  renderAdminActions() {
    return `
      <button id="reanalyze-btn" class="btn btn-primary">
        Re-analyze Article
      </button>
      <button id="manual-score-btn" class="btn btn-outline">
        Manual Score
      </button>
    `;
  }

  renderManualScoringControls() {
    return `
      <div class="manual-scoring" style="display: none;">
        <h3>Manual Bias Scoring</h3>
        <form id="manual-score-form">
          <div class="form-group">
            <label for="manual-score">Bias Score (-1 to 1)</label>
            <input type="number" 
                   id="manual-score" 
                   min="-1" 
                   max="1" 
                   step="0.01" 
                   value="${this.article.bias.score}"
                   required>
          </div>
          <div class="form-group">
            <label for="manual-confidence">Confidence (0 to 1)</label>
            <input type="number" 
                   id="manual-confidence" 
                   min="0" 
                   max="1" 
                   step="0.01" 
                   value="${this.article.bias.confidence}"
                   required>
          </div>
          <div class="form-group">
            <label for="scoring-notes">Notes (optional)</label>
            <textarea id="scoring-notes" 
                      placeholder="Explanation for manual scoring..."></textarea>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">Update Score</button>
            <button type="button" id="cancel-manual-score" class="btn btn-outline">Cancel</button>
          </div>
        </form>
      </div>
    `;
  }

  setupComponents() {
    // Initialize bias slider
    this.biasSlider = document.querySelector('#article-bias-slider');
    if (this.biasSlider) {
      this.biasSlider.addEventListener('biaschange', (event) => {
        this.handleBiasChange(event.detail);
      });
    }

    // Initialize progress indicator for re-analysis
    this.setupProgressIndicator();

    // Initialize feedback form
    this.setupFeedbackForm();
  }

  setupProgressIndicator() {
    const progressContainer = document.querySelector('.progress-container');
    if (progressContainer) {
      progressContainer.innerHTML = `
        <progress-indicator 
          id="analysis-progress"
          style="display: none;">
        </progress-indicator>
      `;
      
      this.progressIndicator = document.querySelector('#analysis-progress');
    }
  }

  setupFeedbackForm() {
    const feedbackSection = document.querySelector('.feedback-section');
    if (!feedbackSection) return;

    feedbackSection.innerHTML = `
      <h2>Feedback</h2>
      <form id="feedback-form">
        <div class="form-group">
          <label for="feedback-type">Feedback Type</label>
          <select id="feedback-type" required>
            <option value="">Select feedback type</option>
            <option value="bias-incorrect">Bias score incorrect</option>
            <option value="content-issue">Content issue</option>
            <option value="technical-problem">Technical problem</option>
            <option value="other">Other</option>
          </select>
        </div>
        <div class="form-group">
          <label for="feedback-message">Message</label>
          <textarea id="feedback-message" 
                    required 
                    placeholder="Please describe your feedback..."></textarea>
        </div>
        <button type="submit" class="btn btn-primary">Submit Feedback</button>
      </form>
    `;

    this.feedbackForm = document.querySelector('#feedback-form');
  }

  bindEventListeners() {
    // Re-analysis button
    const reanalyzeBtn = document.querySelector('#reanalyze-btn');
    if (reanalyzeBtn) {
      reanalyzeBtn.addEventListener('click', () => this.triggerReanalysis());
    }

    // Manual scoring toggle
    const manualScoreBtn = document.querySelector('#manual-score-btn');
    if (manualScoreBtn) {
      manualScoreBtn.addEventListener('click', () => this.toggleManualScoring());
    }

    // Manual scoring form
    const manualScoreForm = document.querySelector('#manual-score-form');
    if (manualScoreForm) {
      manualScoreForm.addEventListener('submit', (e) => this.handleManualScore(e));
    }

    // Cancel manual scoring
    const cancelBtn = document.querySelector('#cancel-manual-score');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.toggleManualScoring(false));
    }

    // Feedback form
    if (this.feedbackForm) {
      this.feedbackForm.addEventListener('submit', (e) => this.handleFeedbackSubmit(e));
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => this.handleKeyboardShortcuts(e));
  }

  async handleBiasChange(detail) {
    try {
      await this.apiClient.post(`/api/articles/${this.articleId}/bias`, {
        score: detail.value,
        source: 'manual',
        confidence: detail.confidence || 0.8
      });

      this.showSuccessMessage('Bias score updated successfully');
    } catch (error) {
      console.error('Failed to update bias score:', error);
      this.showErrorMessage('Failed to update bias score');
      
      // Revert the slider
      if (this.biasSlider) {
        this.biasSlider.value = this.article.bias.score;
      }
    }
  }

  async triggerReanalysis() {
    if (this.analysisInProgress) return;

    try {
      this.analysisInProgress = true;
      this.showProgressIndicator();

      const response = await this.apiClient.post(`/api/llm/analyze/${this.articleId}`, {
        priority: 'high',
        options: {
          forceReanalyze: true,
          updateExisting: true
        }
      });

      // Start listening for progress updates
      this.startProgressTracking(response.data.requestId);

    } catch (error) {
      console.error('Failed to trigger re-analysis:', error);
      this.showErrorMessage('Failed to trigger re-analysis');
      this.analysisInProgress = false;
      this.hideProgressIndicator();
    }
  }
  startProgressTracking(requestId) {
    if (!this.progressIndicator) return;

    // Use the enhanced SSE client for progress tracking
    this.progressTracker = trackProgress(requestId, {
      onConnect: () => {
        console.log('Progress tracking connected');
      },
      
      onProgress: (progress) => {
        this.updateProgressIndicator(progress);
      },
      
      onComplete: (result) => {
        this.progressTracker.stop();
        this.onAnalysisComplete();
      },
      
      onError: (error) => {
        console.error('Progress tracking error:', error);
        this.progressTracker.stop();
        this.onAnalysisError('Connection lost');
      }
    });
  }

  async onAnalysisComplete() {
    this.analysisInProgress = false;
    this.hideProgressIndicator();
    this.showSuccessMessage('Article re-analysis completed');
    
    // Reload article data
    await this.loadArticle();
  }

  onAnalysisError(error) {
    this.analysisInProgress = false;
    this.hideProgressIndicator();
    this.showErrorMessage(`Analysis failed: ${error}`);
  }

  toggleManualScoring(show = null) {
    const manualScoring = document.querySelector('.manual-scoring');
    if (!manualScoring) return;

    const isVisible = manualScoring.style.display !== 'none';
    const shouldShow = show !== null ? show : !isVisible;

    manualScoring.style.display = shouldShow ? 'block' : 'none';
    
    if (shouldShow) {
      const scoreInput = document.querySelector('#manual-score');
      if (scoreInput) scoreInput.focus();
    }
  }

  async handleManualScore(event) {
    event.preventDefault();

    const formData = new FormData(event.target);
    const score = parseFloat(formData.get('manual-score'));
    const confidence = parseFloat(formData.get('manual-confidence'));
    const notes = formData.get('scoring-notes');

    try {
      await this.apiClient.post(`/api/articles/${this.articleId}/bias`, {
        score,
        confidence,
        source: 'manual',
        notes
      });

      this.article.bias.score = score;
      this.article.bias.confidence = confidence;
      
      this.renderBiasAnalysis();
      this.toggleManualScoring(false);
      this.showSuccessMessage('Manual score updated successfully');

    } catch (error) {
      console.error('Failed to update manual score:', error);
      this.showErrorMessage('Failed to update manual score');
    }
  }

  async handleFeedbackSubmit(event) {
    event.preventDefault();

    const formData = new FormData(event.target);
    const feedbackData = {
      articleId: this.articleId,
      type: formData.get('feedback-type'),
      message: formData.get('feedback-message')
    };

    try {
      await this.apiClient.post('/api/feedback', feedbackData);
      this.showSuccessMessage('Feedback submitted successfully');
      event.target.reset();

    } catch (error) {
      console.error('Failed to submit feedback:', error);
      this.showErrorMessage('Failed to submit feedback');
    }
  }

  async loadRelatedArticles() {
    try {
      const response = await this.apiClient.get(`/api/articles/${this.articleId}/related`);
      this.renderRelatedArticles(response.data);
    } catch (error) {
      console.warn('Failed to load related articles:', error);
    }
  }

  renderRelatedArticles(articles) {
    const relatedSection = document.querySelector('.related-articles');
    if (!relatedSection || !articles.length) return;

    const articlesHtml = articles.map(article => `
      <article-card 
        article-id="${article.id}"
        compact="true"
        clickable="true">
      </article-card>
    `).join('');

    relatedSection.innerHTML = `
      <h2>Related Articles</h2>
      <div class="related-articles-grid">
        ${articlesHtml}
      </div>
    `;

    // Initialize article cards with data
    const cardElements = relatedSection.querySelectorAll('article-card');
    cardElements.forEach((card, index) => {
      card.article = articles[index];
    });
  }

  handleKeyboardShortcuts(event) {
    if (event.ctrlKey || event.metaKey) {
      switch (event.key) {
        case 'r':
          if (this.isAdmin) {
            event.preventDefault();
            this.triggerReanalysis();
          }
          break;
        case 'm':
          if (this.isAdmin) {
            event.preventDefault();
            this.toggleManualScoring();
          }
          break;
      }
    }
  }

  checkUserPermissions() {
    // Check if user has admin permissions
    // This would typically check authentication/authorization
    this.isAdmin = window.location.pathname.includes('/admin') || 
                  document.body.classList.contains('admin-mode');
  }

  updatePageTitle() {
    if (this.article) {
      document.title = `${this.article.title} - NewsBalancer`;
    }
  }

  // UI Helper Methods
  showLoadingState() {
    const mainContent = document.querySelector('#main-content');
    if (mainContent) {
      mainContent.innerHTML = `
        <div class="loading-state">
          <div class="loading-spinner"></div>
          <p>Loading article...</p>
        </div>
      `;
    }
  }

  hideLoadingState() {
    const loadingState = document.querySelector('.loading-state');
    if (loadingState) {
      loadingState.remove();
    }
  }

  showErrorState(error) {
    const mainContent = document.querySelector('#main-content');
    if (mainContent) {
      mainContent.innerHTML = `
        <div class="error-state">
          <h1>Error Loading Article</h1>
          <p>${this.escapeHtml(error.message)}</p>
          <button onclick="window.location.reload()" class="btn btn-primary">
            Try Again
          </button>
          <a href="/articles" class="btn btn-outline">
            Back to Articles
          </a>
        </div>
      `;
    }
  }

  showProgressIndicator() {
    if (this.progressIndicator) {
      this.progressIndicator.style.display = 'block';
    }
  }

  hideProgressIndicator() {
    if (this.progressIndicator) {
      this.progressIndicator.style.display = 'none';
    }
  }

  updateProgressIndicator(progress) {
    if (this.progressIndicator) {
      this.progressIndicator.setAttribute('progress', progress.progress);
      this.progressIndicator.setAttribute('status', progress.status);
      this.progressIndicator.setAttribute('stage', progress.stage);
    }
  }

  showSuccessMessage(message) {
    this.showNotification(message, 'success');
  }

  showErrorMessage(message) {
    this.showNotification(message, 'error');
  }

  showNotification(message, type = 'info') {
    // Create or update notification
    let notification = document.querySelector('.notification');
    if (!notification) {
      notification = document.createElement('div');
      notification.className = 'notification';
      document.body.appendChild(notification);
    }

    notification.className = `notification notification--${type}`;
    notification.textContent = message;
    notification.style.display = 'block';

    // Auto-hide after 5 seconds
    setTimeout(() => {
      notification.style.display = 'none';
    }, 5000);
  }

  // Utility Methods
  getBiasLabel(score) {
    if (score < -0.3) return 'Left Leaning';
    if (score > 0.3) return 'Right Leaning';
    return 'Center';
  }
  getBiasColor(score) {
    if (score < -0.1) return 'var(--color-bias-left)';
    if (score > 0.1) return 'var(--color-bias-right)';
    return 'var(--color-bias-center)';
  }

  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  cleanup() {
    if (this.progressTracker) {
      this.progressTracker.stop();
      this.progressTracker = null;
    }
  }
}

// Initialize the page when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  window.articleDetailPage = new ArticleDetailPage();
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ArticleDetailPage;
}

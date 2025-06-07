/**
 * ProgressIndicator Web Component
 * Real-time progress tracking component with Server-Sent Events (SSE)
 *
 * Features:
 * - SSE-powered real-time progress updates using enhanced SSEClient
 * - Robust reconnection logic with exponential backoff
 * - Multiple states: idle, connecting, processing, completed, error
 * - Optional per-model progress breakdown
 * - Accessibility support with ARIA roles
 * - Responsive design with compact/full modes
 */

import { SSEClient } from '../utils/SSEClient.js';

class ProgressIndicator extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: 'open' });

    // Component state
    this.#sseClient = null;
    this.#progressValue = 0;
    this.#status = 'idle';
    this.#stage = '';
    this.#articleId = null;
    this.#reconnectAttempts = 0;
    this.#reconnectTimer = null;
    this.#maxReconnectAttempts = 5;
    this.#baseReconnectDelay = 1000; // 1 second
    this.#autoConnect = false;
    this.#showDetails = false;
    this.#eta = null;
    this.#modelProgress = null;

    this.#render();
  }

  static get observedAttributes() {
    return ['article-id', 'auto-connect', 'show-details'];
  }
  // Private properties
  #sseClient = null;
  #progressValue = 0;
  #status = 'idle';
  #stage = '';
  #articleId = null;
  #reconnectAttempts = 0;
  #reconnectTimer = null;
  #maxReconnectAttempts = 5;
  #baseReconnectDelay = 1000;
  #autoConnect = false;
  #showDetails = false;
  #eta = null;
  #modelProgress = null;

  // Getters and setters
  get articleId() {
    return this.#articleId;
  }

  set articleId(value) {
    this.#articleId = value;
    if (this.#autoConnect && value) {
      this.connect(value);
    }
  }

  get progress() {
    return this.#progressValue;
  }

  get status() {
    return this.#status;
  }

  get autoConnect() {
    return this.#autoConnect;
  }

  set autoConnect(value) {
    this.#autoConnect = value === true || value === 'true';
  }

  get showDetails() {
    return this.#showDetails;
  }

  set showDetails(value) {
    this.#showDetails = value === true || value === 'true';
    this.#updateDetailsVisibility();
  }

  // Lifecycle callbacks
  attributeChangedCallback(name, oldValue, newValue) {
    if (oldValue === newValue) return;

    switch (name) {
      case 'article-id':
        this.articleId = newValue;
        break;
      case 'auto-connect':
        this.autoConnect = newValue;
        break;
      case 'show-details':
        this.showDetails = newValue;
        break;
    }
  }

  connectedCallback() {
    // Set initial attributes from element attributes
    if (this.hasAttribute('article-id')) {
      this.articleId = this.getAttribute('article-id');
    }
    if (this.hasAttribute('auto-connect')) {
      this.autoConnect = this.getAttribute('auto-connect');
    }
    if (this.hasAttribute('show-details')) {
      this.showDetails = this.getAttribute('show-details');
    }
  }

  disconnectedCallback() {
    this.disconnect();
  }
  // Public methods
  async connect(articleId) {
    if (this.#sseClient && this.#sseClient.connected) {
      this.disconnect();
    }

    this.#articleId = articleId || this.#articleId;

    if (!this.#articleId) {
      console.warn('ProgressIndicator: No article ID provided for connection');
      return;
    }

    this.#updateStatus('connecting', 'Establishing connection...');

    try {
      this.#sseClient = new SSEClient({
        maxReconnectAttempts: this.#maxReconnectAttempts,
        reconnectDelay: this.#baseReconnectDelay
      });

      // Set up event listeners
      this.#sseClient.addEventListener('connected', (data) => {
        this.#updateStatus('processing', 'Connected - awaiting progress data...');
        this.#reconnectAttempts = 0;
      });

      this.#sseClient.addEventListener('message', (data) => {
        this.#handleProgressUpdate(data);
      });

      this.#sseClient.addEventListener('progress', (data) => {
        this.#handleProgressUpdate(data);
      });

      this.#sseClient.addEventListener('completed', (data) => {
        this.#handleProgressUpdate(data);
      });

      this.#sseClient.addEventListener('error', (data) => {
        console.error('ProgressIndicator SSE error:', data);
        this.#updateStatus('error', 'Connection error occurred');
      });

      this.#sseClient.addEventListener('disconnected', (data) => {
        if (data.reason !== 'Manual disconnect') {
          this.#updateStatus('error', 'Connection lost');
        }
      });

      this.#sseClient.addEventListener('reconnecting', (data) => {
        this.#reconnectAttempts = data.attempt;
        this.#updateStatus('connecting', `Reconnecting... (${data.attempt}/${data.maxAttempts})`);
      });

      this.#sseClient.addEventListener('failed', (data) => {
        this.#updateStatus('error', 'Connection failed - max retries reached');
      });

      // Connect to the progress endpoint
      const endpoint = `/api/llm/score-progress/${this.#articleId}`;
      this.#sseClient.connect(endpoint);

    } catch (error) {
      console.error('Failed to establish SSE connection:', error);
      this.#updateStatus('error', `Connection failed: ${error.message}`);
    }
  }

  disconnect() {
    if (this.#sseClient) {
      this.#sseClient.disconnect();
      this.#sseClient = null;
    }

    if (this.#reconnectTimer) {
      clearTimeout(this.#reconnectTimer);
      this.#reconnectTimer = null;
    }

    this.#reconnectAttempts = 0;
  }

  reset() {
    this.disconnect();
    this.#progressValue = 0;
    this.#status = 'idle';
    this.#stage = '';
    this.#eta = null;
    this.#modelProgress = null;
    this.#updateUI();
  }

  // Manual progress update (for non-SSE scenarios)
  updateProgress(progressData) {
    this.#progressValue = Math.max(0, Math.min(100, progressData.progress || 0));
    this.#status = progressData.status || this.#status;
    this.#stage = progressData.stage || this.#stage;
    this.#eta = progressData.eta || null;
    this.#modelProgress = progressData.modelProgress || null;

    this.#updateUI();
    this.#dispatchEvent('progressupdate', progressData);
  }

  // Private methods
  #render() {
    const template = document.createElement('template');
    template.innerHTML = `
      <style>
        :host {
          display: block;
          --progress-height: 8px;
          --progress-color: #3b82f6;
          --progress-bg: #e5e7eb;
          --progress-radius: 4px;
          --transition-duration: 300ms;
          --text-primary: #1f2937;
          --text-secondary: #6b7280;
          --color-success: #10b981;
          --color-error: #ef4444;
          --color-warning: #f59e0b;
        }

        .progress-container {
          width: 100%;
        }

        .progress-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 0.5rem;
          font-size: 0.875rem;
        }

        .progress-status {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          color: var(--text-primary);
          font-weight: 500;
        }

        .progress-percentage {
          color: var(--text-secondary);
          font-variant-numeric: tabular-nums;
        }

        .progress-track {
          position: relative;
          width: 100%;
          height: var(--progress-height);
          background-color: var(--progress-bg);
          border-radius: var(--progress-radius);
          overflow: hidden;
        }

        .progress-fill {
          height: 100%;
          background-color: var(--progress-color);
          border-radius: var(--progress-radius);
          transition: width var(--transition-duration) ease-out;
          width: 0%;
        }

        .progress-stage {
          margin-top: 0.25rem;
          font-size: 0.75rem;
          color: var(--text-secondary);
          min-height: 1rem;
        }

        .progress-eta {
          font-size: 0.75rem;
          color: var(--text-secondary);
          margin-top: 0.25rem;
        }

        /* Status-specific styles */
        .status-idle .progress-fill {
          background-color: var(--progress-bg);
        }

        .status-connecting .progress-fill {
          background: linear-gradient(90deg,
            var(--progress-color) 0%,
            var(--progress-color) 50%,
            transparent 50%,
            transparent 100%);
          background-size: 2rem 100%;
          animation: loading-animation 1s linear infinite;
        }

        .status-processing .progress-fill {
          background-color: var(--progress-color);
        }

        .status-completed .progress-fill {
          background-color: var(--color-success);
        }

        .status-error .progress-fill {
          background-color: var(--color-error);
        }

        @keyframes loading-animation {
          0% {
            background-position: -2rem 0;
          }
          100% {
            background-position: 2rem 0;
          }
        }

        /* Loading spinner */
        .loading-spinner {
          width: 14px;
          height: 14px;
          border: 2px solid var(--progress-bg);
          border-top-color: var(--progress-color);
          border-radius: 50%;
          animation: spin 1s linear infinite;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        /* Error icon */
        .error-icon {
          width: 14px;
          height: 14px;
          color: var(--color-error);
        }

        /* Success icon */
        .success-icon {
          width: 14px;
          height: 14px;
          color: var(--color-success);
        }

        /* Details section */
        .progress-details {
          margin-top: 0.75rem;
          padding: 0.5rem;
          background: #f9fafb;
          border-radius: 4px;
          border: 1px solid var(--progress-bg);
        }

        .progress-details[hidden] {
          display: none;
        }

        .model-progress {
          display: flex;
          flex-direction: column;
          gap: 0.25rem;
        }

        .model-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          font-size: 0.75rem;
        }

        .model-name {
          color: var(--text-primary);
          font-weight: 500;
        }

        .model-status {
          color: var(--text-secondary);
          text-transform: capitalize;
        }

        .model-progress-bar {
          width: 60px;
          height: 4px;
          background: var(--progress-bg);
          border-radius: 2px;
          overflow: hidden;
        }

        .model-progress-fill {
          height: 100%;
          background: var(--progress-color);
          transition: width var(--transition-duration) ease-out;
        }

        /* Dark mode support */
        @media (prefers-color-scheme: dark) {
          :host {
            --progress-bg: #374151;
            --text-primary: #f9fafb;
            --text-secondary: #d1d5db;
          }

          .progress-details {
            background: #1f2937;
            border-color: #374151;
          }
        }

        /* Reduced motion support */
        @media (prefers-reduced-motion: reduce) {
          .progress-fill,
          .model-progress-fill {
            transition: none;
          }

          .loading-spinner {
            animation: none;
          }

          @keyframes loading-animation {
            to { background-position: -2rem 0; }
          }
        }

        /* Compact mode */
        :host([compact]) .progress-header {
          margin-bottom: 0.25rem;
        }

        :host([compact]) .progress-stage,
        :host([compact]) .progress-eta {
          font-size: 0.6875rem;
          margin-top: 0.125rem;
        }

        :host([compact]) .progress-details {
          margin-top: 0.5rem;
          padding: 0.375rem;
        }
      </style>      <div class="progress-container">

        <div class="progress-header">
          <div class="progress-status">
            <span class="status-icon"></span>
            <span class="status-text">Ready</span>
          </div>
          <div class="progress-percentage">0%</div>
        </div>

        <div class="progress-track">
          <div class="progress-fill" role="progressbar"
               aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"
               aria-label="Progress indicator"></div>
        </div>

        <div class="progress-stage"></div>
        <div class="progress-eta"></div>

        <div class="progress-details" hidden>
          <div class="model-progress"></div>
        </div>
      </div>
    `;

    this.shadowRoot.appendChild(template.content.cloneNode(true));

    // Cache DOM elements
    this.container = this.shadowRoot.querySelector('.progress-container');
    this.statusIcon = this.shadowRoot.querySelector('.status-icon');
    this.statusText = this.shadowRoot.querySelector('.status-text');
    this.percentageText = this.shadowRoot.querySelector('.progress-percentage');
    this.progressFill = this.shadowRoot.querySelector('.progress-fill');
    this.stageText = this.shadowRoot.querySelector('.progress-stage');
    this.etaText = this.shadowRoot.querySelector('.progress-eta');
    this.detailsSection = this.shadowRoot.querySelector('.progress-details');
    this.modelProgressContainer = this.shadowRoot.querySelector('.model-progress');
  }
  // Private methods for handling progress updates
  #handleProgressUpdate(data) {
    try {
      // Data might already be parsed if coming from SSEClient
      const progressData = typeof data === 'string' ? JSON.parse(data) : data;

      this.updateProgress(progressData);

      // Check if processing is complete
      if (progressData.progress >= 100 || progressData.status === 'completed') {
        this.#updateStatus('completed', 'Analysis completed');
        this.disconnect();
        this.#dispatchEvent('completed', progressData);
      }

    } catch (error) {
      console.error('Failed to parse progress data:', error);
      this.#updateStatus('error', 'Failed to parse progress data');
    }
  }
  #updateStatus(status, stage = '') {
    const oldStatus = this.#status;
    this.#status = status;
    this.#stage = stage;

    // Update UI classes
    this.container.className = `progress-container status-${status}`;

    // Update status icon
    this.#updateStatusIcon(status);

    // Update status text
    this.statusText.textContent = this.#getStatusDisplayText(status);

    // Update stage text
    this.stageText.textContent = stage;

    // Update ARIA label
    this.container.setAttribute('aria-label', `${this.statusText.textContent}: ${stage}`);

    // Dispatch status change event
    if (oldStatus !== status) {
      this.#dispatchEvent('statuschange', { status, stage, oldStatus });
    }
  }

  #updateStatusIcon(status) {
    let iconHTML = '';

    switch (status) {
      case 'connecting':
        iconHTML = '<div class="loading-spinner"></div>';
        break;
      case 'completed':
        iconHTML = `<svg class="success-icon" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
        </svg>`;
        break;
      case 'error':
        iconHTML = `<svg class="error-icon" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
        </svg>`;
        break;
      default:
        iconHTML = '';
    }

    this.statusIcon.innerHTML = iconHTML;
  }

  #getStatusDisplayText(status) {
    const statusTexts = {
      idle: 'Ready',
      connecting: 'Connecting',
      processing: 'Processing',
      completed: 'Completed',
      error: 'Error',
      disconnected: 'Disconnected'
    };

    return statusTexts[status] || status;
  }

  #updateUI() {
    // Update progress bar
    this.progressFill.style.width = `${this.#progressValue}%`;

    // Update percentage display
    this.percentageText.textContent = `${Math.round(this.#progressValue)}%`;    // Update ARIA attributes
    this.progressFill.setAttribute('aria-valuenow', Math.round(this.#progressValue));

    // Update stage text
    if (this.#stage) {
      this.stageText.textContent = this.#stage;
    }

    // Update ETA display
    if (this.#eta) {
      const etaDate = new Date(this.#eta * 1000);
      const now = new Date();
      const remainingMs = etaDate.getTime() - now.getTime();

      if (remainingMs > 0) {
        const remainingSeconds = Math.ceil(remainingMs / 1000);
        this.etaText.textContent = `ETA: ${this.#formatDuration(remainingSeconds)}`;
      } else {
        this.etaText.textContent = '';
      }
    } else {
      this.etaText.textContent = '';
    }

    // Update model progress if details are shown
    if (this.#showDetails && this.#modelProgress) {
      this.#updateModelProgress();
    }
  }

  #updateDetailsVisibility() {
    if (this.detailsSection) {
      this.detailsSection.hidden = !this.#showDetails;
    }
  }

  #updateModelProgress() {
    if (!this.#modelProgress || !this.modelProgressContainer) return;

    const modelEntries = Object.entries(this.#modelProgress);

    this.modelProgressContainer.innerHTML = modelEntries.map(([modelName, data]) => `
      <div class="model-item">
        <span class="model-name">${this.#escapeHtml(modelName)}</span>
        <div class="model-progress-bar">
          <div class="model-progress-fill" style="width: ${data.progress || 0}%"></div>
        </div>
        <span class="model-status">${this.#escapeHtml(data.status || 'pending')}</span>
      </div>
    `).join('');
  }

  #formatDuration(seconds) {
    if (seconds < 60) {
      return `${seconds}s`;
    } else if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = seconds % 60;
      return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
    } else {
      const hours = Math.floor(seconds / 3600);
      const remainingMinutes = Math.floor((seconds % 3600) / 60);
      return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
    }
  }

  #escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
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
customElements.define('progress-indicator', ProgressIndicator);

// Export for testing environments
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ProgressIndicator;
} else if (typeof window !== 'undefined') {
  window.ProgressIndicator = ProgressIndicator;
}

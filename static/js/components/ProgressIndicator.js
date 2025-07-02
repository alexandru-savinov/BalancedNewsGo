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
  // Private properties - declare once at class level
  _sseClient = null;
  _progressValue = 0;
  _status = 'idle';
  _stage = '';
  _articleId = null;
  _reconnectAttempts = 0;
  _reconnectTimer = null;
  _maxReconnectAttempts = 5;
  _baseReconnectDelay = 1000;
  _autoConnect = false;
  _showDetails = false;
  _eta = null;
  _modelProgress = null;

  constructor() {
    super();
    this.attachShadow({ mode: 'open' });

    // Initialize component state
    this._maxReconnectAttempts = 5;
    this._baseReconnectDelay = 1000; // 1 second

    this._render();
  }

  static get observedAttributes() {
    return ['article-id', 'auto-connect', 'show-details'];
  }

  // Getters and setters
  get articleId() {
    return this._articleId;
  }

  set articleId(value) {
    this._articleId = value;
    if (this._autoConnect && value) {
      this.connect(value);
    }
  }

  get progress() {
    return this._progressValue;
  }

  get status() {
    return this._status;
  }

  get autoConnect() {
    return this._autoConnect;
  }

  set autoConnect(value) {
    this._autoConnect = value === true || value === 'true';
  }

  get showDetails() {
    return this._showDetails;
  }

  set showDetails(value) {
    this._showDetails = value === true || value === 'true';
    this._updateDetailsVisibility();
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
    if (this._sseClient && this._sseClient.connected) {
      this.disconnect();
    }

    this._articleId = articleId || this._articleId;

    if (!this._articleId) {
      console.warn('ProgressIndicator: No article ID provided for connection');
      return;
    }

    this._updateStatus('connecting', 'Establishing connection...');

    try {
      this._sseClient = new SSEClient({
        maxReconnectAttempts: this._maxReconnectAttempts,
        reconnectDelay: this._baseReconnectDelay
      });

      // Set up event listeners
      this._sseClient.addEventListener('connected', (data) => {
        this._updateStatus('processing', 'Connected - awaiting progress data...');
        this._reconnectAttempts = 0;
      });

      this._sseClient.addEventListener('message', (data) => {
        this._handleProgressUpdate(data);
      });

      this._sseClient.addEventListener('progress', (data) => {
        this._handleProgressUpdate(data);
      });

      this._sseClient.addEventListener('completed', (data) => {
        this._handleProgressUpdate(data);
      });

      this._sseClient.addEventListener('error', (data) => {
        console.error('ProgressIndicator SSE error:', data);

        // Don't treat disconnection as error if analysis is already complete
        if (this._status === 'completed' || this._progressValue >= 100) {
          console.log('SSE disconnected after completion - this is normal');
          return;
        }

        this._updateStatus('error', 'Connection error occurred');
        this._dispatchEvent('connectionerror', data);
      });

      this._sseClient.addEventListener('disconnected', (data) => {
        if (data.reason !== 'Manual disconnect') {
          // Don't treat disconnection as error if analysis is already complete
          if (this._status === 'completed' || this._progressValue >= 100) {
            console.log('SSE disconnected after completion - this is normal');
            return;
          }
          this._updateStatus('error', 'Connection lost');
        }
      });

      this._sseClient.addEventListener('reconnecting', (data) => {
        this._reconnectAttempts = data.attempt;
        this._updateStatus('connecting', `Reconnecting... (${data.attempt}/${data.maxAttempts})`);
      });

      this._sseClient.addEventListener('failed', (data) => {
        this._updateStatus('error', 'Connection failed - max retries reached');
        this._dispatchEvent('error', data);
      });

      // Connect to the progress endpoint
      const endpoint = `/api/llm/score-progress/${this._articleId}`;
      this._sseClient.connect(endpoint);

    } catch (error) {
      console.error('Failed to establish SSE connection:', error);
      this._updateStatus('error', `Connection failed: ${error.message}`);
    }
  }

  disconnect() {
    if (this._sseClient) {
      this._sseClient.disconnect();
      this._sseClient = null;
    }

    if (this._reconnectTimer) {
      clearTimeout(this._reconnectTimer);
      this._reconnectTimer = null;
    }

    this._reconnectAttempts = 0;
  }

  reset() {
    this.disconnect();
    this._progressValue = 0;
    this._status = 'idle';
    this._stage = '';
    this._eta = null;
    this._updateUI();
  }

  // Manual progress update (for non-SSE scenarios)
  updateProgress(progressData) {
    // Handle both "progress" and "percent" field names from backend
    const progress = progressData.progress || progressData.percent || 0;
    this._progressValue = Math.max(0, Math.min(100, progress));
    this._status = progressData.status || this._status;
    this._stage = progressData.stage || progressData.step || this._stage;
    this._eta = progressData.eta || null;

    this._updateUI();
    this._dispatchEvent('progressupdate', progressData);
  }

  // Private methods
  _render() {
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
      </style>

      <div class="progress-container" role="progressbar"
           aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"
           aria-label="Progress indicator">

        <div class="progress-header">
          <div class="progress-status">
            <span class="status-icon"></span>
            <span class="status-text">Ready</span>
          </div>
          <div class="progress-percentage">0%</div>
        </div>

        <div class="progress-track">
          <div class="progress-fill"></div>
        </div>

        <div class="progress-stage"></div>
        <div class="progress-eta"></div>
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
  }

  // Private methods for handling progress updates
  _handleProgressUpdate(data) {
    try {
      // Data might already be parsed if coming from SSEClient
      const progressData = typeof data === 'string' ? JSON.parse(data) : data;

      this.updateProgress(progressData);

      // Check if processing is complete
      // Handle both backend formats: "progress"/"percent" and "completed"/"Complete"
      const progress = progressData.progress || progressData.percent || 0;
      const status = progressData.status ? progressData.status.toLowerCase() : '';

      // Check for error status first
      if (status === 'error') {
        const errorMessage = progressData.message || 'Analysis failed';
        this._updateStatus('error', errorMessage);
        this.disconnect();
        this._dispatchEvent('error', progressData);
        return;
      }

      if (progress >= 100 || status === 'completed' || status === 'complete') {
        this._updateStatus('completed', 'Analysis completed');
        this._dispatchEvent('completed', progressData);

        // Delay disconnection and auto-hide to allow users to see completion state
        setTimeout(() => {
          this.disconnect();
          this._autoHideAfterCompletion();
        }, 3000); // 3 second delay before disconnecting
      }

    } catch (error) {
      console.error('Failed to parse progress data:', error);
      this._updateStatus('error', 'Failed to parse progress data');
    }
  }

  _updateStatus(status, stage = '') {
    const oldStatus = this._status;
    this._status = status;
    this._stage = stage;

    // Update UI classes
    this.container.className = `progress-container status-${status}`;

    // Update status icon
    this._updateStatusIcon(status);

    // Update status text
    this.statusText.textContent = this._getStatusDisplayText(status);

    // Update stage text
    this.stageText.textContent = stage;

    // Update ARIA label
    this.container.setAttribute('aria-label', `${this.statusText.textContent}: ${stage}`);

    // Dispatch status change event
    if (oldStatus !== status) {
      this._dispatchEvent('statuschange', { status, stage, oldStatus });
    }
  }

  _updateStatusIcon(status) {
    let iconHTML;

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

  _getStatusDisplayText(status) {
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

  _updateUI() {
    // Update progress bar
    this.progressFill.style.width = `${this._progressValue}%`;

    // Update percentage display
    this.percentageText.textContent = `${Math.round(this._progressValue)}%`;

    // Update ARIA attributes
    this.container.setAttribute('aria-valuenow', Math.round(this._progressValue));

    // Update stage text
    if (this._stage) {
      this.stageText.textContent = this._stage;
    }

    // Update ETA display
    if (this._eta) {
      const etaDate = new Date(this._eta * 1000);
      const now = new Date();
      const remainingMs = etaDate.getTime() - now.getTime();

      if (remainingMs > 0) {
        const remainingSeconds = Math.ceil(remainingMs / 1000);
        this.etaText.textContent = `ETA: ${this._formatDuration(remainingSeconds)}`;
      } else {
        this.etaText.textContent = '';
      }
    } else {
      this.etaText.textContent = '';
    }
  }

  _updateDetailsVisibility() {
    // This method is referenced but details section was removed for simplicity
    // Can be implemented later if needed
  }

  _formatDuration(seconds) {
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



  _dispatchEvent(eventName, detail) {
    this.dispatchEvent(new CustomEvent(eventName, {
      detail,
      bubbles: true,
      composed: true
    }));
  }

  // Auto-hide the progress indicator after completion with a delay
  _autoHideAfterCompletion() {
    // Allow users to see the completion state for a few seconds before hiding
    setTimeout(() => {
      // Dispatch a custom event that the parent can listen to for auto-hiding
      this._dispatchEvent('autohide', { reason: 'completion' });
    }, 2000); // Additional 2 seconds after disconnect (total 5 seconds from completion)
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

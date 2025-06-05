/**
 * Admin Dashboard Page JavaScript
 * Handles admin dashboard functionality with real-time system monitoring
 *
 * Features:
 * - RSS feed management and health status
 * - System metrics visualization with Chart.js
 * - Feed refresh controls with progress tracking
 * - User feedback overview and management
 * - System performance indicators
 * - Database statistics display
 * - Real-time status updates via enhanced SSE client
 */

import { FeedHealthMonitor } from '../utils/RealtimeClient.js';
import { monitorFeedHealth } from '../utils/SSEClient.js';

class AdminDashboardPage {
  constructor() {
    this.feedHealthData = null;
    this.systemStats = null;
    this.charts = {};
    this.refreshInProgress = false;
    this.feedHealthMonitor = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;

    // Initialize API client
    this.apiClient = new ApiClient();

    this.init();
  }

  async init() {
    try {
      this.showLoadingState();
      await this.loadDashboardData();
      await this.loadChartLibrary();
      this.setupCharts();
      this.bindEventListeners();
      this.setupRealTimeUpdates();
      this.updatePageTitle();
      this.hideLoadingState();

      // Set up periodic refresh
      this.setupPeriodicRefresh();
    } catch (error) {
      console.error('Failed to initialize admin dashboard:', error);
      this.showErrorState(error);
    }
  }

  async loadDashboardData() {
    try {
      // Load feed health data
      const feedHealthResponse = await this.apiClient.get('/api/admin/feeds/health');
      this.feedHealthData = feedHealthResponse.data;

      // Load system statistics
      const statsResponse = await this.apiClient.get('/api/admin/stats');
      this.systemStats = statsResponse.data;

      this.renderFeedHealth();
      this.renderSystemStats();
      this.renderFeedManagement();

    } catch (error) {
      console.error('Failed to load dashboard data:', error);
      throw error;
    }
  }

  async loadChartLibrary() {
    // Dynamic import of Chart.js to avoid loading it unnecessarily
    if (!window.Chart) {
      try {
        const ChartModule = await import('https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.js');
        window.Chart = ChartModule.default || ChartModule;
      } catch (error) {
        console.error('Failed to load Chart.js:', error);
        // Fall back to text-based charts
        this.useTextCharts = true;
      }
    }
  }

  renderFeedHealth() {
    const container = document.getElementById('feed-health-container');
    if (!container || !this.feedHealthData) return;

    const { feeds, overall } = this.feedHealthData;

    // Overall status card
    const overallCard = document.getElementById('overall-status-card');
    if (overallCard) {
      overallCard.innerHTML = `
        <div class="status-card">
          <h3>Feed Overview</h3>
          <div class="status-grid">
            <div class="status-item">
              <span class="status-label">Total Feeds</span>
              <span class="status-value">${overall.totalFeeds}</span>
            </div>
            <div class="status-item">
              <span class="status-label">Healthy Feeds</span>
              <span class="status-value ${overall.healthyFeeds === overall.totalFeeds ? 'status-success' : 'status-warning'}">
                ${overall.healthyFeeds}
              </span>
            </div>
            <div class="status-item">
              <span class="status-label">Total Articles</span>
              <span class="status-value">${overall.totalArticles.toLocaleString()}</span>
            </div>
            <div class="status-item">
              <span class="status-label">Last Update</span>
              <span class="status-value">${this.formatTimestamp(overall.lastGlobalUpdate)}</span>
            </div>
          </div>
        </div>
      `;
    }

    // Individual feed status
    const feedListContainer = document.getElementById('feed-list');
    if (feedListContainer) {
      const feedsHtml = Object.entries(feeds).map(([feedUrl, feedData]) => `
        <div class="feed-item" data-feed-url="${feedUrl}">
          <div class="feed-header">
            <h4 class="feed-url">${this.truncateUrl(feedUrl)}</h4>
            <span class="feed-status feed-status--${feedData.status}">
              ${feedData.status.toUpperCase()}
            </span>
          </div>
          <div class="feed-details">
            <div class="feed-stat">
              <span class="stat-label">Articles:</span>
              <span class="stat-value">${feedData.articlesCount}</span>
            </div>
            <div class="feed-stat">
              <span class="stat-label">Last Fetch:</span>
              <span class="stat-value">${this.formatTimestamp(feedData.lastFetch)}</span>
            </div>
            <div class="feed-stat">
              <span class="stat-label">Avg Fetch Time:</span>
              <span class="stat-value">${feedData.avgFetchTime}ms</span>
            </div>
            <div class="feed-stat">
              <span class="stat-label">Errors:</span>
              <span class="stat-value ${feedData.errorCount > 0 ? 'stat-error' : ''}">${feedData.errorCount}</span>
            </div>
          </div>
          <div class="feed-actions">
            <button class="btn btn-secondary btn-sm" onclick="adminDashboard.refreshSingleFeed('${feedUrl}')">
              Refresh
            </button>
            <button class="btn btn-tertiary btn-sm" onclick="adminDashboard.viewFeedDetails('${feedUrl}')">
              Details
            </button>
          </div>
        </div>
      `).join('');

      feedListContainer.innerHTML = feedsHtml;
    }
  }

  renderSystemStats() {
    const container = document.getElementById('system-stats-container');
    if (!container || !this.systemStats) return;

    const stats = this.systemStats;

    container.innerHTML = `
      <div class="stats-grid">
        <div class="stat-card">
          <h3>Database</h3>
          <div class="stat-value">${stats.database.totalArticles.toLocaleString()}</div>
          <div class="stat-label">Total Articles</div>
        </div>
        <div class="stat-card">
          <h3>Analysis</h3>
          <div class="stat-value">${stats.analysis.pendingAnalysis}</div>
          <div class="stat-label">Pending Analysis</div>
        </div>
        <div class="stat-card">
          <h3>Performance</h3>
          <div class="stat-value">${stats.performance.avgResponseTime}ms</div>
          <div class="stat-label">Avg Response Time</div>
        </div>
        <div class="stat-card">
          <h3>Storage</h3>
          <div class="stat-value">${this.formatBytes(stats.storage.dbSize)}</div>
          <div class="stat-label">Database Size</div>
        </div>
      </div>
    `;
  }

  renderFeedManagement() {
    const container = document.getElementById('feed-management-container');
    if (!container) return;

    container.innerHTML = `
      <div class="feed-management">
        <div class="management-actions">
          <button class="btn btn-primary" onclick="adminDashboard.refreshAllFeeds()">
            <span id="refresh-all-text">Refresh All Feeds</span>
            <span id="refresh-all-spinner" class="spinner" style="display: none;"></span>
          </button>
          <button class="btn btn-secondary" onclick="adminDashboard.openAddFeedModal()">
            Add New Feed
          </button>
          <button class="btn btn-tertiary" onclick="adminDashboard.exportData()">
            Export Data
          </button>
        </div>
        <div class="management-options">
          <label>
            <input type="checkbox" id="force-refresh-checkbox">
            Force refresh (ignore cache)
          </label>
        </div>
      </div>
    `;
  }

  setupCharts() {
    if (this.useTextCharts || !window.Chart) {
      this.setupTextCharts();
      return;
    }

    this.setupFeedHealthChart();
    this.setupArticleVolumeChart();
    this.setupResponseTimeChart();
  }

  setupFeedHealthChart() {
    const canvas = document.getElementById('feed-health-chart');
    if (!canvas || !this.feedHealthData) return;

    const ctx = canvas.getContext('2d');
    const { feeds } = this.feedHealthData;

    const statusCounts = {
      healthy: 0,
      warning: 0,
      error: 0
    };

    Object.values(feeds).forEach(feed => {
      statusCounts[feed.status]++;
    });

    this.charts.feedHealth = new Chart(ctx, {
      type: 'doughnut',
      data: {
        labels: ['Healthy', 'Warning', 'Error'],
        datasets: [{
          data: [statusCounts.healthy, statusCounts.warning, statusCounts.error],
          backgroundColor: [
            'var(--color-success-500, #10b981)',
            'var(--color-warning-500, #f59e0b)',
            'var(--color-error-500, #ef4444)'
          ],
          borderWidth: 2,
          borderColor: 'var(--color-bg-primary, #ffffff)'
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            position: 'bottom'
          },
          title: {
            display: true,
            text: 'Feed Health Status'
          }
        }
      }
    });
  }

  setupArticleVolumeChart() {
    const canvas = document.getElementById('article-volume-chart');
    if (!canvas || !this.systemStats || !this.systemStats.articleVolume) return;

    const ctx = canvas.getContext('2d');
    const volumeData = this.systemStats.articleVolume;

    this.charts.articleVolume = new Chart(ctx, {
      type: 'line',
      data: {
        labels: volumeData.map(d => this.formatDate(d.date)),
        datasets: [{
          label: 'Articles Added',
          data: volumeData.map(d => d.count),
          borderColor: 'var(--color-primary-500, #3b82f6)',
          backgroundColor: 'var(--color-primary-100, #dbeafe)',
          tension: 0.4
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          title: {
            display: true,
            text: 'Article Volume (Last 7 Days)'
          }
        },
        scales: {
          y: {
            beginAtZero: true
          }
        }
      }
    });
  }

  setupResponseTimeChart() {
    const canvas = document.getElementById('response-time-chart');
    if (!canvas || !this.systemStats || !this.systemStats.responseTime) return;

    const ctx = canvas.getContext('2d');
    const responseTimeData = this.systemStats.responseTime;

    this.charts.responseTime = new Chart(ctx, {
      type: 'bar',
      data: {
        labels: responseTimeData.map(d => d.endpoint),
        datasets: [{
          label: 'Response Time (ms)',
          data: responseTimeData.map(d => d.avgTime),
          backgroundColor: 'var(--color-primary-200, #93c5fd)'
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          title: {
            display: true,
            text: 'API Response Times'
          }
        },
        scales: {
          y: {
            beginAtZero: true
          }
        }
      }
    });
  }

  setupTextCharts() {
    // Fallback text-based charts for when Chart.js is not available
    console.log('Using text-based charts as fallback');

    const feedHealthChart = document.getElementById('feed-health-chart');
    if (feedHealthChart && this.feedHealthData) {
      const { feeds } = this.feedHealthData;
      const statusCounts = {
        healthy: 0,
        warning: 0,
        error: 0
      };

      Object.values(feeds).forEach(feed => {
        statusCounts[feed.status]++;
      });

      feedHealthChart.innerHTML = `
        <div class="text-chart">
          <h4>Feed Health Status</h4>
          <div class="chart-item">
            <span class="chart-label">Healthy:</span>
            <span class="chart-bar" style="width: ${(statusCounts.healthy / Object.keys(feeds).length) * 100}%"></span>
            <span class="chart-value">${statusCounts.healthy}</span>
          </div>
          <div class="chart-item">
            <span class="chart-label">Warning:</span>
            <span class="chart-bar warning" style="width: ${(statusCounts.warning / Object.keys(feeds).length) * 100}%"></span>
            <span class="chart-value">${statusCounts.warning}</span>
          </div>
          <div class="chart-item">
            <span class="chart-label">Error:</span>
            <span class="chart-bar error" style="width: ${(statusCounts.error / Object.keys(feeds).length) * 100}%"></span>
            <span class="chart-value">${statusCounts.error}</span>
          </div>
        </div>
      `;
    }
  }

  bindEventListeners() {
    // Refresh buttons
    const refreshAllButton = document.querySelector('[onclick*="refreshAllFeeds"]');
    if (refreshAllButton) {
      refreshAllButton.addEventListener('click', () => this.refreshAllFeeds());
    }

    // Force refresh checkbox
    const forceRefreshCheckbox = document.getElementById('force-refresh-checkbox');
    if (forceRefreshCheckbox) {
      forceRefreshCheckbox.addEventListener('change', (e) => {
        this.forceRefresh = e.target.checked;
      });
    }

    // Window unload cleanup
    window.addEventListener('beforeunload', () => {
      this.cleanup();
    });
  }
  setupRealTimeUpdates() {
    // Set up feed health monitoring using the new SSE client
    try {
      this.feedHealthMonitor = monitorFeedHealth({
        onConnect: () => {
          console.log('Feed health monitoring connected');
          this.reconnectAttempts = 0;
          this.updateConnectionStatus('connected');
        },

        onHealthUpdate: (data) => {
          this.handleFeedHealthUpdate(data);
        },

        onError: (error) => {
          console.error('Feed health monitoring error:', error);
          this.updateConnectionStatus('error');
        }
      });

    } catch (error) {
      console.error('Failed to setup feed health monitoring:', error);
      this.updateConnectionStatus('failed');
    }
  }

  handleFeedHealthUpdate(data) {
    // Update feed health data
    if (data.feeds) {
      this.feedHealthData = data;
      this.renderFeedHealth();
      this.updateFeedHealthCharts();
    }

    // Handle different update types
    switch (data.type) {
      case 'feed_update':
        this.updateFeedStatus(data.feedUrl, data.status);
        break;
      case 'system_stats':
        this.systemStats = data.stats;
        this.renderSystemStats();
        break;
      case 'refresh_progress':
        this.updateRefreshProgress(data.progress);
        break;
      default:
        console.log('Feed health update:', data);
    }
  }

  updateConnectionStatus(status) {
    const statusElement = document.querySelector('.connection-status');
    if (statusElement) {
      statusElement.className = `connection-status connection-status--${status}`;
      statusElement.textContent = status === 'connected' ? 'Connected' :
                                  status === 'error' ? 'Connection Error' :
                                  'Connection Failed';
    }
  }

  updateFeedStatus(feedUrl, newStatus) {
    if (this.feedHealthData && this.feedHealthData.feeds[feedUrl]) {
      this.feedHealthData.feeds[feedUrl] = { ...this.feedHealthData.feeds[feedUrl], ...newStatus };

      // Update the feed item in the UI
      const feedItem = document.querySelector(`[data-feed-url="${feedUrl}"]`);
      if (feedItem) {
        const statusElement = feedItem.querySelector('.feed-status');
        if (statusElement) {
          statusElement.className = `feed-status feed-status--${newStatus.status}`;
          statusElement.textContent = newStatus.status.toUpperCase();
        }
      }

      // Update the chart if available
      if (this.charts.feedHealth) {
        this.updateFeedHealthChart();
      }
    }
  }

  updateFeedHealthChart() {
    if (!this.charts.feedHealth || !this.feedHealthData) return;

    const { feeds } = this.feedHealthData;
    const statusCounts = {
      healthy: 0,
      warning: 0,
      error: 0
    };

    Object.values(feeds).forEach(feed => {
      statusCounts[feed.status]++;
    });

    this.charts.feedHealth.data.datasets[0].data = [
      statusCounts.healthy,
      statusCounts.warning,
      statusCounts.error
    ];

    this.charts.feedHealth.update();
  }

  async refreshAllFeeds() {
    if (this.refreshInProgress) {
      console.log('Refresh already in progress');
      return;
    }

    this.refreshInProgress = true;
    this.setRefreshButtonState(true);

    try {
      const response = await this.apiClient.post('/api/admin/feeds/refresh', {
        force: this.forceRefresh || false
      });

      if (response.data.taskId) {
        this.trackRefreshProgress(response.data.taskId);
      }

      // Show success message
      this.showNotification('Feed refresh initiated successfully', 'success');

    } catch (error) {
      console.error('Failed to refresh feeds:', error);
      this.showNotification('Failed to refresh feeds: ' + error.message, 'error');
      this.setRefreshButtonState(false);
      this.refreshInProgress = false;
    }
  }

  async refreshSingleFeed(feedUrl) {
    try {
      const response = await this.apiClient.post('/api/admin/feeds/refresh', {
        feedUrls: [feedUrl],
        force: this.forceRefresh || false
      });

      this.showNotification(`Refresh initiated for ${this.truncateUrl(feedUrl)}`, 'success');

    } catch (error) {
      console.error('Failed to refresh feed:', error);
      this.showNotification('Failed to refresh feed: ' + error.message, 'error');
    }
  }

  trackRefreshProgress(taskId) {
    // This would typically use SSE or polling to track progress
    // For now, we'll simulate progress tracking
    let progress = 0;
    const interval = setInterval(() => {
      progress += Math.random() * 20;
      if (progress >= 100) {
        progress = 100;
        clearInterval(interval);
        this.setRefreshButtonState(false);
        this.refreshInProgress = false;
        this.loadDashboardData(); // Reload data
      }
      this.updateRefreshProgress(progress);
    }, 1000);
  }

  updateRefreshProgress(progress) {
    const progressElement = document.getElementById('refresh-progress');
    if (progressElement) {
      progressElement.style.width = `${progress}%`;
    }
  }

  setRefreshButtonState(isLoading) {
    const button = document.querySelector('[onclick*="refreshAllFeeds"]');
    const text = document.getElementById('refresh-all-text');
    const spinner = document.getElementById('refresh-all-spinner');

    if (button) {
      button.disabled = isLoading;
    }
    if (text) {
      text.style.display = isLoading ? 'none' : 'inline';
    }
    if (spinner) {
      spinner.style.display = isLoading ? 'inline' : 'none';
    }
  }

  setupPeriodicRefresh() {
    // Refresh dashboard data every 30 seconds
    this.refreshInterval = setInterval(() => {
      if (!this.refreshInProgress) {
        this.loadDashboardData();
      }
    }, 30000);
  }
  // Utility methods
  formatTimestamp(timestamp) {
    if (!timestamp) return 'Never';
    const date = new Date(timestamp);
    return date.toLocaleString();
  }

  formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString();
  }

  formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  truncateUrl(url) {
    if (url.length <= 50) return url;
    return url.substring(0, 47) + '...';
  }

  showLoadingState() {
    const loadingElement = document.getElementById('dashboard-loading');
    if (loadingElement) {
      loadingElement.style.display = 'block';
    }
  }

  hideLoadingState() {
    const loadingElement = document.getElementById('dashboard-loading');
    if (loadingElement) {
      loadingElement.style.display = 'none';
    }
  }

  showErrorState(error) {
    const errorElement = document.getElementById('dashboard-error');
    if (errorElement) {
      errorElement.innerHTML = `
        <div class="error-message">
          <h3>Failed to load dashboard</h3>
          <p>${error.message || 'An unexpected error occurred'}</p>
          <button class="btn btn-primary" onclick="window.location.reload()">
            Retry
          </button>
        </div>
      `;
      errorElement.style.display = 'block';
    }
  }

  showNotification(message, type = 'info') {
    // Create or update notification element
    let notification = document.getElementById('admin-notification');
    if (!notification) {
      notification = document.createElement('div');
      notification.id = 'admin-notification';
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

  updatePageTitle() {
    document.title = 'Admin Dashboard - NewsBalancer';
  }

  openAddFeedModal() {
    // This would open a modal for adding new RSS feeds
    console.log('Add feed modal would open here');
  }

  viewFeedDetails(feedUrl) {
    // This would show detailed information about a specific feed
    console.log('Feed details for:', feedUrl);
  }

  exportData() {
    // This would export dashboard data
    console.log('Export data functionality would be implemented here');
  }
  cleanup() {
    if (this.feedHealthMonitor) {
      this.feedHealthMonitor.stop();
      this.feedHealthMonitor = null;
    }
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }

    // Destroy charts to prevent memory leaks
    Object.values(this.charts).forEach(chart => {
      if (chart && typeof chart.destroy === 'function') {
        chart.destroy();
      }
    });
  }
}

// Initialize admin dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  // Only initialize if we're on the admin page
  if (document.getElementById('admin-dashboard')) {
    window.adminDashboard = new AdminDashboardPage();
  }
});

// Export for global access
window.AdminDashboardPage = AdminDashboardPage;

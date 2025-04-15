/**
 * Enhanced error handling utilities for News Filter frontend
 */

class ErrorHandler {
  /**
   * Initialize the error handler
   * @param {Object} options - Configuration options
   * @param {Function} options.onError - Global error callback
   * @param {boolean} options.showNotifications - Whether to show error notifications
   * @param {string} options.notificationContainer - Selector for notification container
   */
  constructor(options = {}) {
    this.options = {
      onError: null,
      showNotifications: true,
      notificationContainer: '#notifications',
      ...options
    };

    // Set up global error handling
    this.setupGlobalHandlers();
  }

  /**
   * Set up global error handlers
   */
  setupGlobalHandlers() {
    // Handle unhandled promise rejections
    window.addEventListener('unhandledrejection', (event) => {
      console.error('Unhandled Promise Rejection:', event.reason);
      this.handleError(event.reason);
    });

    // Handle global errors
    window.addEventListener('error', (event) => {
      console.error('Global Error:', event.error);
      this.handleError(event.error);
    });
  }

  /**
   * Handle API responses and extract errors if present
   * @param {Response} response - Fetch API response
   * @returns {Promise<Object>} - Parsed response data
   * @throws {Error} - Enhanced error with API details
   */
  async handleResponse(response) {
    const data = await response.json();
    
    if (!data.success) {
      const error = new Error(data.error?.message || 'Unknown API error');
      error.status = data.error?.code || response.status;
      error.metadata = data.error?.metadata || {};
      error.apiError = true;
      
      this.handleError(error);
      throw error;
    }
    
    // Check for warnings
    if (data.warnings && data.warnings.length > 0) {
      this.handleWarnings(data.warnings);
    }
    
    return data.data;
  }

  /**
   * Process API warnings
   * @param {Array} warnings - Warning objects from API
   */
  handleWarnings(warnings) {
    warnings.forEach(warning => {
      console.warn(`API Warning [${warning.code}]: ${warning.message}`);
      
      if (this.options.showNotifications) {
        this.showNotification({
          type: 'warning',
          message: warning.message,
          code: warning.code
        });
      }
    });
  }

  /**
   * Handle errors consistently
   * @param {Error} error - The error object
   */
  handleError(error) {
    // Call custom error handler if provided
    if (typeof this.options.onError === 'function') {
      this.options.onError(error);
    }
    
    // Show notification if enabled
    if (this.options.showNotifications) {
      let message = error.message || 'An unexpected error occurred';
      let type = 'error';
      let details = '';
      
      // Handle API errors with more context
      if (error.apiError) {
        // Format based on status code
        switch (error.status) {
          case 400:
            type = 'validation';
            
            // Add validation details if available
            if (error.metadata?.validation_errors) {
              details = error.metadata.validation_errors
                .map(err => `${err.field}: ${err.error}`)
                .join('<br>');
            }
            break;
            
          case 401:
            type = 'auth';
            message = 'Authentication required. Please log in again.';
            break;
            
          case 403:
            type = 'permission';
            message = 'You don\'t have permission to perform this action.';
            break;
            
          case 404:
            type = 'not-found';
            break;
            
          case 429:
            type = 'rate-limit';
            
            // Add rate limit details if available
            if (error.metadata?.headers) {
              const resetTime = error.metadata.headers['X-RateLimit-Reset'];
              if (resetTime) {
                const resetDate = new Date(parseInt(resetTime));
                details = `Rate limit will reset at: ${resetDate.toLocaleTimeString()}`;
              }
            }
            break;
            
          case 500:
          case 502:
          case 503:
            type = 'server';
            message = 'Server error. Please try again later.';
            break;
        }
      }
      
      this.showNotification({
        type,
        message,
        details,
        error
      });
    }
  }

  /**
   * Display a notification to the user
   * @param {Object} options - Notification options
   * @param {string} options.type - Notification type (error, warning, info)
   * @param {string} options.message - Main notification message
   * @param {string} options.details - Optional details HTML
   */
  showNotification({ type, message, details = '', code = '' }) {
    const container = document.querySelector(this.options.notificationContainer);
    if (!container) return;
    
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    
    let content = `
      <div class="notification-header">
        <span class="notification-title">${this.getTypeTitle(type)}</span>
        ${code ? `<span class="notification-code">${code}</span>` : ''}
        <button class="notification-close">&times;</button>
      </div>
      <div class="notification-body">
        <p>${message}</p>
        ${details ? `<div class="notification-details">${details}</div>` : ''}
      </div>
    `;
    
    notification.innerHTML = content;
    
    // Add close button handler
    notification.querySelector('.notification-close').addEventListener('click', () => {
      notification.classList.add('notification-hiding');
      setTimeout(() => {
        notification.remove();
      }, 300);
    });
    
    // Auto-remove after 8 seconds for warnings, keep errors until dismissed
    if (type !== 'error') {
      setTimeout(() => {
        if (notification.parentNode) {
          notification.classList.add('notification-hiding');
          setTimeout(() => {
            notification.remove();
          }, 300);
        }
      }, 8000);
    }
    
    container.appendChild(notification);
    
    // Animate in
    setTimeout(() => {
      notification.classList.add('notification-visible');
    }, 10);
  }

  /**
   * Get a user-friendly title for notification types
   * @param {string} type - Notification type
   * @returns {string} - User-friendly title
   */
  getTypeTitle(type) {
    switch (type) {
      case 'error': return 'Error';
      case 'warning': return 'Warning';
      case 'info': return 'Information';
      case 'validation': return 'Validation Error';
      case 'auth': return 'Authentication Error';
      case 'permission': return 'Permission Error';
      case 'not-found': return 'Not Found';
      case 'rate-limit': return 'Rate Limit Exceeded';
      case 'server': return 'Server Error';
      default: return 'Notification';
    }
  }

  /**
   * Wrapper for fetch API with error handling
   * @param {string} url - URL to fetch
   * @param {Object} options - Fetch options
   * @returns {Promise<Object>} - Parsed response data
   */
  async fetchWithErrorHandling(url, options = {}) {
    try {
      const response = await fetch(url, options);
      return await this.handleResponse(response);
    } catch (error) {
      // If it's already an API error that we've handled, just rethrow
      if (error.apiError) {
        throw error;
      }
      
      // Otherwise handle as a network/fetch error
      console.error('Fetch error:', error);
      this.handleError(error);
      throw error;
    }
  }
}

// Create global instance
const errorHandler = new ErrorHandler();

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { ErrorHandler, errorHandler };
}
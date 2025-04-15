/**
 * Enhanced article scoring with improved error handling
 */

class ArticleScorer {
  /**
   * Initialize the article scorer
   * @param {Object} options - Configuration options
   * @param {string} options.apiBaseUrl - Base URL for API calls
   * @param {Object} options.errorHandler - Error handler instance
   */
  constructor(options = {}) {
    this.options = {
      apiBaseUrl: '/api',
      errorHandler: window.errorHandler || null,
      ...options
    };
    
    this.progressListeners = new Map();
    this.activeEventSources = new Map();
  }
  
  /**
   * Start scoring an article
   * @param {number} articleId - The article ID to score
   * @param {Object} options - Scoring options
   * @returns {Promise<Object>} - Response data
   */
  async startScoring(articleId, options = {}) {
    try {
      const url = `${this.options.apiBaseUrl}/llm/reanalyze/${articleId}`;
      
      // Use error handler if available, otherwise use fetch directly
      let response;
      if (this.options.errorHandler) {
        response = await this.options.errorHandler.fetchWithErrorHandling(url, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(options)
        });
      } else {
        const fetchResponse = await fetch(url, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(options)
        });
        
        if (!fetchResponse.ok) {
          const errorData = await fetchResponse.json();
          throw new Error(errorData.error?.message || 'Failed to start scoring');
        }
        
        response = await fetchResponse.json();
      }
      
      // Start listening for progress updates
      this.listenForProgress(articleId);
      
      return response;
    } catch (error) {
      console.error('Error starting article scoring:', error);
      throw error;
    }
  }
  
  /**
   * Listen for progress updates via SSE
   * @param {number} articleId - The article ID to listen for
   */
  listenForProgress(articleId) {
    // Close any existing connection for this article
    this.closeProgressListener(articleId);
    
    // Create a new EventSource connection
    const eventSource = new EventSource(`${this.options.apiBaseUrl}/llm/score-progress/${articleId}`);
    
    // Store the event source
    this.activeEventSources.set(articleId, eventSource);
    
    // Handle messages
    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.handleProgressUpdate(articleId, data);
        
        // Close the connection if we're done
        if (data.status === 'Success' || data.status === 'Error') {
          this.closeProgressListener(articleId);
        }
      } catch (error) {
        console.error('Error parsing progress data:', error);
      }
    };
    
    // Handle errors
    eventSource.onerror = (error) => {
      console.error('SSE connection error:', error);
      this.closeProgressListener(articleId);
      
      // Notify listeners of the error
      this.notifyProgressListeners(articleId, {
        step: 'Error',
        message: 'Connection to server lost',
        percent: 0,
        status: 'Error',
        error: 'SSE connection failed'
      });
    };
  }
  
  /**
   * Close a progress listener
   * @param {number} articleId - The article ID
   */
  closeProgressListener(articleId) {
    const eventSource = this.activeEventSources.get(articleId);
    if (eventSource) {
      eventSource.close();
      this.activeEventSources.delete(articleId);
    }
  }
  
  /**
   * Handle a progress update
   * @param {number} articleId - The article ID
   * @param {Object} data - Progress data
   */
  handleProgressUpdate(articleId, data) {
    // Process the data
    const progressData = {
      step: data.step,
      message: data.message,
      percent: data.percent,
      status: data.status,
      error: data.error || null,
      metadata: data.metadata || null,
      finalScore: data.metadata?.final_score || null
    };
    
    // Show notifications for errors
    if (data.status === 'Error' && this.options.errorHandler) {
      let errorType = 'error';
      let errorMessage = data.error || 'Unknown error';
      let errorDetails = '';
      
      // Handle specific error types
      if (data.step === 'Rate Limited') {
        errorType = 'rate-limit';
        if (data.metadata?.reset_time) {
          errorDetails = `Rate limit will reset at: ${new Date(data.metadata.reset_time).toLocaleTimeString()}`;
        }
      } else if (data.step === 'Provider Error') {
        errorType = 'provider';
        if (data.metadata?.provider_name) {
          errorDetails = `Provider: ${data.metadata.provider_name}`;
        }
      }
      
      // Show the notification
      this.options.errorHandler.showNotification({
        type: errorType,
        message: errorMessage,
        details: errorDetails,
        code: data.step
      });
    }
    
    // Notify listeners
    this.notifyProgressListeners(articleId, progressData);
  }
  
  /**
   * Add a progress listener
   * @param {number} articleId - The article ID
   * @param {Function} listener - Callback function
   * @returns {Function} - Function to remove the listener
   */
  addProgressListener(articleId, listener) {
    if (!this.progressListeners.has(articleId)) {
      this.progressListeners.set(articleId, new Set());
    }
    
    const listeners = this.progressListeners.get(articleId);
    listeners.add(listener);
    
    // Return a function to remove the listener
    return () => {
      if (this.progressListeners.has(articleId)) {
        const listeners = this.progressListeners.get(articleId);
        listeners.delete(listener);
        
        // Clean up if no listeners remain
        if (listeners.size === 0) {
          this.progressListeners.delete(articleId);
          this.closeProgressListener(articleId);
        }
      }
    };
  }
  
  /**
   * Notify all progress listeners for an article
   * @param {number} articleId - The article ID
   * @param {Object} data - Progress data
   */
  notifyProgressListeners(articleId, data) {
    if (this.progressListeners.has(articleId)) {
      const listeners = this.progressListeners.get(articleId);
      listeners.forEach(listener => {
        try {
          listener(data);
        } catch (error) {
          console.error('Error in progress listener:', error);
        }
      });
    }
  }
}

// Create global instance
const articleScorer = new ArticleScorer({
  errorHandler: window.errorHandler
});

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { ArticleScorer, articleScorer };
}
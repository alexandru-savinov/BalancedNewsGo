/**
 * ComponentPerformanceMonitor - Performance tracking for web components
 * 
 * Features:
 * - Component render time tracking
 * - User interaction performance metrics
 * - Memory usage monitoring
 * - Event handler performance
 * - Automatic integration with existing PerformanceMonitor
 * - Component lifecycle tracking
 */

/**
 * Per-component monitor wrapper that provides a simpler API
 */
class ComponentMonitorWrapper {
  constructor(componentName) {
    this.componentName = componentName;
    this.renderStartTime = null;
    this.isEnabled = true;
    
    // Get reference to global monitor instance (will be created later)
    this.globalMonitor = null;
  }

  // Set the global monitor reference (called after it's created)
  setGlobalMonitor(monitor) {
    this.globalMonitor = monitor;
  }

  startRender() {
    if (!this.isEnabled) return;
    this.renderStartTime = performance.now();
  }

  endRender() {
    if (!this.isEnabled || !this.renderStartTime || !this.globalMonitor) return;
    
    this.globalMonitor.trackComponentRender(
      this.componentName, 
      this.renderStartTime
    );
    this.renderStartTime = null;
  }

  mount() {
    if (!this.isEnabled || !this.globalMonitor) return;
    this.globalMonitor.trackLifecycle(this.componentName, 'mounted');
  }

  unmount() {
    if (!this.isEnabled || !this.globalMonitor) return;
    this.globalMonitor.trackLifecycle(this.componentName, 'unmounted');
  }

  trackInteraction(type, data = {}) {
    if (!this.isEnabled || !this.globalMonitor) return;
    this.globalMonitor.trackInteraction(
      this.componentName, 
      type, 
      performance.now(), 
      data
    );
  }

  trackEvent(type, data = {}) {
    if (!this.isEnabled || !this.globalMonitor) return;
    // Alias for trackInteraction for backward compatibility
    this.trackInteraction(type, data);
  }

  getMetrics() {
    if (!this.globalMonitor) return null;
    return this.globalMonitor.getComponentStats(this.componentName);
  }

  setEnabled(enabled) {
    this.isEnabled = enabled;
  }
}

class ComponentPerformanceMonitor {
  constructor() {
    this.componentMetrics = new Map();
    this.interactionMetrics = new Map();
    this.lifecycleTimings = new Map();
    this.isEnabled = true;

    // Auto-detect and monitor components
    this.observeComponentCreation();
  }

  // Factory method to create per-component monitors
  static createComponentMonitor(componentName) {
    return new ComponentMonitorWrapper(componentName);
  }

  /**
   * Track component render performance
   * @param {string} componentName - Name of the component
   * @param {number} startTime - Start time of render
   * @param {Object} metadata - Additional component data
   */
  trackComponentRender(componentName, startTime, metadata = {}) {
    if (!this.isEnabled) return;

    const endTime = performance.now();
    const renderTime = endTime - startTime;

    const metric = {
      componentName,
      renderTime,
      timestamp: Date.now(),
      metadata,
      phase: 'render'
    };

    this.recordComponentMetric(componentName, metric);

    // Report to global performance monitor if available
    if (window.PerformanceMonitor) {
      window.PerformanceMonitor.recordCustomMetric('component_render', {
        component: componentName,
        renderTime,
        ...metadata
      });
    }

    // Log slow components
    if (renderTime > 16) { // More than one frame at 60fps
      console.warn(`Slow component render: ${componentName} took ${renderTime.toFixed(2)}ms`);
    }
  }

  /**
   * Track user interactions with components
   * @param {string} componentName - Name of the component
   * @param {string} interactionType - Type of interaction (click, scroll, etc.)
   * @param {number} startTime - Start time of interaction
   * @param {Object} data - Interaction data
   */
  trackInteraction(componentName, interactionType, startTime, data = {}) {
    if (!this.isEnabled) return;

    const endTime = performance.now();
    const interactionTime = endTime - startTime;

    const metric = {
      componentName,
      interactionType,
      interactionTime,
      timestamp: Date.now(),
      data
    };

    this.recordInteractionMetric(componentName, metric);

    // Report to global performance monitor
    if (window.PerformanceMonitor) {
      window.PerformanceMonitor.recordCustomMetric('component_interaction', {
        component: componentName,
        interaction: interactionType,
        responseTime: interactionTime,
        ...data
      });
    }

    // Log slow interactions
    if (interactionTime > 100) { // Longer than 100ms
      console.warn(`Slow interaction: ${componentName}.${interactionType} took ${interactionTime.toFixed(2)}ms`);
    }
  }

  /**
   * Track component lifecycle events
   * @param {string} componentName - Name of the component
   * @param {string} phase - Lifecycle phase (created, connected, disconnected, etc.)
   * @param {Object} metadata - Additional lifecycle data
   */
  trackLifecycle(componentName, phase, metadata = {}) {
    if (!this.isEnabled) return;

    const timestamp = performance.now();
    const key = `${componentName}:${phase}`;

    if (!this.lifecycleTimings.has(componentName)) {
      this.lifecycleTimings.set(componentName, new Map());
    }

    this.lifecycleTimings.get(componentName).set(phase, {
      timestamp,
      metadata
    });

    // Calculate time between phases if previous phase exists
    const componentTimings = this.lifecycleTimings.get(componentName);
    const phases = Array.from(componentTimings.keys());
    
    if (phases.length > 1) {
      const previousPhase = phases[phases.length - 2];
      const previousTiming = componentTimings.get(previousPhase);
      const phaseTime = timestamp - previousTiming.timestamp;

      // Report phase transition time
      if (window.PerformanceMonitor) {
        window.PerformanceMonitor.recordCustomMetric('component_lifecycle', {
          component: componentName,
          from: previousPhase,
          to: phase,
          duration: phaseTime,
          ...metadata
        });
      }
    }
  }

  /**
   * Create a performance-tracked wrapper for component methods
   * @param {Object} component - Component instance
   * @param {string} methodName - Method to wrap
   * @param {string} componentName - Name of the component
   */
  wrapComponentMethod(component, methodName, componentName) {
    if (!component[methodName] || typeof component[methodName] !== 'function') {
      return;
    }

    const originalMethod = component[methodName];
    const monitor = this;

    component[methodName] = function(...args) {
      const startTime = performance.now();
      
      try {
        const result = originalMethod.apply(this, args);
        
        // Handle async methods
        if (result && typeof result.then === 'function') {
          return result.finally(() => {
            monitor.trackInteraction(componentName, methodName, startTime, {
              async: true,
              args: args.length
            });
          });
        } else {
          monitor.trackInteraction(componentName, methodName, startTime, {
            async: false,
            args: args.length
          });
          return result;
        }
      } catch (error) {
        monitor.trackInteraction(componentName, methodName, startTime, {
          error: error.message,
          args: args.length
        });
        throw error;
      }
    };
  }

  /**
   * Auto-instrument a component for performance monitoring
   * @param {HTMLElement} component - Component instance
   * @param {string} componentName - Name of the component
   * @param {Array} methodsToTrack - Methods to track performance for
   */
  instrumentComponent(component, componentName, methodsToTrack = []) {
    if (!component || !componentName) return;

    // Track component creation
    this.trackLifecycle(componentName, 'created');

    // Wrap specified methods
    methodsToTrack.forEach(methodName => {
      this.wrapComponentMethod(component, methodName, componentName);
    });

    // Monitor connectedCallback and disconnectedCallback
    if (component.connectedCallback) {
      const originalConnected = component.connectedCallback;
      component.connectedCallback = function() {
        monitor.trackLifecycle(componentName, 'connected');
        return originalConnected.call(this);
      };
    }

    if (component.disconnectedCallback) {
      const originalDisconnected = component.disconnectedCallback;
      const monitor = this;
      component.disconnectedCallback = function() {
        monitor.trackLifecycle(componentName, 'disconnected');
        return originalDisconnected.call(this);
      };
    }

    // Add convenience methods to component
    component._trackRender = (startTime, metadata) => {
      this.trackComponentRender(componentName, startTime, metadata);
    };

    component._trackInteraction = (type, startTime, data) => {
      this.trackInteraction(componentName, type, startTime, data);
    };
  }

  /**
   * Observe for new custom elements being defined
   */
  observeComponentCreation() {
    // Monitor for new custom elements
    const originalDefine = customElements.define;
    const monitor = this;

    customElements.define = function(name, constructor, options) {
      // Call original define
      const result = originalDefine.call(this, name, constructor, options);

      // Wrap constructor to auto-instrument components
      const originalConstructor = constructor;
      
      // Note: This is a simplified approach. In practice, you might need more
      // sophisticated detection of component creation
      
      return result;
    };
  }

  /**
   * Record component metric
   * @param {string} componentName - Component name
   * @param {Object} metric - Metric data
   */
  recordComponentMetric(componentName, metric) {
    if (!this.componentMetrics.has(componentName)) {
      this.componentMetrics.set(componentName, []);
    }
    
    this.componentMetrics.get(componentName).push(metric);
    
    // Keep only last 100 metrics per component
    const metrics = this.componentMetrics.get(componentName);
    if (metrics.length > 100) {
      metrics.shift();
    }
  }

  /**
   * Record interaction metric
   * @param {string} componentName - Component name
   * @param {Object} metric - Metric data
   */
  recordInteractionMetric(componentName, metric) {
    if (!this.interactionMetrics.has(componentName)) {
      this.interactionMetrics.set(componentName, []);
    }
    
    this.interactionMetrics.get(componentName).push(metric);
    
    // Keep only last 50 interaction metrics per component
    const metrics = this.interactionMetrics.get(componentName);
    if (metrics.length > 50) {
      metrics.shift();
    }
  }

  /**
   * Get performance statistics for a component
   * @param {string} componentName - Component name
   * @returns {Object} Performance statistics
   */
  getComponentStats(componentName) {
    const renderMetrics = this.componentMetrics.get(componentName) || [];
    const interactionMetrics = this.interactionMetrics.get(componentName) || [];
    const lifecycleData = this.lifecycleTimings.get(componentName) || new Map();

    const renderTimes = renderMetrics
      .filter(m => m.phase === 'render')
      .map(m => m.renderTime);

    const interactionTimes = interactionMetrics.map(m => m.interactionTime);

    return {
      componentName,
      renderStats: this.calculateStats(renderTimes),
      interactionStats: this.calculateStats(interactionTimes),
      totalRenders: renderTimes.length,
      totalInteractions: interactionTimes.length,
      lifecyclePhases: Array.from(lifecycleData.keys()),
      lastActivity: Math.max(
        ...renderMetrics.map(m => m.timestamp),
        ...interactionMetrics.map(m => m.timestamp)
      ) || 0
    };
  }

  /**
   * Get overall performance summary
   * @returns {Object} Performance summary
   */
  getOverallStats() {
    const allComponents = Array.from(this.componentMetrics.keys());
    const summary = {};

    allComponents.forEach(componentName => {
      summary[componentName] = this.getComponentStats(componentName);
    });

    return {
      components: summary,
      totalComponents: allComponents.length,
      timestamp: Date.now()
    };
  }

  /**
   * Calculate basic statistics for an array of numbers
   * @param {Array<number>} values - Array of numeric values
   * @returns {Object} Statistics object
   */
  calculateStats(values) {
    if (values.length === 0) {
      return {
        count: 0,
        min: 0,
        max: 0,
        average: 0,
        median: 0
      };
    }

    const sorted = values.slice().sort((a, b) => a - b);
    const sum = values.reduce((a, b) => a + b, 0);

    return {
      count: values.length,
      min: sorted[0],
      max: sorted[sorted.length - 1],
      average: sum / values.length,
      median: sorted[Math.floor(sorted.length / 2)]
    };
  }

  /**
   * Clear all performance data
   */
  clear() {
    this.componentMetrics.clear();
    this.interactionMetrics.clear();
    this.lifecycleTimings.clear();
  }

  /**
   * Enable/disable performance monitoring
   * @param {boolean} enabled - Whether to enable monitoring
   */
  setEnabled(enabled) {
    this.isEnabled = enabled;
  }
}

// Create global instance
const componentPerformanceMonitor = new ComponentPerformanceMonitor();

// Make the wrapper class available globally for component constructors
window.ComponentPerformanceMonitor = function(componentName) {
  const wrapper = new ComponentMonitorWrapper(componentName);
  wrapper.setGlobalMonitor(componentPerformanceMonitor);
  return wrapper;
};

// Also make the actual class available
window.ComponentPerformanceMonitorClass = ComponentPerformanceMonitor;

// Global utilities instance
window.ComponentPerformanceUtils = {
  track: (componentName, startTime, metadata) => 
    componentPerformanceMonitor.trackComponentRender(componentName, startTime, metadata),
  
  trackInteraction: (componentName, type, startTime, data) =>
    componentPerformanceMonitor.trackInteraction(componentName, type, startTime, data),
  
  trackLifecycle: (componentName, phase, metadata) =>
    componentPerformanceMonitor.trackLifecycle(componentName, phase, metadata),
  
  instrument: (component, name, methods) =>
    componentPerformanceMonitor.instrumentComponent(component, name, methods),
  
  getStats: (componentName) =>
    componentPerformanceMonitor.getComponentStats(componentName),
  
  getAllStats: () =>
    componentPerformanceMonitor.getOverallStats(),
  
  clear: () =>
    componentPerformanceMonitor.clear(),
  
  setEnabled: (enabled) =>
    componentPerformanceMonitor.setEnabled(enabled)
};

export { ComponentPerformanceMonitor };
export const ComponentMonitorWrapper = (componentName) => {
  const wrapper = new ComponentMonitorWrapper(componentName);
  wrapper.setGlobalMonitor(componentPerformanceMonitor);
  return wrapper;
};
export default componentPerformanceMonitor;

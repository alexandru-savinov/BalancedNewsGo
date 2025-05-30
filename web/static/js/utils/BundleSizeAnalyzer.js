/**
 * Bundle Size Analyzer
 * Utility to analyze and report on bundle size and loading performance
 */

class BundleSizeAnalyzer {
  constructor() {
    this.metrics = {
      scripts: [],
      stylesheets: [],
      images: [],
      totalSize: 0,
      loadTimes: {},
      compressionRatio: null
    };

    this.init();
  }

  init() {
    // Analyze resources when DOM is loaded
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => this.analyzeResources());
    } else {
      this.analyzeResources();
    }

    // Track resource load times
    this.trackResourceTiming();
  }

  analyzeResources() {
    // Analyze JavaScript files
    const scripts = document.querySelectorAll('script[src]');
    scripts.forEach(script => {
      this.metrics.scripts.push({
        src: script.src,
        type: 'script',
        element: script
      });
    });

    // Analyze CSS files
    const stylesheets = document.querySelectorAll('link[rel="stylesheet"]');
    stylesheets.forEach(link => {
      this.metrics.stylesheets.push({
        href: link.href,
        type: 'stylesheet',
        element: link
      });
    });

    // Analyze images
    const images = document.querySelectorAll('img');
    images.forEach(img => {
      this.metrics.images.push({
        src: img.src || img.dataset.src,
        type: 'image',
        element: img,
        isLazy: img.hasAttribute('data-src') || img.loading === 'lazy'
      });
    });

    console.log('Bundle analysis complete:', this.metrics);
  }

  trackResourceTiming() {
    // Use Performance API to track resource loading
    if ('performance' in window && 'getEntriesByType' in performance) {
      const observer = new PerformanceObserver((list) => {
        list.getEntries().forEach(entry => {
          if (entry.entryType === 'resource') {
            this.metrics.loadTimes[entry.name] = {
              duration: entry.duration,
              transferSize: entry.transferSize || 0,
              encodedBodySize: entry.encodedBodySize || 0,
              decodedBodySize: entry.decodedBodySize || 0,
              compressionRatio: entry.encodedBodySize > 0
                ? (entry.decodedBodySize / entry.encodedBodySize).toFixed(2)
                : 1
            };
          }
        });
      });

      observer.observe({ entryTypes: ['resource'] });
    }
  }

  async fetchResourceSizes() {
    const resourceSizes = new Map();

    // Fetch sizes for scripts and stylesheets
    const allResources = [
      ...this.metrics.scripts,
      ...this.metrics.stylesheets
    ];

    for (const resource of allResources) {
      try {
        const url = resource.src || resource.href;
        if (url && !url.startsWith('data:')) {
          const response = await fetch(url, { method: 'HEAD' });
          const contentLength = response.headers.get('content-length');
          if (contentLength) {
            resourceSizes.set(url, parseInt(contentLength, 10));
          }
        }
      } catch (error) {
        console.warn(`Could not fetch size for ${url}:`, error);
      }
    }

    return resourceSizes;
  }

  generateReport() {
    const report = {
      summary: {
        totalScripts: this.metrics.scripts.length,
        totalStylesheets: this.metrics.stylesheets.length,
        totalImages: this.metrics.images.length,
        lazyImages: this.metrics.images.filter(img => img.isLazy).length
      },
      performance: {
        loadTimes: this.metrics.loadTimes,
        totalTransferSize: Object.values(this.metrics.loadTimes)
          .reduce((sum, metric) => sum + (metric.transferSize || 0), 0),
        totalDecodedSize: Object.values(this.metrics.loadTimes)
          .reduce((sum, metric) => sum + (metric.decodedBodySize || 0), 0)
      },
      recommendations: this.generateRecommendations()
    };

    return report;
  }

  generateRecommendations() {
    const recommendations = [];

    // Check for large resources
    Object.entries(this.metrics.loadTimes).forEach(([url, metrics]) => {
      if (metrics.transferSize > 1024 * 1024) { // > 1MB
        recommendations.push({
          type: 'size',
          severity: 'high',
          message: `Large resource detected: ${url} (${(metrics.transferSize / 1024 / 1024).toFixed(2)}MB)`
        });
      }

      if (metrics.duration > 1000) { // > 1 second
        recommendations.push({
          type: 'speed',
          severity: 'medium',
          message: `Slow loading resource: ${url} (${metrics.duration.toFixed(0)}ms)`
        });
      }

      if (metrics.compressionRatio > 3) {
        recommendations.push({
          type: 'compression',
          severity: 'low',
          message: `Well compressed resource: ${url} (${metrics.compressionRatio}x compression)`
        });
      }
    });

    // Check for missing lazy loading
    const nonLazyImages = this.metrics.images.filter(img => !img.isLazy);
    if (nonLazyImages.length > 3) {
      recommendations.push({
        type: 'optimization',
        severity: 'medium',
        message: `Consider lazy loading for ${nonLazyImages.length} images`
      });
    }

    // Check script count
    if (this.metrics.scripts.length > 10) {
      recommendations.push({
        type: 'bundling',
        severity: 'medium',
        message: `Consider bundling ${this.metrics.scripts.length} script files`
      });
    }

    return recommendations;
  }

  displayReport() {
    const report = this.generateReport();

    console.group('📊 Bundle Size Analysis Report');
    console.log('Summary:', report.summary);
    console.log('Performance:', report.performance);

    if (report.recommendations.length > 0) {
      console.group('💡 Recommendations');
      report.recommendations.forEach(rec => {
        const emoji = rec.severity === 'high' ? '🔴' : rec.severity === 'medium' ? '🟡' : '🟢';
        console.log(`${emoji} ${rec.type.toUpperCase()}: ${rec.message}`);
      });
      console.groupEnd();
    }

    console.groupEnd();

    return report;
  }

  // Utility methods for performance optimization
  preloadCriticalResources(urls) {
    urls.forEach(url => {
      const link = document.createElement('link');
      link.rel = 'preload';
      link.href = url;

      if (url.endsWith('.js')) {
        link.as = 'script';
      } else if (url.endsWith('.css')) {
        link.as = 'style';
      } else if (url.match(/\.(jpg|jpeg|png|webp|svg)$/i)) {
        link.as = 'image';
      }

      document.head.appendChild(link);
    });
  }

  measureFirstContentfulPaint() {
    if ('performance' in window && 'getEntriesByType' in performance) {
      const paintEntries = performance.getEntriesByType('paint');
      const fcp = paintEntries.find(entry => entry.name === 'first-contentful-paint');
      return fcp ? fcp.startTime : null;
    }
    return null;
  }

  measureLargestContentfulPaint() {
    return new Promise((resolve) => {
      if ('PerformanceObserver' in window) {
        const observer = new PerformanceObserver((list) => {
          const entries = list.getEntries();
          const lastEntry = entries[entries.length - 1];
          resolve(lastEntry ? lastEntry.startTime : null);
        });

        observer.observe({ entryTypes: ['largest-contentful-paint'] });

        // Fallback timeout
        setTimeout(() => resolve(null), 5000);
      } else {
        resolve(null);
      }
    });
  }
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = BundleSizeAnalyzer;
}

// Make available globally
window.BundleSizeAnalyzer = BundleSizeAnalyzer;

export default BundleSizeAnalyzer;

/**
 * LazyLoader - Lazy loading utility for images and content
 * 
 * Features:
 * - Image lazy loading with IntersectionObserver
 * - Blur-to-clear progressive loading
 * - Fallback for browsers without IntersectionObserver
 * - Preload critical images
 * - WebP format detection and fallback
 * - Loading/error state management
 * - Performance metrics integration
 */

class LazyLoader {
  constructor(options = {}) {
    this.options = {
      rootMargin: '50px 0px',
      threshold: 0.01,
      enableBlurEffect: true,
      enableWebP: true,
      placeholderColor: '#f0f0f0',
      errorPlaceholder: '/assets/images/image-error.svg',
      loadingClass: 'lazy-loading',
      loadedClass: 'lazy-loaded',
      errorClass: 'lazy-error',
      ...options
    };

    this.observer = null;
    this.lazyImages = new Set();
    this.loadedImages = new Map();
    this.supportsWebP = false;
    this.performanceEntries = [];

    this.init();
  }

  async init() {
    // Check WebP support
    this.supportsWebP = await this.checkWebPSupport();

    // Initialize intersection observer
    if ('IntersectionObserver' in window) {
      this.observer = new IntersectionObserver(
        this.handleIntersection.bind(this),
        {
          rootMargin: this.options.rootMargin,
          threshold: this.options.threshold
        }
      );
    }

    // Preload critical images
    this.preloadCriticalImages();
  }

  /**
   * Register an image for lazy loading
   * @param {HTMLImageElement} img - Image element to lazy load
   * @param {Object} options - Loading options
   */
  observe(img, options = {}) {
    if (!img || img.tagName !== 'IMG') {
      console.warn('LazyLoader: Invalid image element provided');
      return;
    }

    // Set up image attributes
    this.setupImageElement(img, options);

    // Add to observation
    if (this.observer) {
      this.observer.observe(img);
      this.lazyImages.add(img);
    } else {
      // Fallback for browsers without IntersectionObserver
      this.loadImage(img);
    }
  }

  /**
   * Unobserve an image element
   * @param {HTMLImageElement} img - Image element to stop observing
   */
  unobserve(img) {
    if (this.observer && this.lazyImages.has(img)) {
      this.observer.unobserve(img);
      this.lazyImages.delete(img);
    }
  }

  /**
   * Create a lazy image element
   * @param {Object} config - Image configuration
   * @returns {HTMLImageElement} Configured image element
   */
  createImage(config) {
    const {
      src,
      alt = '',
      className = '',
      width,
      height,
      sizes,
      srcset,
      webpSrc,
      webpSrcset,
      placeholder = true,
      blur = this.options.enableBlurEffect
    } = config;

    const img = document.createElement('img');
    
    // Set basic attributes
    img.alt = alt;
    img.className = `lazy-image ${className}`.trim();
    
    if (width) img.width = width;
    if (height) img.height = height;
    if (sizes) img.sizes = sizes;

    // Set up data attributes for lazy loading
    const actualSrc = this.supportsWebP && webpSrc ? webpSrc : src;
    const actualSrcset = this.supportsWebP && webpSrcset ? webpSrcset : srcset;

    img.dataset.src = actualSrc;
    if (actualSrcset) img.dataset.srcset = actualSrcset;

    // Set placeholder
    if (placeholder) {
      img.src = this.generatePlaceholder(width, height, blur);
    }

    // Apply blur effect if enabled
    if (blur) {
      img.style.filter = 'blur(5px)';
      img.style.transition = 'filter 0.3s ease';
    }

    return img;
  }

  /**
   * Handle intersection observer callbacks
   * @param {IntersectionObserverEntry[]} entries - Intersection entries
   */
  handleIntersection(entries) {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const img = entry.target;
        this.loadImage(img);
        this.unobserve(img);
      }
    });
  }

  /**
   * Setup image element for lazy loading
   * @param {HTMLImageElement} img - Image element
   * @param {Object} options - Setup options
   */
  setupImageElement(img, options = {}) {
    const { placeholder = true, blur = this.options.enableBlurEffect } = options;

    // Add loading class
    img.classList.add(this.options.loadingClass);

    // Set up placeholder if not already set
    if (placeholder && !img.src) {
      const width = img.getAttribute('width') || img.offsetWidth || 300;
      const height = img.getAttribute('height') || img.offsetHeight || 200;
      img.src = this.generatePlaceholder(width, height, blur);
    }

    // Apply blur effect
    if (blur && !img.style.filter) {
      img.style.filter = 'blur(5px)';
      img.style.transition = 'filter 0.3s ease, opacity 0.3s ease';
    }
  }

  /**
   * Load an image
   * @param {HTMLImageElement} img - Image element to load
   */
  async loadImage(img) {
    const startTime = performance.now();
    
    try {
      // Get actual source
      const src = img.dataset.src;
      const srcset = img.dataset.srcset;

      if (!src) {
        throw new Error('No source URL provided');
      }

      // Create new image for preloading
      const newImg = new Image();
      
      if (srcset) newImg.srcset = srcset;
      if (img.sizes) newImg.sizes = img.sizes;

      // Wait for image to load
      await new Promise((resolve, reject) => {
        newImg.onload = resolve;
        newImg.onerror = reject;
        newImg.src = src;
      });

      // Update original image
      img.src = src;
      if (srcset) img.srcset = srcset;

      // Remove blur effect
      if (img.style.filter) {
        img.style.filter = 'none';
      }

      // Update classes
      img.classList.remove(this.options.loadingClass);
      img.classList.add(this.options.loadedClass);

      // Record performance metrics
      const loadTime = performance.now() - startTime;
      this.recordImageLoad(src, loadTime, true);

      // Dispatch loaded event
      img.dispatchEvent(new CustomEvent('lazyloaded', {
        detail: { src, loadTime }
      }));

    } catch (error) {
      console.warn('LazyLoader: Failed to load image', img.dataset.src, error);
      
      // Set error placeholder
      img.src = this.options.errorPlaceholder;
      img.classList.remove(this.options.loadingClass);
      img.classList.add(this.options.errorClass);

      // Record failed load
      const loadTime = performance.now() - startTime;
      this.recordImageLoad(img.dataset.src, loadTime, false);

      // Dispatch error event
      img.dispatchEvent(new CustomEvent('lazyerror', {
        detail: { src: img.dataset.src, error }
      }));
    }
  }

  /**
   * Generate placeholder image URL
   * @param {number} width - Image width
   * @param {number} height - Image height
   * @param {boolean} blur - Whether to apply blur
   * @returns {string} Data URL for placeholder
   */
  generatePlaceholder(width = 300, height = 200, blur = false) {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    
    canvas.width = width;
    canvas.height = height;

    // Create gradient background
    const gradient = ctx.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, this.options.placeholderColor);
    gradient.addColorStop(1, '#e0e0e0');
    
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, width, height);

    // Add subtle pattern
    ctx.fillStyle = 'rgba(0,0,0,0.05)';
    for (let i = 0; i < width; i += 20) {
      for (let j = 0; j < height; j += 20) {
        if ((i + j) % 40 === 0) {
          ctx.fillRect(i, j, 10, 10);
        }
      }
    }

    return canvas.toDataURL('image/png');
  }

  /**
   * Check WebP support
   * @returns {Promise<boolean>} WebP support status
   */
  checkWebPSupport() {
    return new Promise((resolve) => {
      const webP = new Image();
      webP.onload = webP.onerror = () => {
        resolve(webP.height === 2);
      };
      webP.src = 'data:image/webp;base64,UklGRjoAAABXRUJQVlA4IC4AAACyAgCdASoCAAIALmk0mk0iIiIiIgBoSygABc6WWgAA/veff/0PP8bA//LwYAAA';
    });
  }

  /**
   * Preload critical images
   */
  preloadCriticalImages() {
    // Look for images marked as critical
    const criticalImages = document.querySelectorAll('img[data-critical="true"]');
    
    criticalImages.forEach(img => {
      const src = img.dataset.src || img.src;
      if (src && !this.loadedImages.has(src)) {
        this.preloadImage(src);
      }
    });
  }

  /**
   * Preload a single image
   * @param {string} src - Image source URL
   * @returns {Promise<void>} Preload promise
   */
  preloadImage(src) {
    return new Promise((resolve, reject) => {
      if (this.loadedImages.has(src)) {
        resolve();
        return;
      }

      const img = new Image();
      img.onload = () => {
        this.loadedImages.set(src, true);
        resolve();
      };
      img.onerror = reject;
      img.src = src;
    });
  }

  /**
   * Record image loading performance
   * @param {string} src - Image source
   * @param {number} loadTime - Load time in milliseconds
   * @param {boolean} success - Whether load was successful
   */
  recordImageLoad(src, loadTime, success) {
    const entry = {
      src,
      loadTime,
      success,
      timestamp: Date.now(),
      size: this.getImageSize(src)
    };

    this.performanceEntries.push(entry);

    // Report to performance monitor if available
    if (window.PerformanceMonitor) {
      window.PerformanceMonitor.recordCustomMetric('image_load', {
        src,
        loadTime,
        success,
        size: entry.size
      });
    }
  }

  /**
   * Get image size from cache or estimate
   * @param {string} src - Image source
   * @returns {number} Estimated size in bytes
   */
  getImageSize(src) {
    // This is an estimation - in a real app you might get this from headers
    const img = this.loadedImages.get(src);
    if (img && img.naturalWidth && img.naturalHeight) {
      // Rough estimation: width * height * 3 (RGB) * compression factor
      return img.naturalWidth * img.naturalHeight * 3 * 0.3;
    }
    return 0;
  }

  /**
   * Get performance statistics
   * @returns {Object} Performance stats
   */
  getPerformanceStats() {
    const entries = this.performanceEntries;
    const successful = entries.filter(e => e.success);
    const failed = entries.filter(e => !e.success);

    return {
      totalImages: entries.length,
      successfulLoads: successful.length,
      failedLoads: failed.length,
      averageLoadTime: successful.length > 0 
        ? successful.reduce((sum, e) => sum + e.loadTime, 0) / successful.length 
        : 0,
      totalDataTransferred: successful.reduce((sum, e) => sum + e.size, 0),
      successRate: entries.length > 0 ? (successful.length / entries.length) * 100 : 0
    };
  }

  /**
   * Cleanup resources
   */
  destroy() {
    if (this.observer) {
      this.observer.disconnect();
      this.observer = null;
    }
    
    this.lazyImages.clear();
    this.loadedImages.clear();
    this.performanceEntries = [];
  }
}

// Export singleton instance
const lazyLoader = new LazyLoader();

// Global utility functions
window.LazyLoader = {
  observe: (img, options) => lazyLoader.observe(img, options),
  unobserve: (img) => lazyLoader.unobserve(img),
  createImage: (config) => lazyLoader.createImage(config),
  preloadImage: (src) => lazyLoader.preloadImage(src),
  getStats: () => lazyLoader.getPerformanceStats(),
  destroy: () => lazyLoader.destroy()
};

export default lazyLoader;

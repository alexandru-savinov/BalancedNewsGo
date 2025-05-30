/**
 * Code splitting utility for dynamic imports and lazy loading
 * Implements performance optimization strategies for bundle size reduction
 */
class CodeSplitter {
    constructor() {
        this.loadedModules = new Map();
        this.loadingPromises = new Map();
        this.config = {
            retryAttempts: 3,
            retryDelay: 1000,
            chunkLoadTimeout: 30000
        };
    }

    /**
     * Dynamically load Chart.js when needed
     */
    async loadChart() {
        const moduleKey = 'chart';
        
        if (this.loadedModules.has(moduleKey)) {
            return this.loadedModules.get(moduleKey);
        }

        if (this.loadingPromises.has(moduleKey)) {
            return this.loadingPromises.get(moduleKey);
        }

        const loadPromise = this._loadWithRetry(async () => {
            // Check if Chart.js is already loaded globally
            if (typeof window.Chart !== 'undefined') {
                return window.Chart;
            }

            // Try to load from vendor directory
            try {
                await this._loadScript('/js/vendor/chart.js');
                if (typeof window.Chart !== 'undefined') {
                    return window.Chart;
                }
            } catch (error) {
                console.warn('Failed to load Chart.js from vendor:', error);
            }

            // Fallback to CDN
            await this._loadScript('https://cdn.jsdelivr.net/npm/chart.js');
            
            if (typeof window.Chart === 'undefined') {
                throw new Error('Chart.js failed to load');
            }
            
            return window.Chart;
        });

        this.loadingPromises.set(moduleKey, loadPromise);

        try {
            const Chart = await loadPromise;
            this.loadedModules.set(moduleKey, Chart);
            return Chart;
        } finally {
            this.loadingPromises.delete(moduleKey);
        }
    }

    /**
     * Dynamically load DOMPurify when needed
     */
    async loadDOMPurify() {
        const moduleKey = 'dompurify';
        
        if (this.loadedModules.has(moduleKey)) {
            return this.loadedModules.get(moduleKey);
        }

        if (this.loadingPromises.has(moduleKey)) {
            return this.loadingPromises.get(moduleKey);
        }

        const loadPromise = this._loadWithRetry(async () => {
            // Check if DOMPurify is already loaded globally
            if (typeof window.DOMPurify !== 'undefined') {
                return window.DOMPurify;
            }

            // Try to load from vendor directory
            try {
                await this._loadScript('/js/vendor/dompurify.js');
                if (typeof window.DOMPurify !== 'undefined') {
                    return window.DOMPurify;
                }
            } catch (error) {
                console.warn('Failed to load DOMPurify from vendor:', error);
            }

            // Fallback to CDN
            await this._loadScript('https://cdn.jsdelivr.net/npm/dompurify@3.0.5/dist/purify.min.js');
            
            if (typeof window.DOMPurify === 'undefined') {
                throw new Error('DOMPurify failed to load');
            }
            
            return window.DOMPurify;
        });

        this.loadingPromises.set(moduleKey, loadPromise);

        try {
            const DOMPurify = await loadPromise;
            this.loadedModules.set(moduleKey, DOMPurify);
            return DOMPurify;
        } finally {
            this.loadingPromises.delete(moduleKey);
        }
    }

    /**
     * Lazy load components based on viewport intersection
     */
    lazyLoadComponents() {
        if (!('IntersectionObserver' in window)) {
            // Fallback: load all components immediately
            this._loadAllComponents();
            return;
        }

        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    this._loadComponent(entry.target);
                    observer.unobserve(entry.target);
                }
            });
        }, {
            rootMargin: '50px 0px', // Start loading 50px before component is visible
            threshold: 0.1
        });

        // Observe all lazy-loadable components
        document.querySelectorAll('[data-lazy-component]').forEach(element => {
            observer.observe(element);
        });
    }

    /**
     * Preload critical resources
     */
    preloadCriticalResources() {
        const criticalResources = [
            { href: '/css/components/articles.css', as: 'style' },
            { href: '/css/components/navigation.css', as: 'style' },
            { href: '/js/components/BiasSlider.js', as: 'script' },
            { href: '/js/components/ArticleCard.js', as: 'script' }
        ];

        criticalResources.forEach(resource => {
            const link = document.createElement('link');
            link.rel = 'preload';
            link.href = resource.href;
            link.as = resource.as;
            
            if (resource.as === 'script') {
                link.crossOrigin = 'anonymous';
            }
            
            document.head.appendChild(link);
        });
    }

    /**
     * Load component when it becomes visible
     */
    async _loadComponent(element) {
        const componentName = element.dataset.lazyComponent;
        
        try {
            switch (componentName) {
                case 'chart':
                    await this.loadChart();
                    // Initialize chart if element has chart data
                    if (element.dataset.chartData) {
                        this._initializeChart(element);
                    }
                    break;
                    
                case 'bias-slider':
                    if (!customElements.get('bias-slider')) {
                        await this._loadScript('/js/components/BiasSlider.js');
                    }
                    break;
                    
                case 'article-card':
                    if (!customElements.get('article-card')) {
                        await this._loadScript('/js/components/ArticleCard.js');
                    }
                    break;
                    
                case 'progress-indicator':
                    if (!customElements.get('progress-indicator')) {
                        await this._loadScript('/js/components/ProgressIndicator.js');
                    }
                    break;
                    
                default:
                    console.warn(`Unknown lazy component: ${componentName}`);
            }
            
            element.classList.add('lazy-loaded');
        } catch (error) {
            console.error(`Failed to load component ${componentName}:`, error);
            element.classList.add('lazy-load-error');
        }
    }

    /**
     * Load all components immediately (fallback)
     */
    async _loadAllComponents() {
        const components = ['BiasSlider', 'ArticleCard', 'ProgressIndicator', 'Navigation', 'Modal'];
        
        const loadPromises = components.map(component => 
            this._loadScript(`/js/components/${component}.js`).catch(error => {
                console.warn(`Failed to load ${component}:`, error);
            })
        );

        await Promise.allSettled(loadPromises);
    }

    /**
     * Initialize Chart.js instance
     */
    _initializeChart(element) {
        try {
            const chartData = JSON.parse(element.dataset.chartData);
            const Chart = this.loadedModules.get('chart');
            
            if (Chart && chartData) {
                new Chart(element, chartData);
            }
        } catch (error) {
            console.error('Failed to initialize chart:', error);
        }
    }

    /**
     * Load script with timeout and error handling
     */
    _loadScript(src) {
        return new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = src;
            script.async = true;
            
            const timeout = setTimeout(() => {
                reject(new Error(`Script load timeout: ${src}`));
            }, this.config.chunkLoadTimeout);
            
            script.onload = () => {
                clearTimeout(timeout);
                resolve();
            };
            
            script.onerror = () => {
                clearTimeout(timeout);
                reject(new Error(`Script load error: ${src}`));
            };
            
            document.head.appendChild(script);
        });
    }

    /**
     * Retry mechanism for loading modules
     */
    async _loadWithRetry(loadFunction) {
        let lastError;
        
        for (let attempt = 1; attempt <= this.config.retryAttempts; attempt++) {
            try {
                return await loadFunction();
            } catch (error) {
                lastError = error;
                
                if (attempt < this.config.retryAttempts) {
                    const delay = this.config.retryDelay * attempt;
                    await new Promise(resolve => setTimeout(resolve, delay));
                    console.warn(`Retry attempt ${attempt} failed, retrying in ${delay}ms:`, error);
                }
            }
        }
        
        throw lastError;
    }

    /**
     * Get loading status for debugging
     */
    getLoadingStatus() {
        return {
            loadedModules: Array.from(this.loadedModules.keys()),
            currentlyLoading: Array.from(this.loadingPromises.keys()),
            totalModules: this.loadedModules.size
        };
    }

    /**
     * Clear loaded modules (for testing)
     */
    clearCache() {
        this.loadedModules.clear();
        this.loadingPromises.clear();
    }
}

// Initialize global code splitter instance
const codeSplitter = new CodeSplitter();

// Auto-initialize lazy loading when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        codeSplitter.lazyLoadComponents();
        codeSplitter.preloadCriticalResources();
    });
} else {
    codeSplitter.lazyLoadComponents();
    codeSplitter.preloadCriticalResources();
}

// Export for use in modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = CodeSplitter;
} else if (typeof window !== 'undefined') {
    window.CodeSplitter = CodeSplitter;
    window.codeSplitter = codeSplitter;
}

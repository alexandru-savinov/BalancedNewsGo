/**
 * NewsBalancer Main Application JavaScript
 * Handles global application initialization, performance monitoring, and common utilities
 */

class NewsBalancerApp {
    constructor() {
        this.performanceMonitor = null;
        this.lazyLoader = null;
        this.isInitialized = false;
        this.config = {
            enablePerformanceMonitoring: true,
            enableLazyLoading: true,
            performanceConsoleEnabled: true,
            debugMode: false
        };
        
        this.init();
    }

    async init() {
        try {
            console.log('🚀 NewsBalancer Application Starting...');
            
            // Register service worker for caching
            await this.registerServiceWorker();
            
            // Initialize performance monitoring
            if (this.config.enablePerformanceMonitoring) {
                await this.initializePerformanceMonitoring();
            }
            
            // Initialize lazy loading
            if (this.config.enableLazyLoading) {
                await this.initializeLazyLoading();
            }
            
            // Set up global error handling
            this.setupErrorHandling();
            
            // Initialize performance console if enabled
            if (this.config.performanceConsoleEnabled) {
                this.setupPerformanceConsole();
            }
            
            this.isInitialized = true;
            console.log('✅ NewsBalancer Application Initialized Successfully');
            
            // Dispatch application ready event
            this.dispatchAppReadyEvent();
            
        } catch (error) {
            console.error('❌ Failed to initialize NewsBalancer Application:', error);
        }
    }

    async registerServiceWorker() {
        if ('serviceWorker' in navigator) {
            try {
                console.log('📦 Registering Service Worker...');
                const registration = await navigator.serviceWorker.register('/static/sw.js', {
                    scope: '/'
                });
                
                registration.addEventListener('updatefound', () => {
                    console.log('🔄 Service Worker update found');
                    const newWorker = registration.installing;
                    newWorker.addEventListener('statechange', () => {
                        if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                            console.log('🎉 New Service Worker ready - refresh to activate');
                            // Could show a notification to user about available update
                        }
                    });
                });
                
                console.log('✅ Service Worker registered successfully');
            } catch (error) {
                console.warn('⚠️  Service Worker registration failed:', error);
            }
        } else {
            console.warn('⚠️  Service Worker not supported');
        }
    }

    async initializePerformanceMonitoring() {
        try {
            // Performance monitor should already be loaded via script import
            if (window.PerformanceMonitor) {
                this.performanceMonitor = new window.PerformanceMonitor();
                this.performanceMonitor.startMonitoring();
                console.log('📊 Performance Monitoring Initialized');
            } else {
                console.warn('⚠️ PerformanceMonitor not available');
            }
        } catch (error) {
            console.error('❌ Failed to initialize performance monitoring:', error);
        }
    }

    async initializeLazyLoading() {
        try {
            // LazyLoader should already be loaded via script import
            if (window.LazyLoader) {
                this.lazyLoader = new window.LazyLoader({
                    rootMargin: '50px',
                    threshold: 0.1
                });
                console.log('🖼️ Lazy Loading Initialized');
            } else {
                console.warn('⚠️ LazyLoader not available');
            }
        } catch (error) {
            console.error('❌ Failed to initialize lazy loading:', error);
        }
    }

    setupErrorHandling() {
        // Global error handler
        window.addEventListener('error', (event) => {
            console.error('Global Error:', event.error);
            if (this.performanceMonitor) {
                this.performanceMonitor.trackEvent('global_error', {
                    message: event.message,
                    filename: event.filename,
                    lineno: event.lineno,
                    colno: event.colno
                });
            }
        });

        // Unhandled promise rejection handler
        window.addEventListener('unhandledrejection', (event) => {
            console.error('Unhandled Promise Rejection:', event.reason);
            if (this.performanceMonitor) {
                this.performanceMonitor.trackEvent('unhandled_rejection', {
                    reason: event.reason?.message || 'Unknown rejection'
                });
            }
        });
    }

    setupPerformanceConsole() {
        // Create a simple performance console in development
        if (this.config.debugMode) {
            const consoleElement = document.createElement('div');
            consoleElement.id = 'performance-console';
            consoleElement.style.cssText = `
                position: fixed;
                bottom: 10px;
                right: 10px;
                width: 300px;
                max-height: 200px;
                background: rgba(0, 0, 0, 0.9);
                color: #00ff00;
                font-family: monospace;
                font-size: 12px;
                padding: 10px;
                border-radius: 5px;
                overflow-y: auto;
                z-index: 10000;
                display: none;
            `;
            document.body.appendChild(consoleElement);

            // Toggle console with Ctrl+Shift+P
            document.addEventListener('keydown', (event) => {
                if (event.ctrlKey && event.shiftKey && event.key === 'P') {
                    const console = document.getElementById('performance-console');
                    console.style.display = console.style.display === 'none' ? 'block' : 'none';
                }
            });
        }
    }

    dispatchAppReadyEvent() {
        const event = new CustomEvent('newsbalancer:ready', {
            detail: {
                app: this,
                performanceMonitor: this.performanceMonitor,
                lazyLoader: this.lazyLoader,
                timestamp: Date.now()
            }
        });
        window.dispatchEvent(event);
    }

    // Public API methods
    getPerformanceMetrics() {
        return this.performanceMonitor?.getMetrics() || null;
    }

    enableDebugMode() {
        this.config.debugMode = true;
        console.log('🐛 Debug mode enabled. Press Ctrl+Shift+P to toggle performance console.');
    }

    disableDebugMode() {
        this.config.debugMode = false;
        const console = document.getElementById('performance-console');
        if (console) {
            console.style.display = 'none';
        }
    }

    // Utility method to track custom events
    trackEvent(eventName, data = {}) {
        if (this.performanceMonitor) {
            this.performanceMonitor.trackEvent(eventName, data);
        }
    }

    // Get application info
    getAppInfo() {
        return {
            name: 'NewsBalancer',
            version: '1.0.0',
            initialized: this.isInitialized,
            features: {
                performanceMonitoring: !!this.performanceMonitor,
                lazyLoading: !!this.lazyLoader
            },
            config: this.config
        };
    }
}

// Initialize the application when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.NewsBalancerApp = new NewsBalancerApp();
    });
} else {
    window.NewsBalancerApp = new NewsBalancerApp();
}

// Export for use in other modules
window.NewsBalancerApp = NewsBalancerApp;

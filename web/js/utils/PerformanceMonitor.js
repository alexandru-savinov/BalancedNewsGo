/**
 * Performance monitoring utility for Core Web Vitals and application metrics
 * Tracks LCP, FID, CLS, FCP, TTI and reports to backend
 */
class PerformanceMonitor {
    constructor() {
        this.metrics = new Map();
        this.observers = new Map();
        this.config = {
            enableReporting: true,
            reportingEndpoint: '/api/metrics/performance',
            reportingInterval: 30000, // 30 seconds
            batchSize: 10
        };
        this.pendingMetrics = [];
        this.init();
    }

    init() {
        if (typeof window === 'undefined') return;
        
        // Core Web Vitals targets from requirements
        this.targets = {
            LCP: 2500,  // Largest Contentful Paint < 2.5s
            FID: 100,   // First Input Delay < 100ms  
            CLS: 0.1,   // Cumulative Layout Shift < 0.1
            FCP: 1800,  // First Contentful Paint < 1.8s
            TTI: 3500   // Time to Interactive < 3.5s
        };

        this.observeLCP();
        this.observeFID();
        this.observeCLS();
        this.observeFCP();
        this.observeResourceTiming();
        this.startReporting();
        
        // Track custom application metrics
        this.trackPageLoad();
        this.trackAPICallPerformance();
    }

    /**
     * Observe Largest Contentful Paint (LCP)
     */
    observeLCP() {
        if (!('PerformanceObserver' in window)) return;

        try {
            const observer = new PerformanceObserver((entryList) => {
                const entries = entryList.getEntries();
                const lastEntry = entries[entries.length - 1];
                
                this.recordMetric('LCP', lastEntry.startTime, {
                    element: lastEntry.element?.tagName || 'unknown',
                    url: lastEntry.url || window.location.href,
                    isGood: lastEntry.startTime <= this.targets.LCP
                });
            });
            
            observer.observe({ entryTypes: ['largest-contentful-paint'] });
            this.observers.set('LCP', observer);
        } catch (error) {
            console.warn('Failed to observe LCP:', error);
        }
    }

    /**
     * Observe First Input Delay (FID)  
     */
    observeFID() {
        if (!('PerformanceObserver' in window)) return;

        try {
            const observer = new PerformanceObserver((entryList) => {
                for (const entry of entryList.getEntries()) {
                    this.recordMetric('FID', entry.processingStart - entry.startTime, {
                        eventType: entry.name,
                        isGood: (entry.processingStart - entry.startTime) <= this.targets.FID
                    });
                }
            });
            
            observer.observe({ entryTypes: ['first-input'] });
            this.observers.set('FID', observer);
        } catch (error) {
            console.warn('Failed to observe FID:', error);
        }
    }

    /**
     * Observe Cumulative Layout Shift (CLS)
     */
    observeCLS() {
        if (!('PerformanceObserver' in window)) return;

        try {
            let clsValue = 0;
            const observer = new PerformanceObserver((entryList) => {
                for (const entry of entryList.getEntries()) {
                    if (!entry.hadRecentInput) {
                        clsValue += entry.value;
                    }
                }
                
                this.recordMetric('CLS', clsValue, {
                    isGood: clsValue <= this.targets.CLS
                });
            });
            
            observer.observe({ entryTypes: ['layout-shift'] });
            this.observers.set('CLS', observer);
        } catch (error) {
            console.warn('Failed to observe CLS:', error);
        }
    }

    /**
     * Observe First Contentful Paint (FCP)
     */
    observeFCP() {
        if (!('PerformanceObserver' in window)) return;

        try {
            const observer = new PerformanceObserver((entryList) => {
                for (const entry of entryList.getEntries()) {
                    if (entry.name === 'first-contentful-paint') {
                        this.recordMetric('FCP', entry.startTime, {
                            isGood: entry.startTime <= this.targets.FCP
                        });
                    }
                }
            });
            
            observer.observe({ entryTypes: ['paint'] });
            this.observers.set('FCP', observer);
        } catch (error) {
            console.warn('Failed to observe FCP:', error);
        }
    }

    /**
     * Observe resource timing for optimization insights
     */
    observeResourceTiming() {
        if (!('PerformanceObserver' in window)) return;

        try {
            const observer = new PerformanceObserver((entryList) => {
                for (const entry of entryList.getEntries()) {
                    if (entry.duration > 1000) { // Log slow resources
                        this.recordMetric('SLOW_RESOURCE', entry.duration, {
                            name: entry.name,
                            type: entry.initiatorType,
                            size: entry.transferSize || 0
                        });
                    }
                }
            });
            
            observer.observe({ entryTypes: ['resource'] });
            this.observers.set('RESOURCE', observer);
        } catch (error) {
            console.warn('Failed to observe resource timing:', error);
        }
    }

    /**
     * Track page load performance
     */
    trackPageLoad() {
        window.addEventListener('load', () => {
            const navigation = performance.getEntriesByType('navigation')[0];
            if (navigation) {
                this.recordMetric('PAGE_LOAD', navigation.loadEventEnd - navigation.fetchStart, {
                    domContentLoaded: navigation.domContentLoadedEventEnd - navigation.fetchStart,
                    domInteractive: navigation.domInteractive - navigation.fetchStart
                });
            }
        });
    }

    /**
     * Track API call performance
     */
    trackAPICallPerformance() {
        // Intercept fetch calls to track API performance
        const originalFetch = window.fetch;
        window.fetch = async (...args) => {
            const start = performance.now();
            const url = args[0];
            
            try {
                const response = await originalFetch.apply(this, args);
                const duration = performance.now() - start;
                
                if (typeof url === 'string' && url.includes('/api/')) {
                    this.recordMetric('API_CALL', duration, {
                        url: url,
                        status: response.status,
                        success: response.ok
                    });
                }
                
                return response;
            } catch (error) {
                const duration = performance.now() - start;
                if (typeof url === 'string' && url.includes('/api/')) {
                    this.recordMetric('API_CALL', duration, {
                        url: url,
                        status: 0,
                        success: false,
                        error: error.message
                    });
                }
                throw error;
            }
        };
    }

    /**
     * Record a performance metric
     */
    recordMetric(name, value, metadata = {}) {
        const metric = {
            name,
            value,
            timestamp: Date.now(),
            url: window.location.href,
            userAgent: navigator.userAgent,
            ...metadata
        };

        this.metrics.set(`${name}_${Date.now()}`, metric);
        this.pendingMetrics.push(metric);

        // Log important metrics to console for debugging
        if (['LCP', 'FID', 'CLS', 'FCP'].includes(name)) {
            const status = metadata.isGood ? '✅' : '⚠️';
            console.log(`${status} ${name}: ${value.toFixed(2)}ms (target: ${this.targets[name]}ms)`);
        }
    }

    /**
     * Get performance summary
     */
    getPerformanceSummary() {
        const summary = {
            coreWebVitals: {},
            apiCalls: {
                total: 0,
                successful: 0,
                failed: 0,
                averageDuration: 0
            },
            pageLoad: null
        };

        // Core Web Vitals
        for (const [name, target] of Object.entries(this.targets)) {
            const metric = Array.from(this.metrics.values())
                .filter(m => m.name === name)
                .sort((a, b) => b.timestamp - a.timestamp)[0];
            
            if (metric) {
                summary.coreWebVitals[name] = {
                    value: metric.value,
                    target: target,
                    isGood: metric.value <= target,
                    timestamp: metric.timestamp
                };
            }
        }

        // API calls summary
        const apiCalls = Array.from(this.metrics.values()).filter(m => m.name === 'API_CALL');
        if (apiCalls.length > 0) {
            summary.apiCalls.total = apiCalls.length;
            summary.apiCalls.successful = apiCalls.filter(c => c.success).length;
            summary.apiCalls.failed = apiCalls.filter(c => !c.success).length;
            summary.apiCalls.averageDuration = apiCalls.reduce((sum, c) => sum + c.value, 0) / apiCalls.length;
        }

        // Page load
        const pageLoadMetric = Array.from(this.metrics.values()).find(m => m.name === 'PAGE_LOAD');
        if (pageLoadMetric) {
            summary.pageLoad = pageLoadMetric;
        }

        return summary;
    }

    /**
     * Start periodic reporting
     */
    startReporting() {
        if (!this.config.enableReporting) return;

        setInterval(() => {
            this.sendMetrics();
        }, this.config.reportingInterval);

        // Send metrics on page unload
        window.addEventListener('beforeunload', () => {
            this.sendMetrics(true);
        });
    }

    /**
     * Send metrics to backend
     */
    async sendMetrics(immediate = false) {
        if (this.pendingMetrics.length === 0) return;

        const metricsToSend = this.pendingMetrics.splice(0, this.config.batchSize);
        
        try {
            const method = immediate ? 'sendBeacon' : 'fetch';
            
            if (immediate && 'sendBeacon' in navigator) {
                navigator.sendBeacon(
                    this.config.reportingEndpoint,
                    JSON.stringify({ metrics: metricsToSend })
                );
            } else {
                await fetch(this.config.reportingEndpoint, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ metrics: metricsToSend })
                });
            }
        } catch (error) {
            console.warn('Failed to send performance metrics:', error);
            // Put metrics back in queue for retry
            this.pendingMetrics.unshift(...metricsToSend);
        }
    }

    /**
     * Cleanup observers
     */
    cleanup() {
        this.observers.forEach(observer => observer.disconnect());
        this.observers.clear();
    }

    /**
     * Get current performance score (0-100)
     */
    getPerformanceScore() {
        const summary = this.getPerformanceSummary();
        const vitals = summary.coreWebVitals;
        
        let score = 0;
        let totalVitals = 0;
        
        for (const [name, data] of Object.entries(vitals)) {
            if (data && typeof data.value === 'number') {
                totalVitals++;
                if (data.isGood) score += 25; // Each vital worth 25 points
            }
        }
        
        return totalVitals > 0 ? (score / totalVitals) * 4 : 0; // Scale to 0-100
    }
}

// Export for use in modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = PerformanceMonitor;
} else if (typeof window !== 'undefined') {
    window.PerformanceMonitor = PerformanceMonitor;
}

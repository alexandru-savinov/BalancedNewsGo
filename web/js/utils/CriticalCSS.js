/**
 * Critical CSS utility for inlining above-the-fold styles
 * Optimizes initial page rendering by reducing render-blocking CSS
 */
class CriticalCSS {
    constructor() {
        this.criticalStyles = new Map();
        this.loadedStyles = new Set();
        this.config = {
            maxInlineSize: 14000, // 14KB recommended limit for critical CSS
            loadTimeout: 3000
        };

        this.init();
    }

    init() {
        // Extract critical CSS that should be inlined
        this.defineCriticalStyles();

        // Defer non-critical CSS loading
        this.deferNonCriticalCSS();

        // Handle font loading optimization
        this.optimizeFontLoading();
    }

    /**
     * Define critical CSS styles for above-the-fold content
     */
    defineCriticalStyles() {
        // Navigation critical styles
        this.criticalStyles.set('navigation', `
            /* Critical Navigation Styles */
            nav {
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                z-index: 1000;
                background: #fff;
                border-bottom: 1px solid #e0e0e0;
                height: 60px;
                display: flex;
                align-items: center;
                padding: 0 1rem;
            }

            .nav-brand {
                font-size: 1.25rem;
                font-weight: 600;
                color: #333;
                text-decoration: none;
            }

            .nav-links {
                display: flex;
                list-style: none;
                margin: 0;
                padding: 0;
                margin-left: auto;
            }

            .nav-links li {
                margin-left: 1rem;
            }

            .nav-links a {
                color: #666;
                text-decoration: none;
                padding: 0.5rem;
                border-radius: 4px;
                transition: color 0.2s ease;
            }

            .nav-links a:hover,
            .nav-links a[aria-current="page"] {
                color: #f56a6a;
            }
        `);

        // Layout critical styles
        this.criticalStyles.set('layout', `
            /* Critical Layout Styles */
            * {
                box-sizing: border-box;
            }

            body {
                margin: 0;
                padding: 0;
                font-family: 'Open Sans', sans-serif;
                font-size: 13pt;
                line-height: 1.65;
                color: #7f888f;
                background: #ffffff;
                padding-top: 60px; /* Account for fixed nav */
            }

            .container {
                max-width: 1200px;
                margin: 0 auto;
                padding: 0 1rem;
            }

            .main-content {
                min-height: calc(100vh - 60px);
                padding: 2rem 0;
            }

            /* Loading states */
            .loading {
                display: flex;
                align-items: center;
                justify-content: center;
                padding: 2rem;
                color: #999;
            }

            .loading::after {
                content: '';
                width: 20px;
                height: 20px;
                border: 2px solid #f3f3f3;
                border-top: 2px solid #f56a6a;
                border-radius: 50%;
                animation: spin 1s linear infinite;
                margin-left: 0.5rem;
            }

            @keyframes spin {
                0% { transform: rotate(0deg); }
                100% { transform: rotate(360deg); }
            }

            /* Error states */
            .error {
                background: #fff5f5;
                border: 1px solid #fed7d7;
                color: #c53030;
                padding: 1rem;
                border-radius: 4px;
                margin: 1rem 0;
            }
        `);

        // Article card critical styles for list view
        this.criticalStyles.set('article-cards', `
            /* Critical Article Card Styles */
            .articles-grid {
                display: grid;
                grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
                gap: 1.5rem;
                margin: 2rem 0;
            }

            .article-card {
                background: #fff;
                border: 1px solid #e0e0e0;
                border-radius: 8px;
                padding: 1.5rem;
                transition: box-shadow 0.2s ease;
                cursor: pointer;
            }

            .article-card:hover {
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
            }

            .article-title {
                font-size: 1.1rem;
                font-weight: 600;
                color: #333;
                margin-bottom: 0.5rem;
                line-height: 1.4;
            }

            .article-meta {
                display: flex;
                align-items: center;
                gap: 1rem;
                margin-bottom: 1rem;
                font-size: 0.9rem;
                color: #666;
            }

            .article-source {
                font-weight: 500;
            }

            .article-date {
                opacity: 0.8;
            }

            /* Bias slider critical styles */
            .bias-slider {
                position: relative;
                height: 6px;
                background: linear-gradient(to right, #3182ce 0%, #e2e8f0 50%, #e53e3e 100%);
                border-radius: 3px;
                margin: 1rem 0;
            }

            .bias-indicator {
                position: absolute;
                top: -3px;
                width: 12px;
                height: 12px;
                background: #fff;
                border: 2px solid #333;
                border-radius: 50%;
                transform: translateX(-50%);
            }
        `);

        // Responsive critical styles
        this.criticalStyles.set('responsive', `
            /* Critical Responsive Styles */
            @media (max-width: 768px) {
                .nav-links {
                    display: none;
                }

                .container {
                    padding: 0 0.5rem;
                }

                .articles-grid {
                    grid-template-columns: 1fr;
                    gap: 1rem;
                }

                .article-card {
                    padding: 1rem;
                }

                body {
                    font-size: 12pt;
                }
            }

            @media (max-width: 480px) {
                .main-content {
                    padding: 1rem 0;
                }

                .article-title {
                    font-size: 1rem;
                }

                body {
                    font-size: 11pt;
                }
            }
        `);
    }

    /**
     * Inject critical CSS into document head
     */
    injectCriticalCSS() {
        const criticalCSS = Array.from(this.criticalStyles.values()).join('\n');

        // Check if critical CSS would exceed size limit
        if (criticalCSS.length > this.config.maxInlineSize) {
            console.warn(`Critical CSS size (${criticalCSS.length}) exceeds recommended limit (${this.config.maxInlineSize})`);
        }

        const style = document.createElement('style');
        style.textContent = criticalCSS;
        style.setAttribute('data-critical', 'true');

        // Insert before any existing stylesheets
        const firstLink = document.head.querySelector('link[rel="stylesheet"]');
        if (firstLink) {
            document.head.insertBefore(style, firstLink);
        } else {
            document.head.appendChild(style);
        }

        console.log(`Injected ${criticalCSS.length} bytes of critical CSS`);
    }

    /**
     * Defer loading of non-critical CSS
     */
    deferNonCriticalCSS() {
        // Mark existing stylesheets as non-critical
        const stylesheets = document.querySelectorAll('link[rel="stylesheet"]');

        stylesheets.forEach(link => {
            if (!this.isCriticalStylesheet(link.href)) {
                this.deferStylesheet(link);
            }
        });
    }

    /**
     * Check if a stylesheet is critical
     */
    isCriticalStylesheet(href) {
        const criticalPatterns = [
            '/css/normalize.css',
            '/css/critical.css'
        ];

        return criticalPatterns.some(pattern => href.includes(pattern));
    }

    /**
     * Defer loading of a stylesheet
     */
    deferStylesheet(link) {
        // Change media to a non-matching query to prevent blocking
        link.media = 'print';
        link.onload = () => {
            link.media = 'all';
            this.loadedStyles.add(link.href);
        };

        // Fallback for browsers that don't support onload
        setTimeout(() => {
            if (!this.loadedStyles.has(link.href)) {
                link.media = 'all';
                this.loadedStyles.add(link.href);
            }
        }, this.config.loadTimeout);
    }

    /**
     * Optimize font loading
     */
    optimizeFontLoading() {
        // Add font-display: swap to improve loading performance
        const fontOptimizations = `
            @font-face {
                font-family: 'Open Sans';
                font-display: swap;
            }

            @font-face {
                font-family: 'Roboto Slab';
                font-display: swap;
            }
        `;

        const style = document.createElement('style');
        style.textContent = fontOptimizations;
        style.setAttribute('data-font-optimizations', 'true');
        document.head.appendChild(style);
    }

    /**
     * Load CSS file asynchronously
     */
    async loadCSS(href, options = {}) {
        return new Promise((resolve, reject) => {
            if (this.loadedStyles.has(href)) {
                resolve();
                return;
            }

            const link = document.createElement('link');
            link.rel = 'stylesheet';
            link.href = href;

            if (options.media) {
                link.media = options.media;
            }

            link.onload = () => {
                this.loadedStyles.add(href);
                resolve();
            };

            link.onerror = () => {
                reject(new Error(`Failed to load CSS: ${href}`));
            };

            document.head.appendChild(link);
        });
    }

    /**
     * Load CSS files in order
     */
    async loadCSSSequence(hrefs) {
        for (const href of hrefs) {
            try {
                await this.loadCSS(href);
            } catch (error) {
                console.warn(`Failed to load CSS file: ${href}`, error);
            }
        }
    }

    /**
     * Get critical CSS for server-side rendering
     */
    getCriticalCSS() {
        return Array.from(this.criticalStyles.values()).join('\n');
    }

    /**
     * Add custom critical CSS
     */
    addCriticalCSS(key, css) {
        this.criticalStyles.set(key, css);
    }

    /**
     * Remove critical CSS
     */
    removeCriticalCSS(key) {
        this.criticalStyles.delete(key);
    }

    /**
     * Get loading status
     */
    getLoadingStatus() {
        return {
            criticalStyles: Array.from(this.criticalStyles.keys()),
            loadedStyles: Array.from(this.loadedStyles),
            totalCriticalSize: this.getCriticalCSS().length
        };
    }
}

// Auto-initialize critical CSS
const criticalCSS = new CriticalCSS();

// Inject critical CSS if not already done server-side
if (!document.querySelector('style[data-critical]')) {
    criticalCSS.injectCriticalCSS();
}

// Export for use in modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = CriticalCSS;
} else if (typeof window !== 'undefined') {
    window.CriticalCSS = CriticalCSS;
    window.criticalCSS = criticalCSS;
}

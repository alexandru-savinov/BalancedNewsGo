/**
 * Performance Dashboard for Admin Panel
 * Provides real-time performance monitoring and analytics
 */

class PerformanceDashboard {
    constructor() {
        this.performanceMonitor = null;
        this.charts = {};
        this.updateInterval = null;
        this.isInitialized = false;

        this.init();
    }

    async init() {
        try {
            console.log('🚀 Initializing Performance Dashboard...');

            // Wait for NewsBalancer app to be ready
            if (window.NewsBalancerApp) {
                this.performanceMonitor = window.NewsBalancerApp.performanceMonitor;
            } else {
                // Wait for app ready event
                window.addEventListener('newsbalancer:ready', (event) => {
                    this.performanceMonitor = event.detail.performanceMonitor;
                    this.setupDashboard();
                });
                return;
            }

            this.setupDashboard();

        } catch (error) {
            console.error('❌ Failed to initialize Performance Dashboard:', error);
        }
    }

    setupDashboard() {
        if (!this.performanceMonitor) {
            console.warn('⚠️ Performance Monitor not available');
            return;
        }

        this.createDashboardHTML();
        this.setupCharts();
        this.startRealTimeUpdates();
        this.isInitialized = true;

        console.log('✅ Performance Dashboard Initialized');
    }

    createDashboardHTML() {
        const dashboardContainer = document.getElementById('performance-dashboard');
        if (!dashboardContainer) {
            // Create performance dashboard section if it doesn't exist
            const adminDashboard = document.getElementById('admin-dashboard');
            if (adminDashboard) {
                const performanceSection = document.createElement('div');
                performanceSection.id = 'performance-dashboard';
                performanceSection.innerHTML = this.getDashboardHTML();
                adminDashboard.appendChild(performanceSection);
            }
        } else {
            dashboardContainer.innerHTML = this.getDashboardHTML();
        }
    }

    getDashboardHTML() {
        return `
            <div class="admin-section">
                <div class="section-header">
                    <h2 class="section-title">Performance Monitoring</h2>
                    <div class="section-actions">
                        <button type="button" class="btn btn-sm btn-secondary" onclick="performanceDashboard.exportMetrics()">
                            Export Metrics
                        </button>
                        <button type="button" class="btn btn-sm btn-primary" onclick="performanceDashboard.refreshMetrics()">
                            Refresh
                        </button>
                    </div>
                </div>

                <!-- Performance Summary Cards -->
                <div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
                    <div class="metric-card">
                        <div class="metric-card-header">
                            <h3>Page Load Time</h3>
                            <span class="metric-status" id="load-time-status">●</span>
                        </div>
                        <div class="metric-value" id="page-load-time">--</div>
                        <div class="metric-label">milliseconds</div>
                    </div>

                    <div class="metric-card">
                        <div class="metric-card-header">
                            <h3>Memory Usage</h3>
                            <span class="metric-status" id="memory-status">●</span>
                        </div>
                        <div class="metric-value" id="memory-usage">--</div>
                        <div class="metric-label">MB</div>
                    </div>

                    <div class="metric-card">
                        <div class="metric-card-header">
                            <h3>Component Count</h3>
                            <span class="metric-status" id="component-status">●</span>
                        </div>
                        <div class="metric-value" id="component-count">--</div>
                        <div class="metric-label">active</div>
                    </div>

                    <div class="metric-card">
                        <div class="metric-card-header">
                            <h3>Interactions/min</h3>
                            <span class="metric-status" id="interaction-status">●</span>
                        </div>
                        <div class="metric-value" id="interaction-rate">--</div>
                        <div class="metric-label">events</div>
                    </div>
                </div>

                <!-- Performance Charts -->
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div class="chart-container">
                        <h3 class="chart-title">Render Performance</h3>
                        <canvas id="render-performance-chart" width="400" height="200"></canvas>
                    </div>

                    <div class="chart-container">
                        <h3 class="chart-title">Memory Usage Over Time</h3>
                        <canvas id="memory-usage-chart" width="400" height="200"></canvas>
                    </div>

                    <div class="chart-container">
                        <h3 class="chart-title">Component Lifecycle</h3>
                        <canvas id="component-lifecycle-chart" width="400" height="200"></canvas>
                    </div>

                    <div class="chart-container">
                        <h3 class="chart-title">User Interactions</h3>
                        <canvas id="interaction-chart" width="400" height="200"></canvas>
                    </div>
                </div>

                <!-- Performance Log -->
                <div class="performance-log mt-8">
                    <h3 class="log-title">Recent Performance Events</h3>
                    <div class="log-container" id="performance-log">
                        <!-- Log entries will be populated here -->
                    </div>
                </div>
            </div>
        `;
    }

    setupCharts() {
        if (!window.Chart) {
            console.warn('⚠️ Chart.js not available');
            return;
        }

        // Render Performance Chart
        this.charts.renderPerformance = new Chart(
            document.getElementById('render-performance-chart'),
            {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Render Time (ms)',
                        data: [],
                        borderColor: '#3b82f6',
                        backgroundColor: 'rgba(59, 130, 246, 0.1)',
                        tension: 0.1
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Time (ms)'
                            }
                        }
                    }
                }
            }
        );

        // Memory Usage Chart
        this.charts.memoryUsage = new Chart(
            document.getElementById('memory-usage-chart'),
            {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Memory Usage (MB)',
                        data: [],
                        borderColor: '#10b981',
                        backgroundColor: 'rgba(16, 185, 129, 0.1)',
                        tension: 0.1
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Memory (MB)'
                            }
                        }
                    }
                }
            }
        );

        // Component Lifecycle Chart
        this.charts.componentLifecycle = new Chart(
            document.getElementById('component-lifecycle-chart'),
            {
                type: 'doughnut',
                data: {
                    labels: ['Mounted', 'Unmounted', 'Rendering'],
                    datasets: [{
                        data: [0, 0, 0],
                        backgroundColor: ['#10b981', '#f59e0b', '#3b82f6']
                    }]
                },
                options: {
                    responsive: true
                }
            }
        );

        // User Interactions Chart
        this.charts.interactions = new Chart(
            document.getElementById('interaction-chart'),
            {
                type: 'bar',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Interactions',
                        data: [],
                        backgroundColor: '#8b5cf6'
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Count'
                            }
                        }
                    }
                }
            }
        );
    }

    startRealTimeUpdates() {
        this.updateInterval = setInterval(() => {
            this.updateMetrics();
        }, 2000); // Update every 2 seconds
    }

    updateMetrics() {
        if (!this.performanceMonitor) return;

        const metrics = this.performanceMonitor.getMetrics();
        if (!metrics) return;

        // Update summary cards
        this.updateSummaryCards(metrics);

        // Update charts
        this.updateCharts(metrics);

        // Update performance log
        this.updatePerformanceLog(metrics);
    }

    updateSummaryCards(metrics) {
        // Page Load Time
        const loadTime = metrics.pageLoadTime || 0;
        document.getElementById('page-load-time').textContent = Math.round(loadTime);
        document.getElementById('load-time-status').className =
            `metric-status ${loadTime < 1000 ? 'status-good' : loadTime < 3000 ? 'status-warning' : 'status-critical'}`;

        // Memory Usage
        const memory = (metrics.memory?.usedJSHeapSize || 0) / (1024 * 1024);
        document.getElementById('memory-usage').textContent = Math.round(memory);
        document.getElementById('memory-status').className =
            `metric-status ${memory < 50 ? 'status-good' : memory < 100 ? 'status-warning' : 'status-critical'}`;

        // Component Count
        const componentCount = metrics.components?.active || 0;
        document.getElementById('component-count').textContent = componentCount;
        document.getElementById('component-status').className = 'metric-status status-good';

        // Interaction Rate
        const interactions = metrics.userInteractions?.length || 0;
        const rate = Math.round(interactions / 60); // per minute approximation
        document.getElementById('interaction-rate').textContent = rate;
        document.getElementById('interaction-status').className = 'metric-status status-good';
    }

    updateCharts(metrics) {
        const now = new Date().toLocaleTimeString();

        // Update render performance chart
        if (this.charts.renderPerformance && metrics.renderTimes?.length > 0) {
            const chart = this.charts.renderPerformance;
            const avgRenderTime = metrics.renderTimes.reduce((a, b) => a + b, 0) / metrics.renderTimes.length;

            chart.data.labels.push(now);
            chart.data.datasets[0].data.push(avgRenderTime);

            // Keep only last 20 data points
            if (chart.data.labels.length > 20) {
                chart.data.labels.shift();
                chart.data.datasets[0].data.shift();
            }

            chart.update('none');
        }

        // Update memory usage chart
        if (this.charts.memoryUsage && metrics.memory) {
            const chart = this.charts.memoryUsage;
            const memoryMB = metrics.memory.usedJSHeapSize / (1024 * 1024);

            chart.data.labels.push(now);
            chart.data.datasets[0].data.push(memoryMB);

            if (chart.data.labels.length > 20) {
                chart.data.labels.shift();
                chart.data.datasets[0].data.shift();
            }

            chart.update('none');
        }

        // Update component lifecycle chart
        if (this.charts.componentLifecycle && metrics.components) {
            const chart = this.charts.componentLifecycle;
            chart.data.datasets[0].data = [
                metrics.components.mounted || 0,
                metrics.components.unmounted || 0,
                metrics.components.rendering || 0
            ];
            chart.update('none');
        }
    }

    updatePerformanceLog(metrics) {
        const logContainer = document.getElementById('performance-log');
        if (!logContainer) return;

        // Add recent events to log
        if (metrics.events && metrics.events.length > 0) {
            const recentEvents = metrics.events.slice(-5); // Show last 5 events
            const logHTML = recentEvents.map(event => `
                <div class="log-entry">
                    <span class="log-time">${new Date(event.timestamp).toLocaleTimeString()}</span>
                    <span class="log-type">${event.type}</span>
                    <span class="log-message">${event.message || JSON.stringify(event.data)}</span>
                </div>
            `).join('');

            logContainer.innerHTML = logHTML;
        }
    }

    refreshMetrics() {
        if (this.performanceMonitor) {
            this.performanceMonitor.refreshMetrics();
            this.updateMetrics();
        }
    }

    exportMetrics() {
        if (!this.performanceMonitor) {
            alert('Performance monitor not available');
            return;
        }

        const metrics = this.performanceMonitor.getMetrics();
        const dataStr = JSON.stringify(metrics, null, 2);
        const dataBlob = new Blob([dataStr], { type: 'application/json' });

        const url = URL.createObjectURL(dataBlob);
        const link = document.createElement('a');
        link.href = url;
        link.download = `performance-metrics-${new Date().toISOString().slice(0, 19)}.json`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    }

    destroy() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
        }

        Object.values(this.charts).forEach(chart => {
            if (chart && typeof chart.destroy === 'function') {
                chart.destroy();
            }
        });
    }
}

// Auto-initialize when DOM is ready and we're on admin page
document.addEventListener('DOMContentLoaded', () => {
    if (document.getElementById('admin-dashboard')) {
        window.performanceDashboard = new PerformanceDashboard();
    }
});

// Export for global access
window.PerformanceDashboard = PerformanceDashboard;

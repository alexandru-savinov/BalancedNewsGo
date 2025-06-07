/**
 * Admin API Service
 *
 * Handles all admin-related API operations including:
 * - Source management (add, update, delete sources)
 * - System monitoring and statistics
 * - User management and permissions
 * - Configuration management
 * - Bulk operations and maintenance tasks
 */

class AdminService {
    constructor(apiClient) {
        this.client = apiClient || new ApiClient();
        this.baseUrl = '/admin';
    }

    /**
     * Get system statistics and overview
     * @returns {Promise} System stats response
     */
    async getStats() {
        try {
            return await this.client.getJson(`${this.baseUrl}/stats`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getStats'
            });
        }
    }

    /**
     * Get detailed system health information
     * @returns {Promise} Health check response
     */
    async getHealth() {
        try {
            return await this.client.getJson(`${this.baseUrl}/health`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getHealth'
            });
        }
    }

    // Source Management

    /**
     * Get all news sources
     * @param {Object} options - Query options
     * @returns {Promise} Sources response
     */
    async getSources(options = {}) {
        const params = {
            active: options.active,
            sort: options.sort || 'name',
            search: options.search
        };

        // Remove undefined parameters
        Object.keys(params).forEach(key => {
            if (params[key] === undefined) {
                delete params[key];
            }
        });

        try {
            return await this.client.getJson(`${this.baseUrl}/sources`, { params });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getSources',
                params
            });
        }
    }

    /**
     * Get specific source details
     * @param {number} sourceId - Source ID
     * @returns {Promise} Source details
     */
    async getSource(sourceId) {
        try {
            return await this.client.getJson(`${this.baseUrl}/sources/${sourceId}`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getSource',
                sourceId
            });
        }
    }

    /**
     * Create new news source
     * @param {Object} sourceData - Source configuration
     * @returns {Promise} Created source
     */
    async createSource(sourceData) {
        const requiredFields = ['name', 'url', 'rss_url'];
        const missingFields = requiredFields.filter(field => !sourceData[field]);

        if (missingFields.length > 0) {
            throw new Error(`Missing required fields: ${missingFields.join(', ')}`);
        }

        try {
            return await this.client.postJson(`${this.baseUrl}/sources`, sourceData);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'createSource',
                sourceData
            });
        }
    }

    /**
     * Update existing news source
     * @param {number} sourceId - Source ID
     * @param {Object} updateData - Updated source data
     * @returns {Promise} Updated source
     */
    async updateSource(sourceId, updateData) {
        try {
            return await this.client.putJson(`${this.baseUrl}/sources/${sourceId}`, updateData);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'updateSource',
                sourceId,
                updateData
            });
        }
    }

    /**
     * Delete news source
     * @param {number} sourceId - Source ID
     * @returns {Promise} Deletion confirmation
     */
    async deleteSource(sourceId) {
        try {
            return await this.client.delete(`${this.baseUrl}/sources/${sourceId}`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'deleteSource',
                sourceId
            });
        }
    }

    /**
     * Test RSS feed for a source
     * @param {string} rssUrl - RSS feed URL to test
     * @returns {Promise} Feed test results
     */
    async testRssFeed(rssUrl) {
        try {
            return await this.client.postJson(`${this.baseUrl}/sources/test-rss`, { rss_url: rssUrl });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'testRssFeed',
                rssUrl
            });
        }
    }

    // User Management

    /**
     * Get all users
     * @param {Object} options - Query options
     * @returns {Promise} Users list
     */
    async getUsers(options = {}) {
        const params = {
            limit: options.limit || 50,
            offset: options.offset || 0,
            role: options.role,
            active: options.active,
            search: options.search
        };

        // Remove undefined parameters
        Object.keys(params).forEach(key => {
            if (params[key] === undefined) {
                delete params[key];
            }
        });

        try {
            return await this.client.getJson(`${this.baseUrl}/users`, { params });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getUsers',
                params
            });
        }
    }

    /**
     * Update user permissions
     * @param {number} userId - User ID
     * @param {Object} permissions - Permission updates
     * @returns {Promise} Updated user
     */
    async updateUserPermissions(userId, permissions) {
        try {
            return await this.client.putJson(`${this.baseUrl}/users/${userId}/permissions`, permissions);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'updateUserPermissions',
                userId,
                permissions
            });
        }
    }

    /**
     * Deactivate user account
     * @param {number} userId - User ID
     * @returns {Promise} Deactivation confirmation
     */
    async deactivateUser(userId) {
        try {
            return await this.client.putJson(`${this.baseUrl}/users/${userId}/deactivate`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'deactivateUser',
                userId
            });
        }
    }

    // Configuration Management

    /**
     * Get system configuration
     * @returns {Promise} Configuration object
     */
    async getConfig() {
        try {
            return await this.client.getJson(`${this.baseUrl}/config`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getConfig'
            });
        }
    }

    /**
     * Update system configuration
     * @param {Object} configData - Configuration updates
     * @returns {Promise} Updated configuration
     */
    async updateConfig(configData) {
        try {
            return await this.client.putJson(`${this.baseUrl}/config`, configData);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'updateConfig',
                configData
            });
        }
    }

    /**
     * Get LLM service configuration and status
     * @returns {Promise} LLM configuration
     */
    async getLlmConfig() {
        try {
            return await this.client.getJson(`${this.baseUrl}/config/llm`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getLlmConfig'
            });
        }
    }

    /**
     * Update LLM service configuration
     * @param {Object} llmConfig - LLM configuration updates
     * @returns {Promise} Updated LLM configuration
     */
    async updateLlmConfig(llmConfig) {
        try {
            return await this.client.putJson(`${this.baseUrl}/config/llm`, llmConfig);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'updateLlmConfig',
                llmConfig
            });
        }
    }

    // Bulk Operations

    /**
     * Trigger bulk article re-analysis
     * @param {Object} criteria - Re-analysis criteria
     * @returns {Promise} Bulk operation status
     */
    async bulkReanalyze(criteria = {}) {
        try {
            return await this.client.postJson(`${this.baseUrl}/bulk/reanalyze`, criteria);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'bulkReanalyze',
                criteria
            });
        }
    }

    /**
     * Get bulk operation status
     * @param {string} operationId - Operation ID
     * @returns {Promise} Operation status
     */
    async getBulkOperationStatus(operationId) {
        try {
            return await this.client.getJson(`${this.baseUrl}/bulk/status/${operationId}`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getBulkOperationStatus',
                operationId
            });
        }
    }

    /**
     * Cancel bulk operation
     * @param {string} operationId - Operation ID
     * @returns {Promise} Cancellation confirmation
     */
    async cancelBulkOperation(operationId) {
        try {
            return await this.client.delete(`${this.baseUrl}/bulk/cancel/${operationId}`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'cancelBulkOperation',
                operationId
            });
        }
    }

    // Maintenance Tasks

    /**
     * Clean up old articles and data
     * @param {Object} cleanupOptions - Cleanup configuration
     * @returns {Promise} Cleanup results
     */
    async cleanupData(cleanupOptions = {}) {
        const options = {
            days_old: cleanupOptions.daysOld || 365,
            include_articles: cleanupOptions.includeArticles !== false,
            include_feedback: cleanupOptions.includeFeedback !== false,
            include_logs: cleanupOptions.includeLogs !== false,
            dry_run: cleanupOptions.dryRun === true
        };

        try {
            return await this.client.postJson(`${this.baseUrl}/maintenance/cleanup`, options);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'cleanupData',
                options
            });
        }
    }

    /**
     * Rebuild search indexes
     * @returns {Promise} Rebuild operation status
     */
    async rebuildIndexes() {
        try {
            return await this.client.postJson(`${this.baseUrl}/maintenance/rebuild-indexes`);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'rebuildIndexes'
            });
        }
    }

    /**
     * Generate system backup
     * @param {Object} backupOptions - Backup configuration
     * @returns {Promise} Backup operation status
     */
    async createBackup(backupOptions = {}) {
        const options = {
            include_articles: backupOptions.includeArticles !== false,
            include_config: backupOptions.includeConfig !== false,
            include_users: backupOptions.includeUsers !== false,
            compression: backupOptions.compression || 'gzip'
        };

        try {
            return await this.client.postJson(`${this.baseUrl}/maintenance/backup`, options);
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'createBackup',
                options
            });
        }
    }

    // Logs and Monitoring

    /**
     * Get system logs
     * @param {Object} options - Log query options
     * @returns {Promise} Log entries
     */
    async getLogs(options = {}) {
        const params = {
            level: options.level,
            source: options.source,
            limit: options.limit || 100,
            offset: options.offset || 0,
            since: options.since,
            until: options.until
        };

        // Remove undefined parameters
        Object.keys(params).forEach(key => {
            if (params[key] === undefined) {
                delete params[key];
            }
        });

        try {
            return await this.client.getJson(`${this.baseUrl}/logs`, { params });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getLogs',
                params
            });
        }
    }

    /**
     * Get system metrics
     * @param {Object} options - Metrics query options
     * @returns {Promise} System metrics
     */
    async getMetrics(options = {}) {
        const params = {
            metric: options.metric,
            timeframe: options.timeframe || '1h',
            granularity: options.granularity || '5m'
        };

        // Remove undefined parameters
        Object.keys(params).forEach(key => {
            if (params[key] === undefined) {
                delete params[key];
            }
        });

        try {
            return await this.client.getJson(`${this.baseUrl}/metrics`, { params });
        } catch (error) {
            throw this.client.errorHandler.handle(error, {
                operation: 'getMetrics',
                params
            });
        }
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = AdminService;
} else if (typeof window !== 'undefined') {
    window.AdminService = AdminService;
}

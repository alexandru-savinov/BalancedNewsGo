"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const test_1 = require("@playwright/test");
const child_process_1 = require("child_process");
const eventsource_1 = require("eventsource");
// Prepare the environment before tests
// This will run e2e_prep.js to snapshot feeds, check health, and ingest articles
test_1.test.beforeAll(async () => {
    (0, child_process_1.execSync)('node e2e_prep.js', { stdio: 'inherit' });
});
test_1.test.describe('Backend API E2E', () => {
    (0, test_1.test)('should get articles and rescore', async ({ request }) => {
        // Get articles
        const articlesRes = await request.get('/api/articles');
        (0, test_1.expect)(articlesRes.ok()).toBeTruthy();
        const articlesResponse = await articlesRes.json();
        console.log('ARTICLES RESPONSE:', articlesResponse);
        (0, test_1.expect)(Array.isArray(articlesResponse.data)).toBeTruthy();
        (0, test_1.expect)(articlesResponse.data.length).toBeGreaterThan(0);
        const article = articlesResponse.data[0];
        (0, test_1.expect)(article.id).toBeDefined();
        // Rescore the article
        const rescoreRes = await request.post(`/api/llm/reanalyze/${article.id}`, {
            data: {}
        });
        (0, test_1.expect)(rescoreRes.ok()).toBeTruthy();
        const rescore = await rescoreRes.json();
        (0, test_1.expect)(['reanalyze queued', 'score updated']).toContain(rescore.data.status);
        // Use SSE to wait for scoring to complete
        const sseUrl = `http://localhost:8080/api/llm/score-progress/${article.id}`;
        let sseDone = false;
        let sseStatus = null;
        let sseError = null;
        await new Promise((resolve, reject) => {
            const es = new eventsource_1.EventSource(sseUrl);
            es.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    if (data.status === 'Success' || data.status === 'Error') {
                        sseStatus = data.status;
                        sseDone = true;
                        es.close();
                        resolve();
                    }
                }
                catch (e) {
                    sseError = e;
                    es.close();
                    reject(e);
                }
            };
            es.onerror = (err) => {
                sseError = err;
                es.close();
                reject(err);
            };
            // Timeout after 40s
            setTimeout(() => {
                if (!sseDone) {
                    es.close();
                    reject(new Error('SSE progress timeout'));
                }
            }, 40000);
        });
        (0, test_1.expect)(sseError).toBeNull();
        (0, test_1.expect)(['Success', 'Error']).toContain(sseStatus);
        // Now get ensemble details
        const ensembleRes = await request.get(`/api/articles/${article.id}/ensemble`);
        (0, test_1.expect)(ensembleRes.ok()).toBeTruthy();
        const ensemble = await ensembleRes.json();
        (0, test_1.expect)(ensemble).toHaveProperty('scores');
    });
});

import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';
import { EventSource } from 'eventsource';

// Prepare the environment before tests
// This will run e2e_prep.js to snapshot feeds, check health, and ingest articles

test.beforeAll(async () => {
  execSync('node e2e_prep.js', { stdio: 'inherit' });
});

test.describe('Backend API E2E', () => {
  test('should get articles and rescore', async ({ request }) => {
    // Get articles
    const articlesRes = await request.get('/api/articles');
    expect(articlesRes.ok()).toBeTruthy();
    const articlesResponse = await articlesRes.json();
    console.log('ARTICLES RESPONSE:', articlesResponse);
    expect(Array.isArray(articlesResponse.data)).toBeTruthy();
    expect(articlesResponse.data.length).toBeGreaterThan(0);
    const article = articlesResponse.data[0];
    expect(article.id).toBeDefined();

    // Rescore the article
    const rescoreRes = await request.post(`/api/llm/reanalyze/${article.id}`, {
      data: {}
    });
    expect(rescoreRes.ok()).toBeTruthy();
    const rescore = await rescoreRes.json();
    expect(['reanalyze queued', 'score updated']).toContain(rescore.data.status);

    // Use SSE to wait for scoring to complete
    const sseUrl = `http://localhost:8080/api/llm/score-progress/${article.id}`;
    let sseDone = false;
    let sseStatus = null;
    let sseError = null;
    await new Promise<void>((resolve, reject) => {
      const es = new EventSource(sseUrl);
      es.onmessage = (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          if (data.status === 'Success' || data.status === 'Error') {
            sseStatus = data.status;
            sseDone = true;
            es.close();
            resolve();
          }
        } catch (e: any) {
          sseError = e;
          es.close();
          reject(e);
        }
      };
      es.onerror = (err: any) => {
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
    expect(sseError).toBeNull();
    expect(['Success', 'Error']).toContain(sseStatus);

    // Now get ensemble details
    const ensembleRes = await request.get(`/api/articles/${article.id}/ensemble`);
    expect(ensembleRes.ok()).toBeTruthy();
    const ensemble = await ensembleRes.json();
    expect(ensemble).toHaveProperty('scores');
  });
});

import { test, expect, APIRequestContext } from '@playwright/test';
import { execSync } from 'child_process';
import { EventSource } from 'eventsource'; // Ensure 'eventsource' is installed (npm install --save-dev eventsource @types/eventsource)

// Base URL for the API, should match Playwright config or be set here
const baseURL = process.env.BASE_URL || 'http://localhost:8080';

// Helper function to create a unique article for testing
async function createTestArticle(request: APIRequestContext, suffix: string): Promise<{ id: string, url: string }> {
  const uniqueUrl = `https://example.com/test-${suffix}-${Date.now()}`;
  const createRes = await request.post('/api/articles', {
    data: {
      title: `Test Article ${suffix}`,
      content: `This is a test article for ${suffix}.`,
      source: 'test',
      url: uniqueUrl,
      pub_date: new Date().toISOString()
    }
  });
  expect(createRes.ok(), `Failed to create article for ${suffix}`).toBeTruthy();
  const articleData = await createRes.json();
  expect(articleData?.data?.article_id, `Article ID not found in response for ${suffix}`).toBeDefined();
  return { id: articleData.data.article_id, url: uniqueUrl };
}

// Helper function to wait for SSE completion
async function waitForSseCompletion(sseUrl: string, timeoutMs: number = 40000): Promise<{ status: string | null, error: Error | null }> {
  let sseStatus: string | null = null;
  let sseError: Error | null = null;
  let sseDone = false;

  await new Promise<void>((resolve, reject) => {
    const es = new EventSource(sseUrl);
    let timeoutId: NodeJS.Timeout | null = null;

    const cleanup = () => {
      if (timeoutId) clearTimeout(timeoutId);
      es.close();
    };

    es.onmessage = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data);
        // Adjust based on actual SSE message structure ('completed', 'failed', 'Success', 'Error' etc.)
        if (data.status === 'completed' || data.status === 'failed' || data.status === 'Success' || data.status === 'Error') {
          sseStatus = data.status;
          sseDone = true;
          cleanup();
          resolve();
        } else if (data.status === 'in_progress') {
          // Optional: log progress
          // console.log(`SSE Progress: ${data.progress}%`);
        }
      } catch (e: any) {
        sseError = e;
        cleanup();
        reject(e);
      }
    };

    es.onerror = (err: any) => {
      // EventSource can fire 'error' for various reasons, including connection close.
      // Only reject if it's a real error and we haven't finished.
      if (!sseDone) {
        sseError = new Error(`SSE error: ${err.message || 'Unknown SSE error'}`);
        cleanup();
        reject(sseError);
      }
    };

    // Timeout
    timeoutId = setTimeout(() => {
      if (!sseDone) {
        sseError = new Error('SSE progress timeout');
        cleanup();
        reject(sseError);
      }
    }, timeoutMs);
  });

  return { status: sseStatus, error: sseError };
}


// Prepare the environment before tests
test.beforeAll(async () => {
  console.log('Running E2E preparation script...');
  try {
    execSync('node e2e_prep.js', { stdio: 'inherit', timeout: 120000 }); // Increased timeout for prep script
    console.log('E2E preparation script finished.');
  } catch (error) {
    console.error('E2E preparation script failed:', error);
    // Optionally re-throw or exit if prep is critical
    throw new Error('E2E preparation failed, cannot run tests.');
  }
});

test.describe('Backend API E2E Tests', () => {

  test('should get articles and initiate rescore', async ({ request }) => {
    // 1. Get existing articles
    const articlesRes = await request.get('/api/articles');
    // Add logging for diagnostics
    if (!articlesRes.ok()) {
      console.error(`Failed to get articles. Status: ${articlesRes.status()}`);
      try {
        const body = await articlesRes.text(); // Use text() first in case it's not JSON
        console.error(`Response body: ${body}`);
      } catch (e) {
        console.error('Could not read response body.');
      }
    }
    expect(articlesRes.ok(), `Failed to get articles. Status: ${articlesRes.status()}`).toBeTruthy();
    const articlesResponse = await articlesRes.json();
    expect(Array.isArray(articlesResponse?.data), 'Articles response data should be an array').toBeTruthy();
    expect(articlesResponse.data.length, 'Expected at least one article from e2e_prep.js').toBeGreaterThan(0);

    // 2. Pick an article
    const article = articlesResponse.data[0];
    expect(article?.id, 'Article should have an ID').toBeDefined();

    // 3. Initiate LLM rescore
    const rescoreRes = await request.post(`/api/llm/reanalyze/${article.id}`, {
      // LLM reanalyze typically doesn't need a body, or might take specific parameters
      data: {} // Adjust if the endpoint expects specific data
    });
    expect(rescoreRes.ok(), `Failed to initiate rescore for article ${article.id}`).toBeTruthy();
    const rescore = await rescoreRes.json();

    // Check response status - adjust based on actual API behavior
    expect(rescore?.data?.status, 'Rescore initiation response status is unexpected')
      .toMatch(/queued|started|pending|updated/i); // Use regex for flexibility
  });

  test.describe('Manual Scoring API [/api/manual-score]', () => {
    let manualScoreArticleId: string;

    test.beforeAll(async ({ request }) => {
      // Check if endpoint exists before running tests in this group
      const checkRes = await request.get('/api/manual-score/1'); // Use a placeholder ID
      if (checkRes.status() === 404) {
        console.warn('Manual score endpoint (/api/manual-score) returned 404, skipping related tests.');
        test.skip(true, 'Manual score endpoint not implemented or available.');
        return; // Skip beforeAll if endpoint is missing
      }
      // Create one article for all manual scoring tests
      const article = await createTestArticle(request, 'manual-score');
      manualScoreArticleId = article.id;
    });

    test('should accept valid boundary scores', async ({ request }) => {
      // Test upper boundary (1.0)
      const upperRes = await request.post(`/api/manual-score/${manualScoreArticleId}`, {
        data: { score: 1.0 }
      });
      expect(upperRes.ok(), 'Failed to set score to upper boundary 1.0').toBeTruthy();

      // Test lower boundary (-1.0)
      const lowerRes = await request.post(`/api/manual-score/${manualScoreArticleId}`, {
        data: { score: -1.0 }
      });
      expect(lowerRes.ok(), 'Failed to set score to lower boundary -1.0').toBeTruthy();
    });

    test('should reject scores outside boundaries', async ({ request }) => {
      // Test above upper boundary (e.g., 1.1)
      const tooHighRes = await request.post(`/api/manual-score/${manualScoreArticleId}`, {
        data: { score: 1.1 }
      });
      expect(tooHighRes.status(), 'Score 1.1 should be rejected (400)').toBe(400);

      // Test below lower boundary (e.g., -1.1)
      const tooLowRes = await request.post(`/api/manual-score/${manualScoreArticleId}`, {
        data: { score: -1.1 }
      });
      expect(tooLowRes.status(), 'Score -1.1 should be rejected (400)').toBe(400);
    });

    test('should reject invalid input formats', async ({ request }) => {
      // Test with missing score field
      const missingRes = await request.post(`/api/manual-score/${manualScoreArticleId}`, {
        data: {}
      });
      expect(missingRes.status(), 'Request with missing score should be rejected (400)').toBe(400);

      // Test with non-numeric score
      const nonNumericRes = await request.post(`/api/manual-score/${manualScoreArticleId}`, {
        data: { score: "not-a-number" }
      });
      expect(nonNumericRes.status(), 'Request with non-numeric score should be rejected (400)').toBe(400);

      // Test with extra fields (assuming strict validation)
      const extraFieldsRes = await request.post(`/api/manual-score/${manualScoreArticleId}`, {
        data: { score: 0.5, unexpectedField: "value" }
      });
      expect(extraFieldsRes.status(), 'Request with extra fields should be rejected (400)').toBe(400);
    });
  });

  test.describe('Error Handling', () => {
    test('should handle non-existent article ID for LLM rescore', async ({ request }) => {
      const nonExistentId = '999999999';
      const nonExistentRes = await request.post(`/api/llm/reanalyze/${nonExistentId}`, {
        data: {}
      });
      // Expect 404 Not Found if the article doesn't exist
      expect(nonExistentRes.status(), `Rescore for non-existent article ${nonExistentId} should return 404`).toBe(404);
    });

    test('should reject manual score payload sent to LLM endpoint', async ({ request }) => {
      // Create an article specifically for this test
      const article = await createTestArticle(request, 'error-handling');

      // Attempt to send a body with a 'score' field to the LLM endpoint
      const wrongEndpointRes = await request.post(`/api/llm/reanalyze/${article.id}`, {
        data: { score: 0.5 } // This payload structure is for manual scoring
      });
      // Expect 400 Bad Request because the payload is incorrect for this endpoint
      expect(wrongEndpointRes.status(), 'Sending manual score payload to LLM endpoint should return 400').toBe(400);
      const errorBody = await wrongEndpointRes.json();
      expect(errorBody?.error?.message, 'Error message should indicate invalid payload for LLM endpoint')
        .toMatch(/invalid|unexpected|disallowed/i); // Check for keywords like 'invalid field', 'unexpected field', 'score field not allowed'
    });
  });


  test('should get articles, wait for scoring completion via SSE, and get ensemble details', async ({ request }) => {
    // 1. Get articles
    const articlesRes = await request.get('/api/articles');
    expect(articlesRes.ok(), 'Failed to get articles').toBeTruthy();
    const articlesResponse = await articlesRes.json();
    expect(Array.isArray(articlesResponse?.data), 'Articles response data should be an array').toBeTruthy();
    expect(articlesResponse.data.length, 'Expected at least one article').toBeGreaterThan(0);
    const article = articlesResponse.data[0]; // Use the first available article
    expect(article?.id, 'Article should have an ID').toBeDefined();
    console.log(`Using article ID: ${article.id} for SSE test`);

    // 2. Initiate Rescore (ensure it happens before listening to SSE)
    const rescoreRes = await request.post(`/api/llm/reanalyze/${article.id}`, { data: {} });
    expect(rescoreRes.ok(), `Failed to initiate rescore for article ${article.id}`).toBeTruthy();
    console.log(`Rescore initiated for article ${article.id}`);

    // 3. Use SSE to wait for scoring to complete
    // Construct the SSE URL using the baseURL
    const sseUrl = `${baseURL}/api/llm/score-progress/${article.id}`;
    console.log(`Listening to SSE at: ${sseUrl}`);
    const sseResult = await waitForSseCompletion(sseUrl);

    expect(sseResult.error, `SSE connection or processing failed: ${sseResult.error?.message}`).toBeNull();
    // Check for a successful completion status (adjust based on actual API)
    expect(sseResult.status, `SSE did not complete successfully. Final status: ${sseResult.status}`)
      .toMatch(/completed|success/i);
    console.log(`SSE completed with status: ${sseResult.status}`);

    // 4. Now get ensemble details
    const ensembleRes = await request.get(`/api/articles/${article.id}/ensemble`);
    expect(ensembleRes.ok(), `Failed to get ensemble details for article ${article.id}`).toBeTruthy();
    const ensemble = await ensembleRes.json();
    expect(ensemble, 'Ensemble response should not be null').toBeDefined();
    expect(ensemble.data, 'Ensemble response should have a "data" property').toBeDefined();
    expect(ensemble.data.ensembles, 'Ensemble data should have an "ensembles" property').toBeDefined();
    expect(Array.isArray(ensemble.data.ensembles), '"ensembles" should be an array').toBeTruthy();
    console.log(`Successfully retrieved ensemble details for article ${article.id}`);
  });

});

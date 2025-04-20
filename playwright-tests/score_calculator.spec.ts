import { test, expect } from '@playwright/test';
import { spawn } from 'child_process';

// Test the DefaultScoreCalculator component
test.describe('DefaultScoreCalculator Tests', () => {
  let testProcess;
  let testOutput = '';

  // Helper to run Go test executable
  async function runGoTest(testName: string): Promise<string> {
    return new Promise((resolve, reject) => {
      // Assumes the test binary is built before running these tests
      const testProcess = spawn('go', ['test', '-run', testName, './internal/llm']);
      let output = '';

      testProcess.stdout.on('data', (data) => {
        const chunk = data.toString();
        output += chunk;
        console.log(chunk);
      });

      testProcess.stderr.on('data', (data) => {
        const chunk = data.toString();
        output += chunk;
        console.error(chunk);
      });

      testProcess.on('close', (code) => {
        if (code === 0) {
          resolve(output);
        } else {
          reject(new Error(`Test process exited with code ${code}: ${output}`));
        }
      });
    });
  }

  // Consolidated test for the Go DefaultScoreCalculator logic
  test('DefaultScoreCalculator - Go Logic Test', async () => {
    // Run the single available Go test function that covers various scenarios
    const result = await runGoTest('TestDefaultScoreCalculator_CalculateScore');
    expect(result).toContain('PASS');
    expect(result).not.toContain('FAIL');
  });

  // Test integration with the API
  test('API integration - Score calculation endpoint', async ({ request }) => {
    // Test the API endpoint that uses the score calculator
    const response = await request.post('/api/test/calculate-score', {
      data: {
        scores: [
          { model: "left", score: -0.8, metadata: JSON.stringify({ confidence: 0.9 }) },
          { model: "center", score: 0.0, metadata: JSON.stringify({ confidence: 0.8 }) },
          { model: "right", score: 0.6, metadata: JSON.stringify({ confidence: 0.7 }) }
        ]
      }
    });
    
    expect(response.ok()).toBeTruthy();
    const result = await response.json();
    expect(result.score).toBeCloseTo(-0.067, 2);
    expect(result.confidence).toBeCloseTo(0.8, 2);
  });

  // Integration with SSE progress
  test('Integration - Score calculator with progress reporting', async ({ page }) => {
    // Set up test article ID
    const testArticleId = '12345';
    
    // Start the score calculation process
    const apiResponse = await page.request.post(`/api/llm/reanalyze/${testArticleId}`, {
      data: {}
    });
    expect(apiResponse.ok()).toBeTruthy();
    
    // Navigate to a page that displays score updates
    await page.goto(`/article/${testArticleId}`);
    
    // Wait for score calculation to complete and verify UI updates
    await page.waitForSelector('.bias-slider .composite-indicator', { state: 'visible', timeout: 30000 });
    
    // Verify confidence indicator appears
    await page.waitForSelector('.confidence-indicator', { state: 'visible' });
    
    // Verify final score is displayed
    const scoreText = await page.textContent('.composite-score');
    expect(scoreText).toMatch(/Score: -?\d+\.\d+/);
  });
});
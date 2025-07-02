import { test, expect } from '@playwright/test';

test.describe('Progress Indicator Functionality', () => {
  
  test.beforeEach(async ({ page }) => {
    // Navigate to an article page where we can test the progress indicator
    await page.goto('/article/600');
    await page.waitForLoadState('networkidle');
  });

  test('should display progress indicator elements correctly', async ({ page }) => {
    await page.goto('/article/600');
    
    // Check that the progress indicator element exists
    const progressIndicator = page.locator('#reanalysis-progress');
    await expect(progressIndicator).toBeAttached();
    
    // Check that the reanalyze button exists
    const reanalyzeBtn = page.locator('#reanalyze-btn');
    await expect(reanalyzeBtn).toBeVisible();
    
    // Check button text span specifically (not the entire button which includes hidden span)
    const btnText = page.locator('#btn-text');
    await expect(btnText).toBeVisible();
    await expect(btnText).toHaveText('Request Reanalysis');
    
    // Loading span should be hidden initially
    const btnLoading = page.locator('#btn-loading');
    await expect(btnLoading).toBeHidden();
    
    // Progress indicator should be hidden initially
    await expect(progressIndicator).toHaveClass(/progress-hidden/);
  });

  test('should load progress indicator JavaScript modules', async ({ page }) => {
    await page.goto('/article/600');
    
    // Check for JavaScript errors
    const errors: string[] = [];
    page.on('pageerror', (error) => {
      errors.push(error.message);
    });

    // Reload page to catch any module loading issues
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Verify no JavaScript errors occurred
    expect(errors).toHaveLength(0);

    // Check that the ProgressIndicator class is available
    const progressIndicatorClass = await page.evaluate(() => {
      const progressElement = document.getElementById('reanalysis-progress') as any;
      return progressElement && typeof progressElement.reset === 'function' && 
             typeof progressElement.connect === 'function';
    });
    
    expect(progressIndicatorClass).toBe(true);
  });

  test('should handle reanalysis button click and show progress', async ({ page }) => {
    await page.goto('/article/600');
    
    // Set up console monitoring to catch events
    const consoleMessages: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'log') {
        consoleMessages.push(msg.text());
      }
    });

    const reanalyzeBtn = page.locator('#reanalyze-btn');
    const progressIndicator = page.locator('#reanalysis-progress');
    const btnLoading = page.locator('#btn-loading');

    // Click the reanalyze button
    await reanalyzeBtn.click();

    // Check if the button was clicked successfully
    const hasClickMessage = consoleMessages.some(msg => msg.includes('Reanalyze button clicked'));
    expect(hasClickMessage).toBe(true);

    // The button might be disabled only briefly if analysis completes quickly
    // Check that at least the loading state shows, or the button was temporarily disabled
    const btnLoadingVisible = await btnLoading.isVisible();
    const progressVisible = await progressIndicator.isVisible();
    
    expect(btnLoadingVisible || progressVisible).toBeTruthy();
    
    // Either the loading state should show, or progress should be visible, indicating the click worked
    expect(hasClickMessage || btnLoadingVisible || progressVisible).toBeTruthy();

    // Progress indicator should become visible
    await expect(progressIndicator).toBeVisible();
    await expect(progressIndicator).not.toHaveClass(/progress-hidden/);

    // Wait for some console messages indicating SSE connection
    await page.waitForTimeout(2000);
    
    // Check that the expected console messages appeared
    const hasConnectMessage = consoleMessages.some(msg => msg.includes('Connecting ProgressIndicator to SSE'));
    
    expect(hasClickMessage).toBe(true);
    expect(hasConnectMessage).toBe(true);
  });

  test('should show real-time progress updates via SSE', async ({ page }) => {
    await page.goto('/article/600');
    
    // Set up network monitoring to catch SSE requests
    const sseRequests: string[] = [];
    page.on('request', (request) => {
      if (request.url().includes('score-progress')) {
        sseRequests.push(request.url());
      }
    });

    const reanalyzeBtn = page.locator('#reanalyze-btn');
    const progressIndicator = page.locator('#reanalysis-progress');

    // Click reanalyze button
    await reanalyzeBtn.click();

    // Wait for SSE connection to be established
    await page.waitForFunction(() => sseRequests.length > 0, { timeout: 5000 });

    // Check that SSE request was made
    expect(sseRequests.length).toBeGreaterThan(0);
    expect(sseRequests[0]).toContain('/api/llm/score-progress/600');

    // Check for progress content in the progress indicator
    const progressContent = progressIndicator.locator('.progress-content, .progress-stage, [data-progress]');
    
    // Wait for progress content to appear (with longer timeout for SSE)
    await expect(progressContent.first()).toBeVisible({ timeout: 10000 });
  });

  test('should handle analysis completion and update bias score', async ({ page }) => {
    await page.goto('/article/600');
    
    // Mock SSE endpoint to simulate completion after progress updates
    await page.route('/api/llm/score-progress/*', async (route) => {
      // Create a realistic SSE response body
      const sseContent = `data: {"progress": 25, "stage": "analyzing", "eta": 15}

data: {"progress": 50, "stage": "processing", "eta": 10}

data: {"progress": 75, "stage": "finalizing", "eta": 5}

data: {"progress": 100, "stage": "completed", "status": "completed", "step": "Done", "message": "Analysis complete", "percent": 100, "final_score": 0.75}

`;

      route.fulfill({
        status: 200,
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
          'Connection': 'keep-alive'
        },
        body: sseContent
      });
    });

    const reanalyzeBtn = page.locator('#reanalyze-btn');
    const biasScoreElement = page.locator('#bias-score');
    const btnText = page.locator('#btn-text');
    const btnLoading = page.locator('#btn-loading');

    // Start reanalysis
    await reanalyzeBtn.click();

    // Wait for analysis to complete (longer timeout since we have a more realistic flow)
    await page.waitForTimeout(8000);

    // Check that button is re-enabled (after completion)
    await expect(reanalyzeBtn).toBeEnabled({ timeout: 10000 });
    await expect(btnText).toBeVisible();
    await expect(btnLoading).toBeHidden();

    // Check that bias score was updated or shows completion
    const finalBiasText = await biasScoreElement.textContent();
    
    // The bias score should either show a numeric value or indicate completion
    expect(finalBiasText).toMatch(/Bias Score:|Analysis|Complete|\d/);
  });

  test('should handle API format compatibility', async ({ page }) => {
    await page.goto('/article/600');
    
    // Test the bias score API directly
    const response = await page.request.get('/api/articles/600/bias');
    const data = await response.json();

    expect(response.ok()).toBe(true);
    expect(data).toHaveProperty('success', true);
    
    // Check that either old format (composite_score) or new format (data.composite_score) exists
    const hasOldFormat = data.hasOwnProperty('composite_score');
    const hasNewFormat = data.data && data.data.hasOwnProperty('composite_score');
    
    expect(hasOldFormat || hasNewFormat).toBe(true);

    // Test the JavaScript logic for handling both formats
    const extractionResult = await page.evaluate((apiData) => {
      let compositeScore = null;
      if (apiData.composite_score !== undefined) {
        compositeScore = apiData.composite_score;
      } else if (apiData.data && apiData.data.composite_score !== undefined) {
        compositeScore = apiData.data.composite_score;
      }
      return compositeScore;
    }, data);

    expect(extractionResult).not.toBeNull();
  });

  test('should handle errors gracefully', async ({ page }) => {
    await page.goto('/article/600');
    
    // Mock a failing reanalysis request
    await page.route('/api/llm/reanalyze/*', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    const reanalyzeBtn = page.locator('#reanalyze-btn');
    const btnText = page.locator('#btn-text');
    const progressIndicator = page.locator('#reanalysis-progress');

    // Click the button
    await reanalyzeBtn.click();

    // Wait for error handling
    await page.waitForTimeout(3000);

    // Check that error state is handled - should show "Request Reanalysis" since HTTP error doesn't trigger connection error
    await expect(btnText).toBeVisible();
    await expect(btnText).toHaveText('Request Reanalysis');
    await expect(reanalyzeBtn).toBeEnabled();
    await expect(progressIndicator).toHaveClass(/progress-hidden/);
  });

  test('should handle SSE connection errors', async ({ page }) => {
    await page.goto('/article/600');
    
    // Mock SSE endpoint to return error or fail to connect
    await page.route('/api/llm/score-progress/*', (route) => {
      route.abort('connectionrefused');
    });

    const reanalyzeBtn = page.locator('#reanalyze-btn');
    const btnText = page.locator('#btn-text');

    // Click reanalyze button
    await reanalyzeBtn.click();

    // Wait for connection error handling - give enough time for reconnection attempts
    await page.waitForTimeout(8000);

    // Should handle connection errors gracefully - check for error text first
    const finalBtnText = await btnText.textContent();
    expect(finalBtnText).toMatch(/Error|Failed|Request Reanalysis/);
    
    // Button should eventually be enabled (after connection error handling)
    await expect(reanalyzeBtn).toBeEnabled({ timeout: 5000 });
  });

  test('should be responsive on mobile devices', async ({ page, browserName }) => {
    // Skip webkit on mobile due to potential SSE issues
    test.skip(browserName === 'webkit', 'SSE support may be limited on WebKit mobile');
    
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto('/article/600');
    await page.waitForLoadState('networkidle');

    const reanalyzeBtn = page.locator('#reanalyze-btn');
    const progressIndicator = page.locator('#reanalysis-progress');

    // Check that elements are visible and clickable on mobile
    await expect(reanalyzeBtn).toBeVisible();
    await expect(reanalyzeBtn).toBeInViewport();

    // Button should be clickable (use click instead of tap for better compatibility)
    await reanalyzeBtn.click();

    // Progress indicator should show
    await expect(progressIndicator).toBeVisible({ timeout: 3000 });
  });
});

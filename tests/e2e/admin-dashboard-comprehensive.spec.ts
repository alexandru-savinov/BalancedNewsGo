import { test, expect } from '@playwright/test';

test.describe('Admin Dashboard Comprehensive E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to admin dashboard
    await page.goto('http://localhost:8080/admin');
    
    // Wait for page to load
    await page.waitForLoadState('networkidle');
  });

  test('should load admin dashboard with all sections', async ({ page }) => {
    // Check page title
    await expect(page).toHaveTitle(/Admin Dashboard/);
    
    // Check main sections are present
    await expect(page.locator('h1')).toContainText('Admin Dashboard');
    
    // Check all admin sections exist
    await expect(page.locator('h2:has-text("Feed Management")')).toBeVisible();
    await expect(page.locator('h2:has-text("Analysis Control")')).toBeVisible();
    await expect(page.locator('h2:has-text("Database Management")')).toBeVisible();
    await expect(page.locator('h2:has-text("System Monitoring")')).toBeVisible();
  });

  test('should handle feed management operations', async ({ page }) => {
    // Test refresh feeds button
    const refreshButton = page.locator('button:has-text("Refresh Feeds")');
    await expect(refreshButton).toBeVisible();
    
    // Click refresh feeds and handle alert
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Feed refresh');
      await dialog.accept();
    });
    
    await refreshButton.click();
    
    // Test reset feed errors button
    const resetButton = page.locator('button:has-text("Reset Feed Errors")');
    await expect(resetButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Feed errors');
      await dialog.accept();
    });
    
    await resetButton.click();
  });

  test('should handle analysis control operations', async ({ page }) => {
    // Test reanalyze recent button
    const reanalyzeButton = page.locator('button:has-text("Reanalyze Recent")');
    await expect(reanalyzeButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Reanalysis');
      await dialog.accept();
    });
    
    await reanalyzeButton.click();
    
    // Test clear analysis errors button
    const clearButton = page.locator('button:has-text("Clear Analysis Errors")');
    await expect(clearButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Analysis errors');
      await dialog.accept();
    });
    
    await clearButton.click();
    
    // Test validate scores button
    const validateButton = page.locator('button:has-text("Validate Scores")');
    await expect(validateButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Score validation');
      await dialog.accept();
    });
    
    await validateButton.click();
  });

  test('should handle database management operations', async ({ page }) => {
    // Test optimize database button
    const optimizeButton = page.locator('button:has-text("Optimize Database")');
    await expect(optimizeButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Database optimization');
      await dialog.accept();
    });
    
    await optimizeButton.click();
    
    // Test export data button
    const exportButton = page.locator('button:has-text("Export Data")');
    await expect(exportButton).toBeVisible();
    
    // Click export and wait for download
    const downloadPromise = page.waitForEvent('download');
    await exportButton.click();
    const download = await downloadPromise;
    
    // Verify download
    expect(download.suggestedFilename()).toContain('.csv');
    
    // Test cleanup old data button
    const cleanupButton = page.locator('button:has-text("Cleanup Old Articles")');
    await expect(cleanupButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Cleanup');
      await dialog.accept();
    });
    
    await cleanupButton.click();
  });

  test('should handle system monitoring operations', async ({ page }) => {
    // Test view metrics button
    const metricsButton = page.locator('a:has-text("View Metrics")');
    await expect(metricsButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Metrics');
      await dialog.accept();
    });
    
    await metricsButton.click();
    
    // Test view logs button
    const logsButton = page.locator('a:has-text("View Logs")');
    await expect(logsButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Logs');
      await dialog.accept();
    });
    
    await logsButton.click();
    
    // Test health check button
    const healthButton = page.locator('button:has-text("Run Health Check")');
    await expect(healthButton).toBeVisible();
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Health check');
      await dialog.accept();
    });
    
    await healthButton.click();
  });

  test('should handle error scenarios gracefully', async ({ page }) => {
    // Test with server potentially unavailable
    // This tests the frontend error handling
    
    // Mock a failed request
    await page.route('**/api/admin/**', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: {
            code: 'INTERNAL_ERROR',
            message: 'Test error'
          }
        })
      });
    });
    
    const refreshButton = page.locator('button:has-text("Refresh Feeds")');
    
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Error');
      await dialog.accept();
    });
    
    await refreshButton.click();
  });

  test('should display system status information', async ({ page }) => {
    // Check if system status elements are present
    const statusElements = [
      'Server Status',
      'Database Status', 
      'LLM Service Status',
      'RSS Service Status'
    ];
    
    for (const status of statusElements) {
      await expect(page.locator(`text=${status}`)).toBeVisible();
    }
  });

  test('should have responsive design', async ({ page }) => {
    // Test mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await expect(page.locator('h1')).toBeVisible();
    
    // Test tablet viewport
    await page.setViewportSize({ width: 768, height: 1024 });
    await expect(page.locator('h1')).toBeVisible();
    
    // Test desktop viewport
    await page.setViewportSize({ width: 1920, height: 1080 });
    await expect(page.locator('h1')).toBeVisible();
  });

  test('should handle concurrent operations', async ({ page }) => {
    // Test multiple button clicks in quick succession
    const buttons = [
      page.locator('button:has-text("Refresh Feeds")'),
      page.locator('button:has-text("Reset Feed Errors")')
    ];
    
    // Set up dialog handlers
    page.on('dialog', async dialog => {
      await dialog.accept();
    });
    
    // Click buttons rapidly
    await Promise.all(buttons.map(button => button.click()));
    
    // Verify no JavaScript errors occurred
    const errors: string[] = [];
    page.on('pageerror', error => {
      errors.push(error.message);
    });
    
    await page.waitForTimeout(1000);
    expect(errors).toHaveLength(0);
  });

  test('should maintain state during navigation', async ({ page }) => {
    // Navigate away and back
    await page.goto('http://localhost:8080/');
    await page.goto('http://localhost:8080/admin');
    
    // Verify page loads correctly
    await expect(page.locator('h1')).toContainText('Admin Dashboard');
    await expect(page.locator('button:has-text("Refresh Feeds")')).toBeVisible();
  });
});

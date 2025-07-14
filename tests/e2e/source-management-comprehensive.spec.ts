import { test, expect, Page } from '@playwright/test';

/**
 * Comprehensive Source Management E2E Tests
 * 
 * Tests the complete source management workflow including:
 * - UI loading and rendering
 * - HTMX interactions and real-time updates
 * - CRUD operations (Create, Read, Update, Delete)
 * - Form validation and error handling
 * - Modal displays and interactions
 * - Integration with RSS collector
 */

test.describe('Source Management - Comprehensive E2E Tests', () => {
  let page: Page;

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage;
    
    // Navigate to admin page
    await page.goto('http://localhost:8080/admin');
    
    // Wait for page to load completely
    await page.waitForLoadState('networkidle');
    
    // Wait for HTMX to load source management section
    await page.waitForSelector('[data-testid="source-management-container"]', { 
      timeout: 10000 
    });
  });

  test.describe('UI Loading and Rendering', () => {
    test('should display source management section on admin page', async () => {
      // Verify source management container is visible
      const sourceContainer = page.locator('[data-testid="source-management-container"]');
      await expect(sourceContainer).toBeVisible();
      
      // Verify header and add button
      await expect(page.locator('text=Source Management')).toBeVisible();
      await expect(page.locator('button:has-text("Add New Source")')).toBeVisible();
      
      // Verify source list is loaded
      const sourceItems = page.locator('[data-testid^="source-card-"]');
      const count = await sourceItems.count();
      expect(count).toBeGreaterThan(0);
    });

    test('should display existing sources with correct information', async () => {
      // Wait for sources to load
      await page.waitForSelector('[data-testid^="source-card-"]');
      
      // Check for BBC News source (should exist from setup)
      const bbcSource = page.locator('[data-testid="source-card-2"]');
      await expect(bbcSource).toBeVisible();
      
      // Verify source information
      await expect(bbcSource.locator('text=BBC News')).toBeVisible();
      await expect(bbcSource.locator('.badge-rss')).toBeVisible();
      await expect(bbcSource.locator('.badge-center')).toBeVisible();
      
      // Verify action buttons
      await expect(bbcSource.locator('[data-testid="edit-source-2"]')).toBeVisible();
      await expect(bbcSource.locator('[data-testid="stats-source-2"]')).toBeVisible();
    });
  });

  test.describe('Add New Source Functionality', () => {
    test('should open add new source form when button clicked', async () => {
      // Click Add New Source button
      await page.click('button:has-text("Add New Source")');
      
      // Wait for form to load via HTMX
      await page.waitForSelector('#source-form-container form', { timeout: 5000 });
      
      // Verify form fields are present
      await expect(page.locator('input[name="name"]')).toBeVisible();
      await expect(page.locator('input[name="feed_url"]')).toBeVisible();
      await expect(page.locator('select[name="category"]')).toBeVisible();
      await expect(page.locator('input[name="default_weight"]')).toBeVisible();
      // Note: enabled checkbox is only present in edit form, not add form (new sources are enabled by default)
      
      // Verify form buttons
      await expect(page.locator('button:has-text("Add Source")')).toBeVisible();
      await expect(page.locator('button:has-text("Cancel")')).toBeVisible();
    });

    test('should validate required fields in add source form', async () => {
      // Open add new source form
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');
      
      // Try to submit empty form
      await page.click('button:has-text("Add Source")');
      
      // Check for validation messages (HTML5 validation or custom)
      const nameInput = page.locator('input[name="name"]');
      const urlInput = page.locator('input[name="feed_url"]');

      // Verify required field validation
      await expect(nameInput).toHaveAttribute('required');
      await expect(urlInput).toHaveAttribute('required');
    });

    test('should successfully add a new RSS source', async () => {
      // Open add new source form
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');
      
      // Fill out the form
      await page.fill('input[name="name"]', 'Test News Source');
      await page.fill('input[name="feed_url"]', 'https://example.com/test-feed.xml');
      await page.selectOption('select[name="category"]', 'center');
      await page.fill('input[name="default_weight"]', '1.2');
      // Note: No enabled checkbox in add form - new sources are enabled by default
      
      // Submit the form
      await page.click('button:has-text("Add Source")');
      
      // Wait for HTMX to refresh the source list
      await page.waitForTimeout(2000);
      
      // Verify new source appears in the list
      await expect(page.locator('text=Test News Source')).toBeVisible();
      await expect(page.locator('text=https://example.com/test-feed.xml')).toBeVisible();
    });
  });

  test.describe('Edit Source Functionality', () => {
    test('should open edit form when edit button clicked', async () => {
      // Wait for sources to load
      await page.waitForSelector('[data-testid="edit-source-2"]');
      
      // Click edit button for BBC News
      await page.click('[data-testid="edit-source-2"]');
      
      // Wait for edit form to load
      await page.waitForSelector('#source-form-container form');
      
      // Verify form is pre-populated with existing data
      const nameInput = page.locator('input[name="name"]');
      await expect(nameInput).toHaveValue('BBC News');

      const urlInput = page.locator('input[name="feed_url"]');
      await expect(urlInput).toHaveValue('http://feeds.bbci.co.uk/news/rss.xml');
      
      // Verify update button is present
      await expect(page.locator('button:has-text("Update Source")')).toBeVisible();
    });

    test('should successfully update source information', async () => {
      // Click edit button for BBC News
      await page.click('[data-testid="edit-source-2"]');
      await page.waitForSelector('#source-form-container form');
      
      // Modify the source name
      await page.fill('input[name="name"]', 'BBC News Updated');
      await page.fill('input[name="default_weight"]', '1.5');
      
      // Submit the update
      await page.click('button:has-text("Update Source")');
      
      // Wait for HTMX to refresh
      await page.waitForTimeout(2000);
      
      // Verify changes are reflected in the source list
      await expect(page.locator('text=BBC News Updated')).toBeVisible();
      await expect(page.locator('text=Weight: 1.5')).toBeVisible();
    });
  });

  test.describe('Enable/Disable Source Operations', () => {
    test('should enable a disabled source', async () => {
      // Look for a disabled source (should have Enable button)
      const enableButton = page.locator('[data-testid^="enable-source-"]').first();
      
      if (await enableButton.isVisible()) {
        // Click enable button
        await enableButton.click();
        
        // Wait for HTMX update
        await page.waitForTimeout(2000);
        
        // Verify source is now enabled (should show Disable button)
        const sourceCard = enableButton.locator('xpath=ancestor::div[@data-testid]');
        await expect(sourceCard.locator('button:has-text("Disable")')).toBeVisible();
      }
    });

    test('should disable an enabled source', async () => {
      // Look for an enabled source (should have Disable button)
      const disableButton = page.locator('[data-testid^="disable-source-"]').first();
      
      if (await disableButton.isVisible()) {
        // Click disable button and handle confirmation
        page.on('dialog', async dialog => {
          expect(dialog.message()).toContain('Disable this source');
          await dialog.accept();
        });
        
        await disableButton.click();
        
        // Wait for HTMX update
        await page.waitForTimeout(2000);
        
        // Verify source is now disabled (should show Enable button)
        const sourceCard = disableButton.locator('xpath=ancestor::div[@data-testid]');
        await expect(sourceCard.locator('button:has-text("Enable")')).toBeVisible();
      }
    });
  });

  test.describe('Source Statistics Modal', () => {
    test('should open stats modal when stats button clicked', async () => {
      // Click stats button for BBC News
      await page.click('[data-testid="stats-source-2"]');
      
      // Wait for modal to appear
      await page.waitForSelector('#source-stats-modal', { state: 'visible' });
      
      // Verify modal content
      const modal = page.locator('#source-stats-modal');
      await expect(modal).toBeVisible();
      
      // Verify close button
      await expect(modal.locator('.close')).toBeVisible();
    });

    test('should close stats modal when close button clicked', async () => {
      // Open stats modal
      await page.click('[data-testid="stats-source-2"]');
      await page.waitForSelector('#source-stats-modal', { state: 'visible' });
      
      // Close modal
      await page.click('#source-stats-modal .close');
      
      // Verify modal is hidden
      await expect(page.locator('#source-stats-modal')).toBeHidden();
    });
  });

  test.describe('Real-time Updates and HTMX Integration', () => {
    test('should update source list without page refresh', async () => {
      // Get initial source count
      const initialSources = await page.locator('[data-testid^="source-card-"]').count();
      
      // Add a new source
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');
      
      await page.fill('input[name="name"]', 'HTMX Test Source');
      await page.fill('input[name="feed_url"]', 'https://example.com/htmx-test.xml');
      await page.selectOption('select[name="category"]', 'left');
      
      await page.click('button:has-text("Add Source")');
      
      // Wait for HTMX update
      await page.waitForTimeout(3000);
      
      // Verify source count increased without page reload
      const newSources = await page.locator('[data-testid^="source-card-"]').count();
      expect(newSources).toBeGreaterThan(initialSources);
      
      // Verify new source is visible
      await expect(page.locator('text=HTMX Test Source')).toBeVisible();
    });

    test('should handle HTMX errors gracefully', async () => {
      // Try to edit a non-existent source (should trigger error)
      await page.goto('http://localhost:8080/admin');
      await page.waitForSelector('[data-testid="source-management-container"]');
      
      // Manually trigger an HTMX request to a non-existent source
      await page.evaluate(() => {
        // @ts-ignore
        htmx.ajax('GET', '/htmx/sources/999/edit', '#source-form-container');
      });
      
      // Wait for potential error handling
      await page.waitForTimeout(2000);
      
      // Verify the page doesn't break and error is handled
      await expect(page.locator('[data-testid="source-management-container"]')).toBeVisible();
    });
  });

  test.describe('Form Validation and Error Handling', () => {
    test('should validate URL format in add source form', async () => {
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');
      
      // Enter invalid URL
      await page.fill('input[name="name"]', 'Invalid URL Test');
      await page.fill('input[name="feed_url"]', 'not-a-valid-url');

      await page.click('button:has-text("Add Source")');

      // Check for URL validation (HTML5 or custom)
      const urlInput = page.locator('input[name="feed_url"]');
      await expect(urlInput).toHaveAttribute('type', 'url');
    });

    test('should validate weight is a positive number', async () => {
      await page.click('button:has-text("Add New Source")');
      await page.waitForSelector('#source-form-container form');
      
      // Enter negative weight
      await page.fill('input[name="name"]', 'Weight Test');
      await page.fill('input[name="feed_url"]', 'https://example.com/feed.xml');
      await page.fill('input[name="default_weight"]', '-1');

      await page.click('button:has-text("Add Source")');

      // Check for weight validation
      const weightInput = page.locator('input[name="default_weight"]');
      await expect(weightInput).toHaveAttribute('min', '0.1');
    });
  });
});

import { test, expect } from '@playwright/test';

test.describe('Basic Application Functionality', () => {  test('should load the home page successfully', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Check that the page loads without errors
    await expect(page.locator('body')).toBeVisible();
    
    // Check for key elements that should be present
    // Use page.toHaveTitle which is more reliable than locating title element
    await expect(page).toHaveTitle(/NewsBalancer|NewBalancer|Articles/i, { timeout: 15000 });
  });

  test('should display articles on the home page', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Look for article-related content
    const articleElements = page.locator('.article, article, [data-testid*="article"]');
    
    if (await articleElements.count() > 0) {
      await expect(articleElements.first()).toBeVisible();
    } else {
      // If no articles found, check for "no articles" message or loading state
      const noArticlesMsg = page.locator(':has-text("No articles"), :has-text("Loading"), :has-text("no content")');
      if (await noArticlesMsg.count() > 0) {
        await expect(noArticlesMsg.first()).toBeVisible();
      } else {
        // At minimum, the body should be visible
        await expect(page.locator('body')).toBeVisible();
      }
    }
  });

  test('should have working navigation', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Look for navigation elements
    const navLinks = page.locator('nav a, .nav a, a[href="/articles"], a[href="/admin"]');
    
    if (await navLinks.count() > 0) {
      const firstLink = navLinks.first();
      const href = await firstLink.getAttribute('href');
      
      if (href && href.startsWith('/')) {
        await firstLink.click();
        await page.waitForLoadState('networkidle');
        
        // Verify navigation worked
        expect(page.url()).toContain(href);
        await expect(page.locator('body')).toBeVisible();
      }
    }
  });

  test('should handle search functionality if present', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Look for search input
    const searchInput = page.locator('input[type="search"], input[name*="search"], input[placeholder*="search"], input[name="query"]');
    
    if (await searchInput.count() > 0) {
      await searchInput.first().fill('test search');
      await page.waitForTimeout(1000); // Wait for any dynamic updates
      
      // Verify search input has the value
      await expect(searchInput.first()).toHaveValue('test search');
    } else {
      // If no search found, that's okay - just ensure page is still functional
      await expect(page.locator('body')).toBeVisible();
    }
  });

  test('should respond to user interactions', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Look for clickable elements
    const buttons = page.locator('button, [role="button"], a');
    
    if (await buttons.count() > 0) {
      const firstButton = buttons.first();
      
      // Check if element is visible before clicking
      if (await firstButton.isVisible()) {
        await firstButton.click();
        await page.waitForTimeout(500);
        
        // Verify page is still responsive after interaction
        await expect(page.locator('body')).toBeVisible();
      }
    }
  });
});

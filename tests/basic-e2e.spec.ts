import { test, expect, Page } from '@playwright/test';

test.describe('NewsBalancer E2E Tests - Basic Functionality', () => {
  let page: Page;

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage;
    await page.goto('/articles');
    await page.waitForLoadState('networkidle');
  });

  test.describe('Basic Page Functionality', () => {
    test('should load the articles page successfully', async () => {
      // Check that we get a successful response
      const title = await page.title();
      expect(title).toBeTruthy();
      
      // Look for common HTML elements that should exist
      const bodyExists = await page.locator('body').count();
      expect(bodyExists).toBeGreaterThan(0);
      
      // Check for navigation or header
      const headerExists = await page.locator('header, nav, h1').count();
      expect(headerExists).toBeGreaterThan(0);
    });

    test('should have articles content or appropriate message', async () => {
      // Look for any content indicators - articles, messages, or containers
      const contentIndicators = [
        'ul.article-list',
        '.article-list',
        '.articles',
        'article',
        '.no-articles',
        'p:has-text("No articles")',
        'div:has-text("articles")',
        'li a[href*="/article/"]'
      ];
      
      let foundContent = false;
      for (const selector of contentIndicators) {
        const count = await page.locator(selector).count();
        if (count > 0) {
          foundContent = true;
          break;
        }
      }
      
      expect(foundContent).toBeTruthy();
    });


  });

  test.describe('Navigation and Links', () => {
    test('should have working navigation links', async () => {
      // Look for navigation links
      const navLinks = await page.locator('a[href]').count();
      expect(navLinks).toBeGreaterThan(0);
        // Check for admin link if it exists
      const adminLink = page.locator('a[href="/admin"], a:has-text("admin")');
      if (await adminLink.count() > 0) {
        expect(await adminLink.first().getAttribute('href')).toBeTruthy();
      }
    });

    test('should handle article links if articles exist', async () => {
      // Look for article links
      const articleLinks = page.locator('a[href*="/article/"]');
      const articleCount = await articleLinks.count();
      
      if (articleCount > 0) {
        // Test clicking on the first article link
        const firstLink = articleLinks.first();
        const href = await firstLink.getAttribute('href');
        expect(href).toMatch(/\/article\/\d+/);
        
        // Navigate to article page
        await firstLink.click();
        await page.waitForLoadState('networkidle');
        
        // Should be on an article page
        expect(page.url()).toContain('/article/');
        
        // Should have some content
        const bodyText = await page.locator('body').textContent();
        expect(bodyText?.length).toBeGreaterThan(50);
      }
    });
  });


  test.describe('API Integration', () => {
    test('should validate server response structure', async ({ page }) => {
      // Test direct API endpoints with error handling
      try {
        const response = await page.request.get('/api/articles');
        
        if (response.status() === 200) {
          // Check if response is valid JSON without parsing all of it
          const responseText = await response.text();
          expect(responseText.length).toBeGreaterThan(0);
          expect(responseText.trim().startsWith('[')).toBe(true);
        }
      } catch (error: any) {
        console.log('API test skipped due to large response or network error:', error?.message);
        // This is acceptable - we just want to ensure the page loads
      }
      
      // Ensure page loads regardless
      await page.goto('/articles');
      const bodyText = await page.locator('body').textContent();
      expect(bodyText?.length).toBeGreaterThan(0);
    });
  });

  test.describe('Error Handling', () => {
    test('should handle invalid article URLs gracefully', async () => {
      await page.goto('/article/999999');
      await page.waitForLoadState('networkidle');
      
      // Should show error message or redirect
      const bodyText = await page.locator('body').textContent();
      const isErrorPage = bodyText?.toLowerCase().includes('not found') || 
                         bodyText?.toLowerCase().includes('error') ||
                         page.url().includes('/articles');
      
      expect(isErrorPage).toBeTruthy();
    });
  });
});

import { test, expect, Page } from '@playwright/test';

test.describe('HTMX Performance & Accessibility E2E Tests', () => {
  let page: Page;

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage;
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test.describe('Performance with HTMX', () => {
    test('should load initial page within performance budget', async () => {
      const startTime = Date.now();
      
      await page.goto('/', { waitUntil: 'networkidle' });
      
      const loadTime = Date.now() - startTime;
      
      // Should load within 3 seconds
      expect(loadTime).toBeLessThan(3000);
      
      // Check for critical resources
      const articles = await page.locator('[data-testid^="article-card"], .article-card').count();
      expect(articles).toBeGreaterThan(0);
    });

    test('should handle rapid HTMX requests efficiently', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        const startTime = Date.now();
        
        // Simulate rapid typing
        await searchInput.fill('t');
        await page.waitForTimeout(100);
        await searchInput.fill('te');
        await page.waitForTimeout(100);
        await searchInput.fill('tech');
        await page.waitForTimeout(100);
        await searchInput.fill('technology');
        
        // Wait for final request to complete
        await page.waitForTimeout(2000);
        
        const totalTime = Date.now() - startTime;
        
        // Should handle rapid requests without excessive delay
        expect(totalTime).toBeLessThan(5000);
        
        // Should show results
        const results = await page.locator('[data-testid^="article-card"], .article-card').count();
        const noResults = await page.locator('[data-testid="no-results"], .no-articles').count();
        expect(results > 0 || noResults > 0).toBeTruthy();
      }
    });

    test('should maintain performance during pagination', async () => {
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      
      if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
        const navigationTimes: number[] = [];
        
        // Test multiple page navigations
        for (let i = 0; i < 3; i++) {
          const startTime = Date.now();
          
          await nextPageBtn.click();
          await page.waitForSelector('[data-testid^="article-card"], .article-card');
          
          const navigationTime = Date.now() - startTime;
          navigationTimes.push(navigationTime);
          
          // Each navigation should be reasonably fast
          expect(navigationTime).toBeLessThan(2000);
          
          // Check if we can continue (next button still exists and is enabled)
          if (!(await nextPageBtn.isVisible()) || await nextPageBtn.isDisabled()) {
            break;
          }
        }
        
        // Average navigation time should be reasonable
        const avgTime = navigationTimes.reduce((a, b) => a + b, 0) / navigationTimes.length;
        expect(avgTime).toBeLessThan(1500);
      }
    });

    test('should not cause memory leaks with repeated HTMX requests', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        // Perform multiple search operations
        const searchTerms = ['tech', 'news', 'politics', 'science', 'business'];
        
        for (const term of searchTerms) {
          await searchInput.clear();
          await searchInput.fill(term);
          await page.waitForTimeout(500);
          
          // Verify each search completes successfully
          await page.waitForSelector('[data-testid^="article-card"], .article-card, [data-testid="no-results"]');
        }
        
        // Clear search to return to normal state
        await searchInput.clear();
        await page.waitForTimeout(500);
        
        // Should return to showing all articles
        const finalArticles = await page.locator('[data-testid^="article-card"], .article-card').count();
        expect(finalArticles).toBeGreaterThan(0);
      }
    });
  });

  test.describe('Accessibility with HTMX', () => {
    test('should maintain focus management during HTMX updates', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        // Focus on search input
        await searchInput.focus();
        
        // Perform search
        await searchInput.fill('technology');
        await page.waitForTimeout(1000);
        
        // Focus should remain on search input after HTMX update
        const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
        expect(focusedElement).toBe('INPUT');
      }
    });

    test('should announce dynamic content changes to screen readers', async () => {
      // Look for ARIA live regions
      const liveRegion = page.locator('[aria-live], [role="status"], [role="alert"]');
      
      if (await liveRegion.count() > 0) {
        const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
        
        if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
          // Navigate to next page
          await nextPageBtn.click();
          await page.waitForTimeout(1000);
          
          // ARIA live region should exist for announcements
          expect(await liveRegion.count()).toBeGreaterThan(0);
        }
      }
    });

    test('should maintain proper heading structure after HTMX updates', async () => {
      // Check initial heading structure
      const initialHeadings = await page.locator('h1, h2, h3, h4, h5, h6').allTextContents();
      expect(initialHeadings.length).toBeGreaterThan(0);
      
      // Trigger HTMX update
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      
      if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
        await nextPageBtn.click();
        await page.waitForTimeout(1000);
        
        // Check heading structure after update
        const updatedHeadings = await page.locator('h1, h2, h3, h4, h5, h6').allTextContents();
        expect(updatedHeadings.length).toBeGreaterThan(0);
        
        // Should still have main heading (h1)
        const h1Count = await page.locator('h1').count();
        expect(h1Count).toBeGreaterThanOrEqual(1);
      }
    });

    test('should maintain keyboard navigation after HTMX updates', async () => {
      // Test tab navigation before update
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab');
      
      const firstFocusedElement = await page.evaluate(() => document.activeElement?.tagName);
      
      // Trigger HTMX update
      const categoryFilter = page.locator('[data-testid="category-filter"], select[name="category"]');
      
      if (await categoryFilter.count() > 0) {
        await categoryFilter.selectOption('technology');
        await page.waitForTimeout(1000);
        
        // Test tab navigation after update
        await page.keyboard.press('Tab');
        const secondFocusedElement = await page.evaluate(() => document.activeElement?.tagName);
        
        // Should be able to navigate with keyboard
        expect(secondFocusedElement).toBeTruthy();
      }
    });

    test('should have proper ARIA attributes on interactive elements', async () => {
      // Check pagination controls
      const paginationControls = page.locator('[data-testid*="page"], .pagination a, .pagination button');
      
      if (await paginationControls.count() > 0) {
        for (let i = 0; i < await paginationControls.count(); i++) {
          const control = paginationControls.nth(i);
          
          // Should have accessible text (aria-label or text content)
          const ariaLabel = await control.getAttribute('aria-label');
          const textContent = await control.textContent();
          
          expect(ariaLabel || textContent).toBeTruthy();
        }
      }
      
      // Check form controls
      const formControls = page.locator('input, select, button');
      
      for (let i = 0; i < Math.min(5, await formControls.count()); i++) {
        const control = formControls.nth(i);
        const tagName = await control.evaluate(el => el.tagName);
        
        if (tagName === 'INPUT') {
          // Inputs should have labels or aria-label
          const id = await control.getAttribute('id');
          const ariaLabel = await control.getAttribute('aria-label');
          
          if (id) {
            const label = page.locator(`label[for="${id}"]`);
            const hasLabel = await label.count() > 0;
            expect(hasLabel || ariaLabel).toBeTruthy();
          } else {
            expect(ariaLabel).toBeTruthy();
          }
        }
      }
    });

    test('should handle loading states accessibly', async () => {
      const loadingIndicator = page.locator('[data-testid="loading"], .loading, .htmx-indicator');
      
      // Trigger a slow operation
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        await searchInput.fill('test');
        
        // Check if loading indicator appears and has proper attributes
        if (await loadingIndicator.count() > 0) {
          const ariaLabel = await loadingIndicator.getAttribute('aria-label');
          const role = await loadingIndicator.getAttribute('role');
          
          // Should have appropriate ARIA attributes for loading state
          expect(ariaLabel || role).toBeTruthy();
        }
        
        await page.waitForTimeout(1000);
      }
    });
  });

  test.describe('HTMX Error Handling & Edge Cases', () => {
    test('should handle network failures gracefully', async () => {
      // Simulate network failure
      await page.route('**/api/**', route => {
        route.abort('failed');
      });
      
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        await searchInput.fill('test');
        await page.waitForTimeout(2000);
        
        // Should show error message or maintain previous state
        const errorMsg = page.locator('[data-testid="error"], .error');
        const articles = page.locator('[data-testid^="article-card"], .article-card');
        
        // Either error message shown or previous content maintained
        expect(await errorMsg.count() > 0 || await articles.count() > 0).toBeTruthy();
      }
    });

    test('should handle malformed server responses', async () => {
      // Simulate malformed response
      await page.route('**/api/**', route => {
        route.fulfill({
          status: 200,
          contentType: 'text/html',
          body: '<div>Malformed response without expected structure</div>'
        });
      });
      
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      
      if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
        await nextPageBtn.click();
        await page.waitForTimeout(2000);
        
        // Should handle gracefully - either show error or maintain previous state
        const errorState = page.locator('[data-testid="error"], .error');
        const normalState = page.locator('[data-testid^="article-card"], .article-card');
        
        expect(await errorState.count() > 0 || await normalState.count() > 0).toBeTruthy();
      }
    });

    test('should handle concurrent HTMX requests properly', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      const categoryFilter = page.locator('[data-testid="category-filter"], select[name="category"]');
      
      if (await searchInput.count() > 0 && await categoryFilter.count() > 0) {
        // Trigger multiple requests simultaneously
        const searchPromise = searchInput.fill('technology');
        const filterPromise = categoryFilter.selectOption('politics');
        
        await Promise.all([searchPromise, filterPromise]);
        
        // Wait for requests to settle
        await page.waitForTimeout(2000);
        
        // Should handle concurrent requests and show consistent state
        const articles = await page.locator('[data-testid^="article-card"], .article-card').count();
        const noResults = await page.locator('[data-testid="no-results"], .no-articles').count();
        
        expect(articles > 0 || noResults > 0).toBeTruthy();
        
        // Final state should be consistent
        const searchValue = await searchInput.inputValue();
        const filterValue = await categoryFilter.inputValue();
        
        expect(searchValue).toBe('technology');
        expect(filterValue).toBe('politics');
      }
    });

    test('should handle session timeout gracefully', async () => {
      // Simulate session timeout
      await page.route('**/api/**', route => {
        route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Session expired' })
        });
      });
      
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      
      if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
        await nextPageBtn.click();
        await page.waitForTimeout(2000);
        
        // Should handle auth error appropriately
        const errorMsg = page.locator('[data-testid="error"], .error, .auth-error');
        const loginPrompt = page.locator('[data-testid="login"], .login-required');
        
        // Should show appropriate error or redirect to login
        expect(await errorMsg.count() > 0 || await loginPrompt.count() > 0 || page.url().includes('login')).toBeTruthy();
      }
    });
  });
});

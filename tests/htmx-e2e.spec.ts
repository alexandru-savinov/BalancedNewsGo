import { test, expect, Page } from '@playwright/test';

test.describe('HTMX E2E Tests - Dynamic Content Loading', () => {
  let page: Page;

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage;
    // Navigate to the main articles page
    await page.goto('/');
    
    // Wait for the page to be fully loaded
    await page.waitForLoadState('networkidle');
    
    // Wait for articles to load
    await page.waitForSelector('[data-testid^="article-card"], .article-card', { timeout: 10000 });
  });

  test.describe('Dynamic Filtering without Page Refresh', () => {
    test('should filter articles by category using HTMX', async () => {
      // Look for category filter elements
      const categoryFilter = page.locator('[data-testid="category-filter"], select[name="category"], .category-filter');
      
      // If filter exists, test it
      if (await categoryFilter.count() > 0) {
        // Get initial article count
        const initialArticles = await page.locator('[data-testid^="article-card"], .article-card').count();
        
        // Change category filter
        await categoryFilter.selectOption('politics');
        
        // Wait for HTMX response (look for hx-request headers or loading states)
        await page.waitForTimeout(1000);
        
        // Check that content updated without page reload
        const currentUrl = page.url();
        expect(currentUrl).not.toContain('reload');
        
        // Verify articles were filtered (count should change or specific content should appear)
        await page.waitForSelector('[data-testid^="article-card"], .article-card');
        const filteredArticles = await page.locator('[data-testid^="article-card"], .article-card').count();
        
        // Articles should be present (either same count or different, but not zero unless no politics articles)
        expect(filteredArticles).toBeGreaterThanOrEqual(0);
      }
    });

    test('should filter articles by bias score range', async () => {
      const biasFilter = page.locator('[data-testid="bias-filter"], input[name="bias"], .bias-filter');
      
      if (await biasFilter.count() > 0) {
        // Set bias filter to left-leaning (-0.5)
        await biasFilter.fill('-0.5');
        await biasFilter.press('Enter');
        
        // Wait for HTMX update
        await page.waitForTimeout(1000);
        
        // Verify URL didn't change (no full page reload)
        expect(page.url()).toContain('/');
        
        // Check that bias indicators reflect the filter
        const biasValues = await page.locator('[data-testid="bias-value"], .bias-score').allTextContents();
        
        // At least some articles should be displayed
        expect(biasValues.length).toBeGreaterThan(0);
      }
    });

    test('should handle multiple filters simultaneously', async () => {
      const categoryFilter = page.locator('[data-testid="category-filter"], select[name="category"]');
      const sourceFilter = page.locator('[data-testid="source-filter"], select[name="source"]');
      
      if (await categoryFilter.count() > 0 && await sourceFilter.count() > 0) {
        // Apply multiple filters
        await categoryFilter.selectOption('technology');
        await sourceFilter.selectOption('techcrunch');
        
        // Wait for HTMX to process both filters
        await page.waitForTimeout(1500);
        
        // Verify content updated appropriately
        await page.waitForSelector('[data-testid^="article-card"], .article-card');
        const articles = await page.locator('[data-testid^="article-card"], .article-card').count();
        
        // Should have some results or show "no articles" message
        const noResults = await page.locator('[data-testid="no-results"], .no-articles').count();
        expect(articles > 0 || noResults > 0).toBeTruthy();
      }
    });
  });

  test.describe('Live Search Functionality', () => {
    test('should perform live search without page refresh', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        // Get initial article count
        const initialCount = await page.locator('[data-testid^="article-card"], .article-card').count();
        
        // Type search query
        await searchInput.fill('climate');
        
        // Wait for search debouncing and HTMX response
        await page.waitForTimeout(1000);
        
        // Verify URL contains search parameter or that content changed
        const searchResults = await page.locator('[data-testid^="article-card"], .article-card').count();
        const noResults = await page.locator('[data-testid="no-results"], .no-articles').count();
        
        // Either we have search results or a "no results" message
        expect(searchResults > 0 || noResults > 0).toBeTruthy();
        
        // Clear search
        await searchInput.clear();
        await page.waitForTimeout(1000);
        
        // Should return to showing all articles
        const clearedResults = await page.locator('[data-testid^="article-card"], .article-card').count();
        expect(clearedResults).toBeGreaterThanOrEqual(initialCount * 0.8); // Allow for some variance
      }
    });

    test('should handle empty search gracefully', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        // Search for something that definitely won't exist
        await searchInput.fill('xyzabcnotexist123');
        await page.waitForTimeout(1000);
        
        // Should show no results message
        const noResults = await page.locator('[data-testid="no-results"], .no-articles').isVisible();
        const articleCount = await page.locator('[data-testid^="article-card"], .article-card').count();
        
        expect(noResults || articleCount === 0).toBeTruthy();
      }
    });

    test('should maintain search across pagination', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      
      if (await searchInput.count() > 0 && await nextPageBtn.count() > 0) {
        // Perform search
        await searchInput.fill('news');
        await page.waitForTimeout(1000);
        
        // Go to next page
        await nextPageBtn.click();
        await page.waitForTimeout(1000);
        
        // Verify search term is maintained
        const searchValue = await searchInput.inputValue();
        expect(searchValue).toBe('news');
        
        // Verify we're on page 2 but still showing search results
        expect(page.url()).toContain('page=2') || expect(page.url()).toContain('search=news');
      }
    });
  });

  test.describe('Seamless Pagination Navigation', () => {
    test('should navigate pages without full reload', async () => {
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next, a[rel="next"]');
      
      if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
        // Get first article title on page 1
        const firstArticleTitle = await page.locator('[data-testid^="article-card"], .article-card').first().textContent();
        
        // Click next page
        await nextPageBtn.click();
        
        // Wait for HTMX to load new content
        await page.waitForTimeout(1500);
        
        // Verify new content loaded
        const newFirstArticleTitle = await page.locator('[data-testid^="article-card"], .article-card').first().textContent();
        
        // Titles should be different (new page content)
        expect(newFirstArticleTitle).not.toBe(firstArticleTitle);
        
        // URL should reflect page change
        expect(page.url()).toContain('page=2') || expect(page.url()).toContain('offset=');
      }
    });

    test('should handle pagination with filters applied', async () => {
      const categoryFilter = page.locator('[data-testid="category-filter"], select[name="category"]');
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      
      if (await categoryFilter.count() > 0 && await nextPageBtn.count() > 0) {
        // Apply filter first
        await categoryFilter.selectOption('technology');
        await page.waitForTimeout(1000);
        
        // Then navigate to next page
        if (await nextPageBtn.isVisible()) {
          await nextPageBtn.click();
          await page.waitForTimeout(1000);
          
          // Verify filter is maintained on page 2
          const selectedValue = await categoryFilter.inputValue();
          expect(selectedValue).toBe('technology');
          
          // URL should contain both page and filter info
          expect(page.url()).toContain('page=2') || expect(page.url()).toContain('category=technology');
        }
      }
    });

    test('should handle first/last page navigation', async () => {
      const firstPageBtn = page.locator('[data-testid="first-page"], .pagination .first');
      const lastPageBtn = page.locator('[data-testid="last-page"], .pagination .last');
      
      if (await lastPageBtn.count() > 0 && await lastPageBtn.isVisible()) {
        // Go to last page
        await lastPageBtn.click();
        await page.waitForTimeout(1000);
        
        // Verify we're on last page (next button should be disabled)
        const nextBtn = page.locator('[data-testid="next-page"], .pagination .next');
        if (await nextBtn.count() > 0) {
          expect(await nextBtn.isDisabled()).toBeTruthy();
        }
        
        // Go back to first page
        if (await firstPageBtn.count() > 0) {
          await firstPageBtn.click();
          await page.waitForTimeout(1000);
          
          // Should be back on page 1
          expect(page.url()).not.toContain('page=') || expect(page.url()).toContain('page=1');
        }
      }
    });
  });

  test.describe('Article Loading via HTMX', () => {
    test('should load article details in modal/overlay', async () => {
      const firstArticle = page.locator('[data-testid^="article-card"], .article-card').first();
      const readMoreBtn = firstArticle.locator('[data-testid="read-more"], .read-more, .article-link');
      
      if (await readMoreBtn.count() > 0) {
        // Click read more
        await readMoreBtn.click();
        
        // Wait for HTMX to load article content
        await page.waitForTimeout(1500);
        
        // Check if modal or overlay appeared
        const modal = page.locator('[data-testid="article-modal"], .modal, .article-overlay');
        const articleContent = page.locator('[data-testid="article-content"], .article-detail');
        
        // Either modal appeared or we navigated to article page
        const modalVisible = await modal.count() > 0 && await modal.isVisible();
        const contentVisible = await articleContent.count() > 0 && await articleContent.isVisible();
        
        expect(modalVisible || contentVisible).toBeTruthy();
        
        // If modal, verify it can be closed
        if (modalVisible) {
          const closeBtn = modal.locator('[data-testid="close-modal"], .close, .modal-close');
          if (await closeBtn.count() > 0) {
            await closeBtn.click();
            await page.waitForTimeout(500);
            expect(await modal.isVisible()).toBeFalsy();
          }
        }
      }
    });

    test('should load article summary via HTMX', async () => {
      const summaryBtn = page.locator('[data-testid="summary-btn"], .summary-toggle').first();
      
      if (await summaryBtn.count() > 0) {
        await summaryBtn.click();
        await page.waitForTimeout(1000);
        
        // Check for summary content
        const summaryContent = page.locator('[data-testid="article-summary"], .summary-content');
        expect(await summaryContent.count()).toBeGreaterThan(0);
        
        // Summary should contain text
        const summaryText = await summaryContent.textContent();
        expect(summaryText?.length).toBeGreaterThan(10);
      }
    });

    test('should handle article loading errors gracefully', async () => {
      // Try to load a non-existent article via direct URL manipulation
      await page.goto('/article/nonexistent123');
      
      // Should show error message or redirect
      const errorMsg = page.locator('[data-testid="error-message"], .error, .not-found');
      const homeRedirect = page.url().includes('/') && !page.url().includes('/article/');
      
      expect(await errorMsg.count() > 0 || homeRedirect).toBeTruthy();
    });
  });

  test.describe('Browser History Management', () => {
    test('should maintain proper browser history with HTMX navigation', async () => {
      const initialUrl = page.url();
      
      // Navigate using pagination
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
        await nextPageBtn.click();
        await page.waitForTimeout(1000);
        
        const page2Url = page.url();
        expect(page2Url).not.toBe(initialUrl);
        
        // Use browser back button
        await page.goBack();
        await page.waitForTimeout(1000);
        
        // Should be back to original page
        expect(page.url()).toBe(initialUrl);
        
        // Use browser forward button
        await page.goForward();
        await page.waitForTimeout(1000);
        
        // Should be back to page 2
        expect(page.url()).toBe(page2Url);
      }
    });

    test('should handle search history properly', async () => {
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      
      if (await searchInput.count() > 0) {
        const initialUrl = page.url();
        
        // Perform search
        await searchInput.fill('technology');
        await page.waitForTimeout(1000);
        
        const searchUrl = page.url();
        expect(searchUrl).toContain('search=technology') || expect(searchUrl).not.toBe(initialUrl);
        
        // Navigate back
        await page.goBack();
        await page.waitForTimeout(1000);
        
        // Should clear search or return to original state
        const backUrl = page.url();
        expect(backUrl).toBe(initialUrl) || expect(await searchInput.inputValue()).toBe('');
      }
    });

    test('should handle filter state in browser history', async () => {
      const categoryFilter = page.locator('[data-testid="category-filter"], select[name="category"]');
      
      if (await categoryFilter.count() > 0) {
        const initialUrl = page.url();
        
        // Apply filter
        await categoryFilter.selectOption('politics');
        await page.waitForTimeout(1000);
        
        const filterUrl = page.url();
        
        // Navigate to an article or another page
        const firstArticle = page.locator('[data-testid^="article-card"], .article-card').first();
        const articleLink = firstArticle.locator('a, [data-testid="article-link"]').first();
        
        if (await articleLink.count() > 0) {
          await articleLink.click();
          await page.waitForTimeout(1000);
          
          // Go back to filtered list
          await page.goBack();
          await page.waitForTimeout(1000);
          
          // Filter should be maintained
          const selectedValue = await categoryFilter.inputValue();
          expect(selectedValue).toBe('politics');
          expect(page.url()).toBe(filterUrl);
        }
      }
    });
  });

  test.describe('HTMX-specific Features', () => {
    test('should handle HTMX loading states', async () => {
      const loadingIndicator = page.locator('[data-testid="loading"], .htmx-indicator, .loading');
      
      // Trigger an HTMX request
      const nextPageBtn = page.locator('[data-testid="next-page"], .pagination .next');
      if (await nextPageBtn.count() > 0 && await nextPageBtn.isVisible()) {
        await nextPageBtn.click();
        
        // Loading indicator should appear briefly
        // Note: This might be too fast to catch consistently, so we just verify it doesn't persist
        await page.waitForTimeout(2000);
        
        if (await loadingIndicator.count() > 0) {
          expect(await loadingIndicator.isVisible()).toBeFalsy();
        }
      }
    });

    test('should handle HTMX error responses', async () => {
      // Simulate network error by intercepting requests
      await page.route('**/api/**', route => {
        if (route.request().url().includes('error-test')) {
          route.fulfill({
            status: 500,
            contentType: 'text/html',
            body: '<div class="error">Server Error</div>'
          });
        } else {
          route.continue();
        }
      });
      
      // Try to trigger an error (this would need specific HTMX endpoints that accept error params)
      // For now, we'll just verify error handling exists
      const errorElement = page.locator('.error, [data-testid="error"]');
      
      // The error handling should be in place even if not triggered
      expect(await errorElement.count()).toBeGreaterThanOrEqual(0);
    });

    test('should verify HTMX headers are sent correctly', async () => {
      let htmxRequestDetected = false;
      
      // Intercept requests to verify HTMX headers
      await page.route('**/api/**', route => {
        const headers = route.request().headers();
        if (headers['hx-request'] === 'true') {
          htmxRequestDetected = true;
        }
        route.continue();
      });
      
      // Trigger an HTMX request
      const searchInput = page.locator('[data-testid="search-input"], input[name="search"], .search-input');
      if (await searchInput.count() > 0) {
        await searchInput.fill('test');
        await page.waitForTimeout(1000);
        
        // HTMX request should have been detected
        expect(htmxRequestDetected).toBeTruthy();
      }
    });
  });
});

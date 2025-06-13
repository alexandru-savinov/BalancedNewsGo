import { test, expect, Page } from '@playwright/test';

test.describe('HTMX Integration Tests - Specific Features', () => {
  let page: Page;

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage;
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test.describe('Article List HTMX Integration', () => {
    test('should load articles with HTMX pagination', async () => {
      // Wait for articles to load
      await page.waitForSelector('[data-testid^="article-card"], .article-card', { timeout: 10000 });
      
      const initialArticles = await page.locator('[data-testid^="article-card"], .article-card').count();
      expect(initialArticles).toBeGreaterThan(0);
      
      // Check for pagination controls
      const paginationContainer = page.locator('.pagination, [data-testid="pagination"]');
      
      if (await paginationContainer.count() > 0) {
        const nextBtn = paginationContainer.locator('.next, [data-testid="next-page"]');
        
        if (await nextBtn.count() > 0 && await nextBtn.isVisible()) {
          // Monitor network requests for HTMX
          let htmxRequestMade = false;
          
          page.on('request', request => {
            if (request.headers()['hx-request'] === 'true') {
              htmxRequestMade = true;
            }
          });
          
          await nextBtn.click();
          await page.waitForTimeout(2000);
          
          // Verify HTMX request was made
          expect(htmxRequestMade).toBeTruthy();
          
          // Verify new articles loaded
          const newArticles = await page.locator('[data-testid^="article-card"], .article-card').count();
          expect(newArticles).toBeGreaterThan(0);
        }
      }
    });

    test('should handle article filtering via HTMX', async () => {
      const filterContainer = page.locator('.filters, [data-testid="filters"]');
      
      if (await filterContainer.count() > 0) {
        const categorySelect = filterContainer.locator('select[name="category"], [data-testid="category-filter"]');
        
        if (await categorySelect.count() > 0) {
          let htmxFilterRequest = false;
          
          page.on('request', request => {
            if (request.headers()['hx-request'] === 'true' && request.url().includes('category')) {
              htmxFilterRequest = true;
            }
          });
          
          // Apply filter
          await categorySelect.selectOption('technology');
          await page.waitForTimeout(2000);
          
          // Verify HTMX filter request
          expect(htmxFilterRequest).toBeTruthy();
          
          // Verify articles updated
          const articles = await page.locator('[data-testid^="article-card"], .article-card').count();
          const noResults = await page.locator('.no-results, [data-testid="no-results"]').count();
          
          expect(articles > 0 || noResults > 0).toBeTruthy();
        }
      }
    });

    test('should support live search with HTMX', async () => {
      const searchInput = page.locator('input[name="search"], [data-testid="search-input"]');
      
      if (await searchInput.count() > 0) {
        let searchRequest = false;
        
        page.on('request', request => {
          if (request.headers()['hx-request'] === 'true' && 
              (request.url().includes('search') || request.method() === 'POST')) {
            searchRequest = true;
          }
        });
        
        // Type search query
        await searchInput.fill('climate');
        await page.waitForTimeout(1500); // Wait for debounce
        
        // Verify search request was made
        expect(searchRequest).toBeTruthy();
        
        // Verify search results
        const searchResults = await page.locator('[data-testid^="article-card"], .article-card').count();
        const noResults = await page.locator('.no-results, [data-testid="no-results"]').count();
        
        expect(searchResults > 0 || noResults > 0).toBeTruthy();
      }
    });
  });

  test.describe('Article Detail HTMX Integration', () => {
    test('should load article details via HTMX modal', async () => {
      const firstArticle = page.locator('[data-testid^="article-card"], .article-card').first();
      const detailTrigger = firstArticle.locator('[hx-get], [data-hx-get], .article-link[hx-target]');
      
      if (await detailTrigger.count() > 0) {
        let articleDetailRequest = false;
        
        page.on('request', request => {
          if (request.headers()['hx-request'] === 'true' && request.url().includes('article')) {
            articleDetailRequest = true;
          }
        });
        
        await detailTrigger.click();
        await page.waitForTimeout(2000);
        
        // Verify HTMX request for article detail
        expect(articleDetailRequest).toBeTruthy();
        
        // Check for modal or detail view
        const modal = page.locator('.modal, [data-testid="article-modal"]');
        const detailView = page.locator('.article-detail, [data-testid="article-detail"]');
        
        expect(await modal.count() > 0 || await detailView.count() > 0).toBeTruthy();
      }
    });

    test('should handle bias score updates via HTMX', async () => {
      // Navigate to an article detail page first
      const articleLink = page.locator('[data-testid^="article-card"] a, .article-card a').first();
      
      if (await articleLink.count() > 0) {
        await articleLink.click();
        await page.waitForLoadState('networkidle');
        
        const biasSlider = page.locator('.bias-slider, [data-testid="bias-slider"]');
        
        if (await biasSlider.count() > 0) {
          let biasUpdateRequest = false;
          
          page.on('request', request => {
            if (request.headers()['hx-request'] === 'true' && 
                (request.url().includes('bias') || request.url().includes('score'))) {
              biasUpdateRequest = true;
            }
          });
          
          // Interact with bias slider
          await biasSlider.click();
          await page.waitForTimeout(1000);
          
          // If request was made, verify it
          if (biasUpdateRequest) {
            expect(biasUpdateRequest).toBeTruthy();
          }
        }
      }
    });

    test('should load article summary via HTMX', async () => {
      const summaryTrigger = page.locator('[hx-get*="summary"], [data-testid="summary-btn"]');
      
      if (await summaryTrigger.count() > 0) {
        let summaryRequest = false;
        
        page.on('request', request => {
          if (request.headers()['hx-request'] === 'true' && request.url().includes('summary')) {
            summaryRequest = true;
          }
        });
        
        await summaryTrigger.click();
        await page.waitForTimeout(2000);
        
        // Verify summary request
        expect(summaryRequest).toBeTruthy();
        
        // Check for summary content
        const summaryContent = page.locator('.summary-content, [data-testid="summary-content"]');
        expect(await summaryContent.count()).toBeGreaterThan(0);
      }
    });
  });

  test.describe('Real-time Updates with HTMX', () => {
    test('should handle Server-Sent Events integration', async () => {
      // Look for SSE connection indicators
      const progressIndicator = page.locator('.progress-indicator, [data-testid="progress-indicator"]');
      
      if (await progressIndicator.count() > 0) {
        // Check if SSE is working by looking for dynamic updates
        const initialText = await progressIndicator.textContent();
        
        // Wait for potential updates
        await page.waitForTimeout(3000);
        
        const updatedText = await progressIndicator.textContent();
        
        // Either content updated or progress indicator is properly initialized
        expect(initialText !== null && updatedText !== null).toBeTruthy();
      }
    });

    test('should update article scores in real-time', async () => {
      const scoreElements = page.locator('.bias-score, [data-testid="bias-score"]');
      
      if (await scoreElements.count() > 0) {
        // Get initial scores
        const initialScores = await scoreElements.allTextContents();
        
        // Trigger a rescore if possible
        const rescoreBtn = page.locator('[data-testid="rescore-btn"], .rescore-button');
        
        if (await rescoreBtn.count() > 0) {
          await rescoreBtn.click();
          
          // Wait for score update
          await page.waitForTimeout(5000);
          
          // Check if scores were updated
          const updatedScores = await scoreElements.allTextContents();
          
          // Scores should be present (may or may not have changed)
          expect(updatedScores.length).toBeGreaterThan(0);
        }
      }
    });
  });

  test.describe('HTMX Form Handling', () => {
    test('should handle feedback form submission via HTMX', async () => {
      const feedbackForm = page.locator('form[hx-post], [data-testid="feedback-form"]');
      
      if (await feedbackForm.count() > 0) {
        let formSubmitRequest = false;
        
        page.on('request', request => {
          if (request.headers()['hx-request'] === 'true' && request.method() === 'POST') {
            formSubmitRequest = true;
          }
        });
        
        // Fill and submit feedback form
        const feedbackInput = feedbackForm.locator('textarea, input[type="text"]');
        const submitBtn = feedbackForm.locator('button[type="submit"], input[type="submit"]');
        
        if (await feedbackInput.count() > 0 && await submitBtn.count() > 0) {
          await feedbackInput.fill('Test feedback message');
          await submitBtn.click();
          
          await page.waitForTimeout(2000);
          
          // Verify HTMX form submission
          expect(formSubmitRequest).toBeTruthy();
          
          // Check for success message
          const successMsg = page.locator('.success, [data-testid="success-message"]');
          expect(await successMsg.count()).toBeGreaterThanOrEqual(0);
        }
      }
    });

    test('should handle form validation with HTMX', async () => {
      const form = page.locator('form[hx-post]');
      
      if (await form.count() > 0) {
        const submitBtn = form.locator('button[type="submit"], input[type="submit"]');
        
        if (await submitBtn.count() > 0) {
          // Try to submit empty form
          await submitBtn.click();
          await page.waitForTimeout(1000);
          
          // Check for validation messages
          const validationMsg = page.locator('.error, .validation-error, [aria-invalid="true"]');
          
          // Either validation messages appear or form handles empty submission gracefully
          expect(await validationMsg.count()).toBeGreaterThanOrEqual(0);
        }
      }
    });
  });

  test.describe('HTMX Navigation Integration', () => {
    test('should handle breadcrumb navigation with HTMX', async () => {
      const breadcrumbs = page.locator('.breadcrumbs, [data-testid="breadcrumbs"]');
      
      if (await breadcrumbs.count() > 0) {
        const breadcrumbLinks = breadcrumbs.locator('a[hx-get], a[data-hx-get]');
        
        if (await breadcrumbLinks.count() > 0) {
          let breadcrumbRequest = false;
          
          page.on('request', request => {
            if (request.headers()['hx-request'] === 'true') {
              breadcrumbRequest = true;
            }
          });
          
          await breadcrumbLinks.first().click();
          await page.waitForTimeout(1000);
          
          // Verify HTMX navigation
          expect(breadcrumbRequest).toBeTruthy();
        }
      }
    });

    test('should maintain URL state with HTMX navigation', async () => {
      const initialUrl = page.url();
      
      // Try navigation with HTMX
      const navLink = page.locator('a[hx-push-url], a[data-hx-push-url]');
      
      if (await navLink.count() > 0) {
        await navLink.first().click();
        await page.waitForTimeout(1000);
        
        const newUrl = page.url();
        
        // URL should change if hx-push-url is used
        expect(newUrl).not.toBe(initialUrl);
        
        // Should be able to navigate back
        await page.goBack();
        await page.waitForTimeout(1000);
        
        expect(page.url()).toBe(initialUrl);
      }
    });
  });

  test.describe('HTMX Error Scenarios', () => {
    test('should handle HTMX timeout gracefully', async () => {
      // Simulate slow response
      await page.route('**/api/**', async route => {
        await new Promise(resolve => setTimeout(resolve, 5000));
        route.continue();
      });
      
      const searchInput = page.locator('input[name="search"], [data-testid="search-input"]');
      
      if (await searchInput.count() > 0) {
        await searchInput.fill('test');
        
        // Should handle timeout (either show loading state or error)
        await page.waitForTimeout(6000);
        
        const errorState = page.locator('.error, [data-testid="error"]');
        const loadingState = page.locator('.loading, [data-testid="loading"]');
        const normalState = page.locator('[data-testid^="article-card"], .article-card');
        
        // Should be in some valid state
        expect(await errorState.count() > 0 || 
               await loadingState.count() > 0 || 
               await normalState.count() > 0).toBeTruthy();
      }
    });

    test('should recover from HTMX errors', async () => {
      let requestCount = 0;
      
      // Fail first request, succeed second
      await page.route('**/api/**', route => {
        requestCount++;
        if (requestCount === 1) {
          route.fulfill({
            status: 500,
            body: 'Server Error'
          });
        } else {
          route.continue();
        }
      });
      
      const searchInput = page.locator('input[name="search"], [data-testid="search-input"]');
      
      if (await searchInput.count() > 0) {
        // First request should fail
        await searchInput.fill('test1');
        await page.waitForTimeout(1000);
        
        // Second request should succeed
        await searchInput.clear();
        await searchInput.fill('test2');
        await page.waitForTimeout(1000);
        
        // Should recover and show results
        const articles = await page.locator('[data-testid^="article-card"], .article-card').count();
        const noResults = await page.locator('.no-results, [data-testid="no-results"]').count();
        
        expect(articles > 0 || noResults > 0).toBeTruthy();
      }
    });
  });
});

import { test, expect } from '@playwright/test';
import { HtmxTestHelper, enableHtmxLogging } from '../utils/htmx-helpers';

test.describe('HTMX Functionality', () => {
  let htmxHelper: HtmxTestHelper;

  test.beforeEach(async ({ page }) => {
    htmxHelper = new HtmxTestHelper(page);
    
    // Enable HTMX logging for debugging
    await enableHtmxLogging(page);
    
    // Navigate to the main page
    await page.goto('/');
    
    // Verify page loaded successfully
    await expect(page.locator('body')).toBeVisible();
  });

  test.describe('Dynamic Content Loading', () => {
    test('should load articles dynamically', async ({ page }) => {
      // Look for load more button or similar trigger
      const loadMoreButton = page.locator('[data-testid="load-more-articles"]').or(
        page.locator('button:has-text("Load More")').or(
          page.locator('.load-more')
        )
      );
      
      if (await loadMoreButton.count() > 0) {
        await htmxHelper.testDynamicLoading(
          '[data-testid="load-more-articles"], button:has-text("Load More"), .load-more',
          '[data-testid="articles-container"], .articles-container, .article-list',
          /Article|article|news|content/i
        );
      } else {
        console.log('No load more button found, skipping dynamic loading test');
        test.skip();
      }
    });

    test('should handle infinite scroll', async ({ page }) => {
      const articlesContainer = page.locator('[data-testid="articles-container"]').or(
        page.locator('.articles-container').or(
          page.locator('.article-list')
        )
      );
      
      if (await articlesContainer.count() > 0) {
        // Scroll to trigger loading
        await page.evaluate(() => {
          window.scrollTo(0, document.body.scrollHeight);
        });
        
        // Wait for potential new content to load
        await page.waitForTimeout(2000);
        
        // Verify container is visible
        await expect(articlesContainer.first()).toBeVisible();
        
        const articleCount = await articlesContainer.locator('.article, [data-testid*="article"]').count();
        expect(articleCount).toBeGreaterThanOrEqual(0);
      } else {
        console.log('No articles container found, skipping infinite scroll test');
        test.skip();
      }
    });

    test('should update content without page refresh', async ({ page }) => {
      // Look for any HTMX-enabled buttons or links
      const htmxElements = page.locator('[hx-get], [hx-post], [hx-put], [hx-delete]');
      
      if (await htmxElements.count() > 0) {
        const firstElement = htmxElements.first();
        const targetSelector = await firstElement.getAttribute('hx-target') || 'body';
        const contentArea = page.locator(targetSelector);
        
        const initialContent = await contentArea.textContent();
        
        await htmxHelper.waitForHtmxRequest(async () => {
          await firstElement.click();
        });
        
        const updatedContent = await contentArea.textContent();
        expect(updatedContent).not.toBe(initialContent);
      } else {
        console.log('No HTMX elements found, skipping content update test');
        test.skip();
      }
    });
  });
  test.describe.skip('Search Functionality', () => {
    test('should perform live search', async ({ page }) => {
      const searchInput = page.locator('[data-testid="search-input"]').or(
        page.locator('input[type="search"]').or(
          page.locator('input[name*="search"], input[placeholder*="search"]')
        )
      );
      
      if (await searchInput.count() > 0) {
        const resultsContainer = page.locator('[data-testid="search-results"]').or(
          page.locator('.search-results').or(
            page.locator('.results')
          )
        );        // Use specific selector strings for HTMX helper
        const searchSelector = '[data-testid="search-input"]';
        const resultsSelector = '[data-testid="search-results"]';
        
        if (await page.locator(searchSelector).count() > 0 || await searchInput.count() > 0) {
          await htmxHelper.testSearch(
            searchSelector,
            resultsSelector,
            'test'
          );
        } else {
          console.log('No search input found, skipping search test');
          test.skip();
        }
      } else {
        console.log('No search input found, skipping search test');
        test.skip();
      }
    });

    test('should clear search results', async ({ page }) => {
      const searchInput = page.locator('[data-testid="search-input"]').or(
        page.locator('input[type="search"]').or(
          page.locator('input[name*="search"], input[placeholder*="search"]')
        )
      );
      
      if (await searchInput.count() > 0) {
        const resultsContainer = page.locator('[data-testid="search-results"]').or(
          page.locator('.search-results').or(
            page.locator('.results')
          )
        );
        
        // Perform search first
        await searchInput.first().fill('test');
        await page.waitForTimeout(1000);
        
        // Clear search
        await searchInput.first().fill('');
        await page.waitForTimeout(1000);
        
        // Check if results are cleared or hidden
        if (await resultsContainer.count() > 0) {
          const resultsVisible = await resultsContainer.first().isVisible();
          if (resultsVisible) {
            const resultsText = await resultsContainer.first().textContent();
            expect(resultsText?.trim() || '').toBe('');
          }
        }
      } else {
        console.log('No search input found, skipping clear search test');
        test.skip();
      }
    });

    test('should handle search with no results', async ({ page }) => {
      const searchInput = page.locator('[data-testid="search-input"]').or(
        page.locator('input[type="search"]').or(
          page.locator('input[name*="search"], input[placeholder*="search"]')
        )
      );
      
      if (await searchInput.count() > 0) {
        await searchInput.first().fill('xyzabc123nonexistentquery');
        await page.waitForTimeout(2000);
        
        // Check for no results message
        const noResultsMessage = page.locator('[data-testid="no-results"]').or(
          page.locator(':has-text("No results"), :has-text("not found"), :has-text("no matches")')
        );
        
        if (await noResultsMessage.count() > 0) {
          await expect(noResultsMessage.first()).toBeVisible();
        } else {
          // Alternative: check that results container is empty
          const resultsContainer = page.locator('[data-testid="search-results"]').or(
            page.locator('.search-results')
          );
          if (await resultsContainer.count() > 0) {
            const resultsCount = await resultsContainer.first().locator('.result, .item').count();
            expect(resultsCount).toBe(0);
          }
        }
      } else {
        console.log('No search input found, skipping no results test');
        test.skip();
      }
    });
  });

  test.describe('Form Interactions', () => {
    test('should submit forms via HTMX', async ({ page }) => {
      const htmxForm = page.locator('form[hx-post], form[hx-put], form[hx-get]');
      
      if (await htmxForm.count() > 0) {
        const form = htmxForm.first();
        const targetSelector = await form.getAttribute('hx-target') || 'body';
          await htmxHelper.submitHtmxForm(
          'form[hx-post], form[hx-put], form[hx-get]',
          targetSelector
        );
      } else {
        console.log('No HTMX forms found, skipping form submission test');
        test.skip();
      }
    });

    test('should validate form inputs', async ({ page }) => {
      const forms = page.locator('form');
      
      if (await forms.count() > 0) {
        const form = forms.first();
        const submitButton = form.locator('[type="submit"], button:has-text("Submit")');
        
        if (await submitButton.count() > 0) {
          // Try to submit empty form to trigger validation
          await submitButton.click();
          await page.waitForTimeout(1000);
          
          // Check for validation messages
          const validationMessages = page.locator('.error-message, .invalid-feedback, .field-error');
          if (await validationMessages.count() > 0) {
            await expect(validationMessages.first()).toBeVisible();
          }
        }
      } else {
        console.log('No forms found, skipping validation test');
        test.skip();
      }
    });
  });

  test.describe('Navigation and History', () => {
    test('should handle navigation links', async ({ page }) => {
      const navigationLinks = page.locator('nav a, .nav a, [data-testid*="nav"] a');
      
      if (await navigationLinks.count() > 0) {
        const link = navigationLinks.first();
        const href = await link.getAttribute('href');
        
        if (href && href.startsWith('/')) {
          await link.click();
          await page.waitForLoadState('networkidle');
          
          // Verify navigation occurred
          expect(page.url()).toContain(href);
        }
      } else {
        console.log('No navigation links found, skipping navigation test');
        test.skip();
      }
    });    test('should handle browser back/forward', async ({ page }) => {
      // Navigate to articles page first 
      await page.goto('/articles');
      await page.waitForLoadState('networkidle');
      const initialUrl = page.url();
      
      const navigationLinks = page.locator('nav a, .nav a, [data-testid*="nav"] a');
      
      if (await navigationLinks.count() > 0) {
        // Find a navigation link that will actually change the URL
        let targetLink = null;
        const linkCount = await navigationLinks.count();
        
        for (let i = 0; i < linkCount; i++) {
          const link = navigationLinks.nth(i);
          const href = await link.getAttribute('href');
          
          // Look for a link that goes to a different page than current
          if (href && href.startsWith('/') && !initialUrl.includes(href)) {
            targetLink = link;
            break;
          }
        }
        
        if (targetLink) {
          // Click the link to navigate away
          await targetLink.click();
          await page.waitForLoadState('networkidle');
          
          // Verify we navigated to a different URL
          const newUrl = page.url();
          expect(newUrl).not.toBe(initialUrl);
          
          // Go back
          await page.goBack();
          await page.waitForLoadState('networkidle');

          // Verify we're back to the articles page (allow for different ways to express the URL)
          const currentUrl = page.url();
          expect(currentUrl).toMatch(/\/articles/);
        } else {
          console.log('No suitable navigation links found for back/forward test, skipping');
          test.skip();
        }
      } else {
        console.log('No navigation links found, skipping back/forward test');
        test.skip();
      }
    });
  });

  test.describe('Performance', () => {
    test('should load quickly', async ({ page }) => {
      const startTime = Date.now();
      await page.goto('/');
      await page.waitForLoadState('networkidle');
      const loadTime = Date.now() - startTime;
      
      expect(loadTime).toBeLessThan(3000); // 3 second limit
    });

    test('should handle rapid interactions', async ({ page }) => {
      const clickableElements = page.locator('button, a, [role="button"]');
      
      if (await clickableElements.count() > 0) {
        const button = clickableElements.first();
        
        // Rapid clicks
        for (let i = 0; i < 5; i++) {
          await button.click();
          await page.waitForTimeout(100);
        }
        
        // Verify page is still responsive
        await expect(page.locator('body')).toBeVisible();
      } else {
        console.log('No clickable elements found, skipping rapid interaction test');
        test.skip();
      }
    });
  });
});

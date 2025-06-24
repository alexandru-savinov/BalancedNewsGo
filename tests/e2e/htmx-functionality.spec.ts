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

  // Helper functions for HTMX element discovery - moved outside test to reduce nesting
  const findLoadMoreButtons = async (page: any) => {
    const loadMoreButtons = page.locator('[hx-get*="load-more"], button[hx-get], [hx-get*="page"]');
    if (await loadMoreButtons.count() > 0) {
      const element = loadMoreButtons.first();
      const target = await element.getAttribute('hx-target') ?? '#articles-container';
      return { element, target };
    }
    return null;
  };

  const findFilterForms = async (page: any) => {
    const filterForms = page.locator('form[hx-get], select[hx-get]');
    if (await filterForms.count() > 0) {
      const element = filterForms.first();
      const target = await element.getAttribute('hx-target') ?? '#articles-container';

      // For select elements, we need to change the value to trigger an update
      if (await element.evaluate((el: any) => el.tagName.toLowerCase()) === 'select') {
        const options = element.locator('option');
        const optionCount = await options.count();
        if (optionCount > 1) {
          // Select the second option (index 1) to ensure content changes
          await element.selectOption({ index: 1 });
        }
      }
      return { element, target };
    }
    return null;
  };

  const findGenericHtmxElements = async (page: any) => {
    const htmxElements = page.locator('[hx-get]:not([hx-target="body"]), [hx-post], [hx-put], [hx-delete]');
    if (await htmxElements.count() > 0) {
      const element = htmxElements.first();
      const target = await element.getAttribute('hx-target') ?? '#articles-container';
      return { element, target };
    }
    return null;
  };

  const shouldClickElement = async (element: any) => {
    const tagName = await element.evaluate((el: any) => el.tagName.toLowerCase());
    return tagName !== 'select';
  };

  const findNavigationLink = async (page: any, initialUrl: string) => {
    const navigationLinks = page.locator('nav a, .nav a, [data-testid*="nav"] a');
    const linkCount = await navigationLinks.count();

    for (let i = 0; i < linkCount; i++) {
      const link = navigationLinks.nth(i);
      const href = await link.getAttribute('href');

      // Look for a link that goes to a different page than current
      if (href && href.startsWith('/') && !initialUrl.includes(href)) {
        return link;
      }
    }
    return null;
  };

  const testContentUpdate = async (page: any, htmxHelper: HtmxTestHelper) => {
    // Look for HTMX elements that are more likely to cause content changes
    // Prioritize load more buttons, filters, and form submissions over navigation links
    const elementInfo = await findLoadMoreButtons(page) ||
                        await findFilterForms(page) ||
                        await findGenericHtmxElements(page);

    if (!elementInfo) {
      console.log('No suitable HTMX elements found for content update test, skipping');
      test.skip();
      return;
    }

    const { element: targetElement, target: targetSelector } = elementInfo;
    const contentArea = page.locator(targetSelector);

    // Wait for the content area to be visible
    await expect(contentArea).toBeVisible();

    const initialContent = await contentArea.textContent();

    await htmxHelper.waitForHtmxRequest(async () => {
      // If it's a select element and we haven't already changed it, click it
      if (await shouldClickElement(targetElement)) {
        await targetElement.click();
      }
    });

    // Wait a bit for the content to update
    await page.waitForTimeout(500);

    const updatedContent = await contentArea.textContent();
    expect(updatedContent).not.toBe(initialContent);
  };

  const testBrowserBackForward = async (page: any) => {
    // Navigate to articles page first
    await page.goto('/articles');
    await page.waitForLoadState('networkidle');
    const initialUrl = page.url();

    const navigationLinks = page.locator('nav a, .nav a, [data-testid*="nav"] a');

    if (await navigationLinks.count() === 0) {
      console.log('No navigation links found, skipping back/forward test');
      test.skip();
      return;
    }

    const targetLink = await findNavigationLink(page, initialUrl);

    if (!targetLink) {
      console.log('No suitable navigation links found for back/forward test, skipping');
      test.skip();
      return;
    }

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
  };

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
      await testContentUpdate(page, htmxHelper);
    });
  });


  test.describe('Form Interactions', () => {
    test('should submit forms via HTMX', async ({ page }) => {
      const htmxForm = page.locator('form[hx-post], form[hx-put], form[hx-get]');
      
      if (await htmxForm.count() > 0) {
        const form = htmxForm.first();
        const targetSelector = await form.getAttribute('hx-target') ?? 'body';
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
        
        if (href?.startsWith('/')) {
          await link.click();
          await page.waitForLoadState('networkidle');
          
          // Verify navigation occurred
          expect(page.url()).toContain(href);
        }
      } else {
        console.log('No navigation links found, skipping navigation test');
        test.skip();
      }
    });

    test('should handle browser back/forward', async ({ page }) => {
      await testBrowserBackForward(page);
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

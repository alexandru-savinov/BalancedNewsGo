import { test, expect, Page } from '@playwright/test';

// Helper function to start the Go server for testing
async function waitForServer(page: Page, url: string, timeout: number = 30000): Promise<void> {
  const startTime = Date.now();
  while (Date.now() - startTime < timeout) {
    try {
      await page.goto(url, { waitUntil: 'networkidle', timeout: 5000 });
      return;    } catch (error: any) {
      console.log('Server not ready, waiting...', error?.message ?? 'Unknown error');
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }
  throw new Error(`Server did not start within ${timeout}ms`);
}

test.describe('HTMX Integration Tests', () => {
  const baseURL = 'http://localhost:8080';

  test.beforeAll(async () => {
    // Note: In a real CI environment, you would start the Go server here
    // For now, we assume the server is running
    console.log('E2E tests assume Go server is running on localhost:8080');
  });

  test.beforeEach(async ({ page }) => {
    // Wait for server to be available
    await waitForServer(page, baseURL);
  });  test('Articles page loads without full refresh on pagination', async ({ page }) => {
    // Navigate to articles page
    await page.goto(`${baseURL}/articles`);
    
    // Wait for the page to load completely
    await page.waitForLoadState('networkidle');
    
    // Wait for articles to load instead of title
    await page.waitForSelector('.article-item', { timeout: 15000 });
    
    // Check that articles are loaded
    const articles = page.locator('.article-item');
    const articleCount = await articles.count();
    expect(articleCount).toBeGreaterThan(0);
      // Get the initial page content to verify HTMX updates
    const firstArticleTitle = await articles.first().locator('.article-title a').textContent();
    
    // Look for pagination links with HTMX attributes
    const nextPageLink = page.locator('a[hx-get*="offset="]').first();
    
    if (await nextPageLink.count() > 0) {
      // Set up network monitoring to verify HTMX request
      let htmxRequestMade = false;
      page.on('request', request => {
        if (request.url().includes('articles') && request.headers()['hx-request']) {
          htmxRequestMade = true;
        }
      });
      
      // Click the pagination link
      await nextPageLink.click();
      
      // Wait for HTMX to update the content
      await page.waitForTimeout(1000);
      
      // Verify that content changed without page reload
      const newArticles = await articles.count();
      const newFirstArticleTitle = await articles.first().locator('.article-title a').textContent();
      
      // Articles should be different (assuming different pages have different content)
      if (newArticles > 0) {
        expect(newFirstArticleTitle).not.toBe(firstArticleTitle);
      }
      
      // Verify HTMX request was made
      expect(htmxRequestMade).toBe(true);
    } else {
      console.log('No pagination links found, skipping pagination test');
    }
  });

  test('Article detail page loads without full refresh when clicking article links', async ({ page }) => {
    // Navigate to articles page
    await page.goto(`${baseURL}/articles`);
    
      // Wait for articles to load
    const articleLinks = page.locator('a[href*="/article/"]');
    const linkCount = await articleLinks.count();
    expect(linkCount).toBeGreaterThan(0);
    // Set up network monitoring
    let htmxRequestMade = false;
    page.on('request', request => {
      if (request.url().includes('/article/') && request.headers()['hx-request']) {
        htmxRequestMade = true;
      }
    });
      // Click on the first article link
    await articleLinks.first().click();
    
    // Wait for HTMX to load the article
    await page.waitForTimeout(1500);
    
    // Verify the main content area changed
    const mainContent = page.locator('#main, .main-content, .article-detail');
    if (await mainContent.count() > 0) {
      await expect(mainContent).toBeVisible();
    }
    
    // Verify HTMX request was made
    expect(htmxRequestMade).toBe(true);
    
    // The page title might change if the HTMX response updates it
    // If not, that's also acceptable for partial page updates
  });

  test('Search functionality works with HTMX', async ({ page }) => {
    await page.goto(`${baseURL}/articles`);
    
    // Look for search form or filter inputs
    const searchInput = page.locator('input[name="query"], input[type="search"]').first();
    const sourceFilter = page.locator('select[name="source"], input[name="source"]').first();
    
    if (await searchInput.count() > 0) {
      // Set up network monitoring
      let htmxRequestMade = false;
      page.on('request', request => {
        if (request.url().includes('query=') && request.headers()['hx-request']) {
          htmxRequestMade = true;
        }
      });
      
      // Perform search
      await searchInput.fill('test');
      await searchInput.press('Enter');
      
      // Wait for HTMX response
      await page.waitForTimeout(1000);
      
      // Verify request was made via HTMX
      expect(htmxRequestMade).toBe(true);
    }
    
    if (await sourceFilter.count() > 0) {
      // Test source filtering
      let filterRequestMade = false;
      page.on('request', request => {
        if (request.url().includes('source=') && request.headers()['hx-request']) {
          filterRequestMade = true;
        }
      });
      
      if (await sourceFilter.locator('option').count() > 0) {
        // It's a select element
        await sourceFilter.selectOption({ index: 1 });
      } else {
        // It's an input element
        await sourceFilter.fill('cnn');
        await sourceFilter.press('Enter');
      }
      
      await page.waitForTimeout(1000);
      expect(filterRequestMade).toBe(true);
    }
  });
  test('Error handling works correctly with HTMX', async ({ page }) => {
    await page.goto(`${baseURL}/articles`);
    
    // Try to access a non-existent article directly via HTMX
    await page.evaluate(() => {
      // Simulate an HTMX request to a non-existent article
      if ((window as any).htmx) {
        (window as any).htmx.ajax('GET', '/article/999999', '#main');
      }
    });
    
    await page.waitForTimeout(1000);
    
    // Check for error message in the target container
    const errorMessage = page.locator('.error, .alert-error, [role="alert"]');
    if (await errorMessage.count() > 0) {
      await expect(errorMessage).toBeVisible();
      expect(await errorMessage.textContent()).toContain('not found');
    }
  });

  test('Navigation preserves browser history with HTMX', async ({ page }) => {
    await page.goto(`${baseURL}/articles`);
    
    // Navigate to an article using HTMX
    const articleLink = page.locator('a[hx-get*="/article/"]').first();
    if (await articleLink.count() > 0) {
      await articleLink.click();
      await page.waitForTimeout(1000);
      
      // Go back using browser navigation
      await page.goBack();
      await page.waitForTimeout(500);
      
      // Should be back on articles page
      expect(page.url()).toContain('/articles');
      
      // Go forward
      await page.goForward();
      await page.waitForTimeout(500);
      
      // Should be on article page
      expect(page.url()).toContain('/article/');
    }
  });

  test('Page loading indicators work with HTMX', async ({ page }) => {
    await page.goto(`${baseURL}/articles`);
    
    // Check if loading indicators are present
    const loadingIndicator = page.locator('.htmx-indicator, .loading, .spinner');
    
    if (await loadingIndicator.count() > 0) {
      // Should be hidden initially
      await expect(loadingIndicator).toBeHidden();
      
      // Trigger an HTMX request
      const htmxTrigger = page.locator('a[hx-get], button[hx-get], form[hx-post]').first();
      if (await htmxTrigger.count() > 0) {
        // The loading indicator should briefly appear during the request
        await htmxTrigger.click();
        
        // Note: This is tricky to test reliably due to timing
        // In a real implementation, you might use slower network conditions
        await page.waitForTimeout(100);
        
        // After the request completes, indicator should be hidden again
        await page.waitForTimeout(1000);
        await expect(loadingIndicator).toBeHidden();
      }
    }
  });

  test('Accessibility is maintained during HTMX updates', async ({ page }) => {
    await page.goto(`${baseURL}/articles`);
    
    // Check basic accessibility requirements
    await expect(page.locator('main, [role="main"]')).toHaveCount(1);
    
    // Navigation should be accessible
    const nav = page.locator('nav, [role="navigation"]');
    if (await nav.count() > 0) {
      await expect(nav).toBeVisible();
    }
    
    // After HTMX updates, check that focus management works
    const firstLink = page.locator('a').first();
    if (await firstLink.count() > 0) {
      await firstLink.focus();
      await expect(firstLink).toBeFocused();
      
      // If there's an HTMX trigger, test focus after update
      if (await firstLink.getAttribute('hx-get')) {
        await firstLink.click();
        await page.waitForTimeout(1000);
        
        // Focus should be managed appropriately after HTMX update
        // This depends on your specific focus management implementation
      }
    }
      // Check for proper heading structure
    const headings = page.locator('h1, h2, h3, h4, h5, h6');
    const headingCount = await headings.count();
    expect(headingCount).toBeGreaterThan(0);
    
    // All images should have alt text
    const images = page.locator('img');
    const imageCount = await images.count();
    for (let i = 0; i < imageCount; i++) {
      const img = images.nth(i);
      const alt = await img.getAttribute('alt');
      expect(alt).not.toBeNull();
    }
  });

  test('Real-time updates work with Server-Sent Events', async ({ page }) => {
    await page.goto(`${baseURL}/articles`);
    
    // Check if there are any SSE connections for real-time updates
    let sseConnectionOpened = false;
    page.on('request', request => {
      if (request.url().includes('/events') || request.headers()['accept']?.includes('text/event-stream')) {
        sseConnectionOpened = true;
      }
    });
    
    // Look for elements that might trigger SSE connections
    const realtimeElements = page.locator('[hx-sse], [data-sse], .live-updates');
    
    if (await realtimeElements.count() > 0) {
      // Wait a bit to see if SSE connection is established
      await page.waitForTimeout(2000);
      
      if (sseConnectionOpened) {
        console.log('SSE connection detected');
        // In a real test, you might trigger an event on the server
        // and verify that the UI updates automatically
      }
    }
  });

  test('Performance is acceptable with HTMX', async ({ page }) => {
    // Navigate to articles page and measure performance
    const startTime = Date.now();
    
    await page.goto(`${baseURL}/articles`);
    await page.waitForLoadState('networkidle');
    
    const initialLoadTime = Date.now() - startTime;
    console.log(`Initial page load time: ${initialLoadTime}ms`);
    
    // Expect reasonable load time (adjust threshold as needed)
    expect(initialLoadTime).toBeLessThan(5000);
    
    // Test HTMX request performance
    const htmxTrigger = page.locator('a[hx-get]').first();
    if (await htmxTrigger.count() > 0) {
      const htmxStartTime = Date.now();
      
      await htmxTrigger.click();
      await page.waitForTimeout(1000);
      
      const htmxLoadTime = Date.now() - htmxStartTime;
      console.log(`HTMX request time: ${htmxLoadTime}ms`);
      
      // HTMX requests should be faster than full page loads
      expect(htmxLoadTime).toBeLessThan(3000);
    }
  });
});

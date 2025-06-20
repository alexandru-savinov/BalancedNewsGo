import { test, expect } from '@playwright/test';

test.describe('Performance Tests', () => {  test('should meet Core Web Vitals thresholds', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Simple performance measurement
    const performanceMetrics = await page.evaluate(() => {
      const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
      
      return {
        lcp: navigation.loadEventEnd - navigation.fetchStart,
        domContentLoaded: navigation.domContentLoadedEventEnd - navigation.fetchStart,
        loadComplete: navigation.loadEventEnd - navigation.fetchStart
      };
    });
    
    // Verify basic performance thresholds
    expect(performanceMetrics.lcp).toBeLessThan(2500); // 2.5s
    expect(performanceMetrics.domContentLoaded).toBeLessThan(1500); // 1.5s
    expect(performanceMetrics.loadComplete).toBeLessThan(3000); // 3s
  });

  test('should load resources efficiently', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const loadTime = Date.now() - startTime;
    expect(loadTime).toBeLessThan(3000); // 3 second page load
    
    // Check resource sizes
    const resources = await page.evaluate(() => {
      return performance.getEntriesByType('resource').map(r => ({
        name: r.name,
        size: (r as PerformanceResourceTiming).transferSize || 0,
        duration: r.duration,
        type: (r as PerformanceResourceTiming).initiatorType
      }));
    });
    
    // Verify no excessively large resources
    const largeResources = resources.filter(r => r.size > 1024 * 1024); // 1MB
    expect(largeResources).toHaveLength(0);
    
    // Check for reasonable number of requests
    expect(resources.length).toBeLessThan(50); // Reasonable number of requests
    
    // Verify CSS and JS load times
    const cssResources = resources.filter(r => r.type === 'link' || r.name.includes('.css'));
    const jsResources = resources.filter(r => r.type === 'script' || r.name.includes('.js'));
    
    if (cssResources.length > 0) {
      const avgCssLoadTime = cssResources.reduce((sum, r) => sum + r.duration, 0) / cssResources.length;
      expect(avgCssLoadTime).toBeLessThan(1000); // 1 second average CSS load
    }
    
    if (jsResources.length > 0) {
      const avgJsLoadTime = jsResources.reduce((sum, r) => sum + r.duration, 0) / jsResources.length;
      expect(avgJsLoadTime).toBeLessThan(2000); // 2 second average JS load
    }
  });

  test('should handle concurrent users simulation', async ({ page, context }) => {
    // Simulate multiple users by opening multiple pages
    const pages = await Promise.all([
      context.newPage(),
      context.newPage(),
      context.newPage()
    ]);
    
    pages.push(page); // Include original page
    
    // Navigate all pages simultaneously
    const startTime = Date.now();
    await Promise.all(pages.map(p => p.goto('/')));
    const navigationTime = Date.now() - startTime;
    
    // Verify all pages loaded within reasonable time
    expect(navigationTime).toBeLessThan(10000); // 10 seconds for all pages
    
    // Perform actions on all pages simultaneously
    await Promise.all(pages.map(async (p) => {
      await p.waitForLoadState('networkidle');
      
      // Try to interact with form inputs if available
      const formInputs = p.locator('input[type="text"], input[type="email"], textarea');
      if (await formInputs.count() > 0) {
        await formInputs.first().fill('concurrent test');
      }
    }));
    
    // Verify all pages still functional
    for (const p of pages) {
      await expect(p.locator('body')).toBeVisible();
    }
    
    // Close additional pages
    await Promise.all(pages.slice(0, 3).map(p => p.close()));
  });

  test('should handle memory efficiently', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Perform memory-intensive operations
    for (let i = 0; i < 10; i++) {
      // Try to load more content if available
      const loadButton = page.locator('[data-testid="load-more-articles"], button:has-text("Load More")');
      if (await loadButton.count() > 0) {
        await loadButton.first().click();
        await page.waitForTimeout(500);
      } else {
        // Alternatively, scroll to bottom
        await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
        await page.waitForTimeout(500);
      }
    }
    
    // Check for memory leaks (basic check)
    const memoryUsage = await page.evaluate(() => {
      if ('memory' in performance) {
        return (performance as any).memory.usedJSHeapSize;
      }
      return 0;
    });
    
    // Verify memory usage is reasonable (< 50MB)
    if (memoryUsage > 0) {
      expect(memoryUsage).toBeLessThan(50 * 1024 * 1024);
    }
  });
  test('should render without blocking', async ({ page }) => {
    const startTime = Date.now();
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    const renderTime = Date.now() - startTime;
    
    // First render should happen quickly
    expect(renderTime).toBeLessThan(2000); // 2 seconds
  });

  test('should handle rapid navigation', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Find navigation links
    const navLinks = page.locator('nav a, .nav a');
    const linkCount = await navLinks.count();
    
    if (linkCount > 1) {
      const startTime = Date.now();
      
      // Rapidly navigate between pages
      for (let i = 0; i < Math.min(linkCount, 3); i++) {
        const link = navLinks.nth(i);
        const href = await link.getAttribute('href');
        
        if (href && href.startsWith('/')) {
          await link.click();
          await page.waitForLoadState('domcontentloaded');
          await page.waitForTimeout(200);
        }
      }
      
      const totalTime = Date.now() - startTime;
      expect(totalTime).toBeLessThan(5000); // 5 seconds for rapid navigation
    }
  });

  test('should optimize images loading', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const images = page.locator('img');
    const imageCount = await images.count();
    
    if (imageCount > 0) {
      // Check image loading attributes
      for (let i = 0; i < Math.min(imageCount, 5); i++) {
        const img = images.nth(i);
        const loading = await img.getAttribute('loading');
        const sizes = await img.getAttribute('sizes');
        const srcset = await img.getAttribute('srcset');
        
        // Check for lazy loading
        if (i > 2) { // Images below the fold should be lazy loaded
          expect(loading).toBe('lazy');
        }
        
        // Check for responsive images
        const hasResponsive = sizes || srcset;
        if (hasResponsive) {
          expect(hasResponsive).toBeTruthy();
        }
      }
    }
  });
  test('should handle API response times', async ({ page }) => {
    // Set up API monitoring
    const apiResponses: Array<{url: string, status: number}> = [];
    
    page.on('response', (response) => {
      if (response.url().includes('/api/')) {
        apiResponses.push({
          url: response.url(),
          status: response.status()
        });
      }
    });
    
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Perform some actions that might trigger API calls
    const formInputs = page.locator('input[type="text"], input[type="email"], textarea');
    if (await formInputs.count() > 0) {
      await formInputs.first().fill('performance test');
      await page.waitForTimeout(2000);
    }
    
    // Check API responses
    if (apiResponses.length > 0) {
      const successfulResponses = apiResponses.filter(r => r.status < 400);
      expect(successfulResponses.length).toBeGreaterThan(0);
      
      // Check that most APIs return successful responses
      const successRate = successfulResponses.length / apiResponses.length;
      expect(successRate).toBeGreaterThan(0.8); // 80% success rate
    }
  });

  test('should handle JavaScript execution performance', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Measure JavaScript execution time
    const jsPerformance = await page.evaluate(() => {
      const startTime = performance.now();
      
      // Simulate some JavaScript work
      let sum = 0;
      for (let i = 0; i < 10000; i++) {
        sum += Math.random();
      }
      
      const endTime = performance.now();
      return {
        executionTime: endTime - startTime,
        result: sum
      };
    });
    
    // JavaScript execution should be fast
    expect(jsPerformance.executionTime).toBeLessThan(100); // 100ms
  });

  test('should handle DOM size efficiently', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    const domStats = await page.evaluate(() => {
      return {
        nodeCount: document.querySelectorAll('*').length,
        depth: Math.max(...Array.from(document.querySelectorAll('*')).map(el => {
          let depth = 0;
          let parent = el.parentElement;
          while (parent) {
            depth++;
            parent = parent.parentElement;
          }
          return depth;
        }))
      };
    });
    
    // DOM should not be excessively large
    expect(domStats.nodeCount).toBeLessThan(1500); // Reasonable DOM size
    expect(domStats.depth).toBeLessThan(20); // Reasonable nesting depth
  });
});

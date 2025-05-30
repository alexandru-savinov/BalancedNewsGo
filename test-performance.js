#!/usr/bin/env node
// filepath: d:\Dev\newbalancer_go\test-performance.js

const puppeteer = require('puppeteer');
const fs = require('fs');

// Performance targets (from FRONTEND_PROPOSAL.md)
const PERFORMANCE_TARGETS = {
  LCP: 2500, // ms
  FCP: 1800, // ms
  CLS: 0.1,
  CRITICAL_BUNDLE_SIZE_KB: 50, // KB for HTML + Critical CSS + Core JS
  // Note: FID and TTI are harder to measure accurately with Puppeteer alone
};

// URLs to test
const PAGES_TO_TEST = [
  { name: 'Articles Page', url: 'http://localhost:8080/articles' },
  { name: 'Article Detail Page', url: 'http://localhost:8080/article/1' }, // Use article ID 1
  { name: 'Admin Page', url: 'http://localhost:8080/admin' },
];

async function measurePerformance(page, url, pageName) {
  console.log(`\n🚀 Measuring performance for: ${pageName} (${url})`);
  let results = {
    lcp: null,
    fcp: null,
    cls: null,
    criticalBundleSizeKB: 0,
    criticalResources: [],
    nonCriticalResources: [],
    inlineCssPresent: false,
    preloadCssPresent: false,
    dynamicImports: { chartJs: 'not loaded initially', domPurify: 'not loaded initially' },
    cacheHeaders: {},
    serviceWorkerActive: false,
    pictureElementsUsed: false,
    lazyLoadingImagesUsed: false,
    resourceHints: { dnsPrefetch: false, preconnect: false, modulePreload: false },
  };

  const resources = {
    html: 0,
    criticalCss: 0, // Inlined or pushed
    coreJs: 0,      // Initial JS
    otherCss: 0,
    otherJs: 0,
    fonts: 0,
    images: 0,
  };
  let initialLoadResources = new Set();

  await page.setRequestInterception(true);
  page.on('request', (request) => {
    // Allow all requests to continue. Modify this if specific request blocking/mocking is needed.
    if (!request.isInterceptResolutionHandled()) {
        request.continue();
    }
  });

  page.on('response', async (response) => {
    const request = response.request();
    const url = response.url();
    const resourceType = request.resourceType();
    const headers = response.headers();
    const contentLength = parseInt(headers['content-length'] || '0', 10);

    if (!url.startsWith('data:')) { // Ignore data URIs for bundle size
        initialLoadResources.add(url);
        if (resourceType === 'document') resources.html += contentLength;
        else if (resourceType === 'stylesheet' && (url.includes('critical') || response.request().isNavigationRequest())) resources.criticalCss += contentLength; // Approximation
        else if (resourceType === 'script' && (url.includes('main.js') || url.includes('core'))) resources.coreJs += contentLength; // Approximation
        else if (resourceType === 'stylesheet') resources.otherCss += contentLength;
        else if (resourceType === 'script') resources.otherJs += contentLength;
        else if (resourceType === 'font') resources.fonts += contentLength;
        else if (resourceType === 'image') resources.images += contentLength;

        if (headers['cache-control']) {
            results.cacheHeaders[url] = headers['cache-control'];
        }
    }
  });

  await page.goto(url, { waitUntil: 'networkidle0' });

  // Core Web Vitals
  results.fcp = await page.evaluate(() => {
    const entry = performance.getEntriesByType('paint').find(e => e.name === 'first-contentful-paint');
    return entry ? entry.startTime : null;
  });

  results.lcp = await page.evaluate(() => {
    const entries = performance.getEntriesByType('largest-contentful-paint');
    return entries.length > 0 ? entries[entries.length - 1].startTime : null;
  });

  results.cls = await page.evaluate(() => {
    return new Promise((resolve) => {
      let cls = 0;
      const observer = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (!entry.hadRecentInput) {
            cls += entry.value;
          }
        }
      });
      observer.observe({ type: 'layout-shift', buffered: true });
      // Give it a moment to collect initial shifts
      setTimeout(() => {
        observer.disconnect();
        resolve(cls);
      }, 1000);
    });
  });

  results.criticalBundleSizeKB = (resources.html + resources.criticalCss + resources.coreJs) / 1024;

  // Resource Loading Strategy
  const pageContent = await page.content();
  console.log(`DEBUG: Page content for ${pageName}:\n`, pageContent.substring(0, 500)); // Log first 500 chars
  results.inlineCssPresent = /<style>[\s\S]*?<\/style>/i.test(pageContent); // More generic regex
  results.preloadCssPresent = /<link[^>]+rel=[\"']preload[\"'][^>]+as=[\"']style[\"']/i.test(pageContent); // More generic regex

  // Check if Chart.js or DOMPurify were loaded initially
  if ([...initialLoadResources].some(r => r.includes('chart.min.js') || r.includes('Chart.js'))) {
    results.dynamicImports.chartJs = 'loaded initially';
  }
  if ([...initialLoadResources].some(r => r.includes('purify.min.js') || r.includes('dompurify'))) {
    results.dynamicImports.domPurify = 'loaded initially';
  }

  // Service Worker
  results.serviceWorkerActive = await page.evaluate(() => navigator.serviceWorker.controller !== null);

  // Image Optimization
  results.pictureElementsUsed = (await page.$$('picture')).length > 0;
  results.lazyLoadingImagesUsed = (await page.$$('img[loading="lazy"]')).length > 0 || (await page.$$('img[data-src]')).length > 0;

  // Resource Hints
  results.resourceHints.dnsPrefetch = (await page.$$('link[rel="dns-prefetch"]')).length > 0;
  results.resourceHints.preconnect = (await page.$$('link[rel="preconnect"]')).length > 0;
  results.resourceHints.modulePreload = (await page.$$('link[rel="modulepreload"]')).length > 0;


  // Output results
  console.log(`  LCP: ${results.lcp ? results.lcp.toFixed(2) + 'ms' : 'N/A'} (${results.lcp <= PERFORMANCE_TARGETS.LCP ? '✅' : '❌ Target: <' + PERFORMANCE_TARGETS.LCP})`);
  console.log(`  FCP: ${results.fcp ? results.fcp.toFixed(2) + 'ms' : 'N/A'} (${results.fcp <= PERFORMANCE_TARGETS.FCP ? '✅' : '❌ Target: <' + PERFORMANCE_TARGETS.FCP})`);
  console.log(`  CLS: ${results.cls ? results.cls.toFixed(4) : 'N/A'} (${results.cls <= PERFORMANCE_TARGETS.CLS ? '✅' : '❌ Target: <' + PERFORMANCE_TARGETS.CLS})`);
  console.log(`  Critical Bundle Size: ${results.criticalBundleSizeKB.toFixed(2)} KB (${results.criticalBundleSizeKB <= PERFORMANCE_TARGETS.CRITICAL_BUNDLE_SIZE_KB ? '✅' : '❌ Target: <' + PERFORMANCE_TARGETS.CRITICAL_BUNDLE_SIZE_KB})`);
  console.log(`    HTML: ${(resources.html / 1024).toFixed(2)} KB`);
  console.log(`    Critical CSS: ${(resources.criticalCss / 1024).toFixed(2)} KB`);
  console.log(`    Core JS: ${(resources.coreJs / 1024).toFixed(2)} KB`);
  console.log(`  Inlined Critical CSS: ${results.inlineCssPresent ? '✅ Present' : '❌ Not found'}`);
  console.log(`  Preloaded Non-Critical CSS: ${results.preloadCssPresent ? '✅ Present' : '❌ Not found'}`);
  console.log(`  Dynamic Imports: Chart.js - ${results.dynamicImports.chartJs}, DOMPurify - ${results.dynamicImports.domPurify}`);
  console.log(`  Service Worker Active: ${results.serviceWorkerActive ? '✅ Yes' : '❌ No'}`);
  console.log(`  <picture> elements used: ${results.pictureElementsUsed ? '✅ Yes' : '❌ No'}`);
  console.log(`  Lazy loading images used: ${results.lazyLoadingImagesUsed ? '✅ Yes' : '❌ No'}`);
  console.log(`  Resource Hints: DNS Prefetch (${results.resourceHints.dnsPrefetch ? '✅' : '❌'}), Preconnect (${results.resourceHints.preconnect ? '✅' : '❌'}), Module Preload (${results.resourceHints.modulePreload ? '✅' : '❌'})`);
  // Check dynamic loading for Chart.js on admin page specifically
  if (pageName === 'Admin Page' && results.dynamicImports.chartJs === 'not loaded initially') {
    // This assumes Chart.js is loaded when some chart is visible or an action is taken.
    // For a more robust test, you'd trigger the specific action that loads Chart.js.
    // Here, we just check if it's loaded after navigating to admin
    const chartJsLoadedAfterNav = results.criticalResources.some(r => r.includes('chart.min.js') || r.includes('Chart.js'));
    console.log(`  Chart.js loaded on Admin Page: ${chartJsLoadedAfterNav ? '✅ Yes' : '❌ No'}`);
  }


  return results;
}

async function runPerformanceTests() {
  console.log('🧪 Starting Performance Tests...');
  const browser = await puppeteer.launch({
    headless: true, // Set to false to watch tests
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--start-maximized']
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 1920, height: 1080 }); // Desktop for consistent metrics

  const allResults = {};

  for (const pageInfo of PAGES_TO_TEST) {
    // Add a dummy article ID for article detail page if not present
    let urlToTest = pageInfo.url;
    if (pageInfo.name === 'Article Detail Page' && !urlToTest.includes('?id=')) {        // Try to find an article ID from the articles page first
        try {
            await page.goto(PAGES_TO_TEST.find(p=>p.name === 'Articles Page').url, { waitUntil: 'networkidle0' });
            const articleLink = await page.$('a.article-card-link, .article-card a'); // Adjust selector as needed
            if (articleLink) {
                const href = await page.evaluate(el => el.getAttribute('href'), articleLink);
                if (href && href.includes('/article/')) {
                    urlToTest = `http://localhost:8080${href}`;
                    console.log(`    Using article link '${href}' for detail page test.`)
                } else {
                     urlToTest = `http://localhost:8080/article/1`; // Fallback
                     console.log("    Could not find article link, using fallback ID for detail page.")
                }
            } else {
                urlToTest = `http://localhost:8080/article/1`; // Fallback
                console.log("    Could not find article link, using fallback ID for detail page.")
            }
        } catch (e) {
            console.warn("    Error trying to get an article ID, using fallback for detail page:", e.message);
            urlToTest = `http://localhost:8080/article/1`; // Fallback
        }
    }


    allResults[pageInfo.name] = await measurePerformance(page, urlToTest, pageInfo.name);
  }

  console.log('\n📋 Performance Test Summary:');
  let allPassed = true;
  for (const pageName in allResults) {
    const r = allResults[pageName];
    const pagePassed =
      (r.lcp === null || r.lcp <= PERFORMANCE_TARGETS.LCP) &&
      (r.fcp === null || r.fcp <= PERFORMANCE_TARGETS.FCP) &&
      (r.cls === null || r.cls <= PERFORMANCE_TARGETS.CLS) &&
      r.criticalBundleSizeKB <= PERFORMANCE_TARGETS.CRITICAL_BUNDLE_SIZE_KB &&
      r.inlineCssPresent &&
      r.preloadCssPresent &&      (pageName !== 'Admin Page' || (pageName === 'Admin Page' && r.criticalResources && r.criticalResources.some(res => res.includes('chart.min.js') || res.includes('Chart.js')))) && // Chart.js should load on admin
      (r.dynamicImports.domPurify === 'not loaded initially' || pageName === 'Article Detail Page') // DOMPurify might load on article detail
      ;
    console.log(`  ${pageName}: ${pagePassed ? '✅ PASSED' : '❌ FAILED'}`);
    if (!pagePassed) allPassed = false;
  }

  if (allPassed) {
    console.log('\n🎉 All performance checks passed or within acceptable limits!');
  } else {
    console.log('\n⚠️ Some performance checks failed. Review logs above.');
  }

  await browser.close();
  return allResults;
}

runPerformanceTests()
  .then(() => console.log('\nPerformance test script finished.'))
  .catch(error => {
    console.error('\n❌ Error during performance tests:', error);
    process.exit(1);
  });

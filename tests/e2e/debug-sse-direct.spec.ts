import { test, expect } from '@playwright/test';

test.describe('Direct SSE Connection Debug', () => {
  test('should test raw SSE connection behavior', async ({ page }) => {
    console.log('🌐 Navigating to article page');
    await page.goto('/article/5');
    await page.waitForLoadState('load');

    // Inject direct SSE test code
    const sseEvents: string[] = [];
    
    await page.evaluate(() => {
      // Create direct EventSource connection
      const eventSource = new EventSource('/api/llm/score-progress/5');
      
      eventSource.onopen = function(event) {
        console.log('✅ SSE connection opened');
        console.log('ReadyState:', eventSource.readyState);
      };

      eventSource.onmessage = function(event) {
        console.log('📨 Default message:', event.data);
      };

      eventSource.addEventListener('progress', function(event) {
        console.log('📊 Progress event:', event.data);
        try {
          const data = JSON.parse(event.data);
          console.log(`Progress: ${data.percent}% - ${data.message}`);
        } catch (e) {
          console.log('Failed to parse progress data:', e.message);
        }
      });

      eventSource.onerror = function(event) {
        console.log('❌ SSE error occurred');
        console.log('ReadyState:', eventSource.readyState);
        console.log('Event type:', event.type);
        
        if (eventSource.readyState === EventSource.CLOSED) {
          console.log('🔒 SSE connection closed by server');
        } else if (eventSource.readyState === EventSource.CONNECTING) {
          console.log('🔄 SSE attempting to reconnect');
        } else {
          console.log('⚠️ SSE error in open state');
        }
      };

      // Store reference globally for cleanup
      (window as any).testEventSource = eventSource;
    });

    // Monitor console messages
    page.on('console', (msg) => {
      const text = msg.text();
      if (text.includes('SSE') || text.includes('Progress') || text.includes('ReadyState')) {
        sseEvents.push(text);
        console.log(`🔍 Browser: ${text}`);
      }
    });

    // Wait a moment for initial connection
    await page.waitForTimeout(1000);

    // Trigger reanalysis
    console.log('🚀 Triggering reanalysis');
    const response = await page.evaluate(async () => {
      const response = await fetch('/api/llm/reanalyze/5', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      });
      return {
        status: response.status,
        text: await response.text()
      };
    });

    console.log('📡 Reanalysis response:', response);

    // Wait for progress events
    console.log('⏳ Waiting for SSE events...');
    await page.waitForTimeout(5000);

    // Log all collected events
    console.log('📋 All SSE events collected:');
    sseEvents.forEach((event, index) => {
      console.log(`  ${index + 1}. ${event}`);
    });

    // Cleanup
    await page.evaluate(() => {
      if ((window as any).testEventSource) {
        (window as any).testEventSource.close();
      }
    });

    // Verify we got some events
    expect(sseEvents.length).toBeGreaterThan(0);
  });
});

// Debug script to test the actual reanalysis button workflow
// This simulates the real user experience to identify the issue

console.log('🔍 Starting reanalysis button debug...');

const ARTICLE_ID = 587;
const BASE_URL = 'http://localhost:8080';

async function debugReanalysisWorkflow() {
    console.log('📋 Step 1: Testing API endpoints');
    
    // Test 1: Check if article exists
    try {
        const articleResponse = await fetch(`${BASE_URL}/api/articles/${ARTICLE_ID}`);
        const articleData = await articleResponse.json();
        console.log('✅ Article API response:', articleData.success ? 'SUCCESS' : 'FAILED');
        if (articleData.success) {
            console.log('   Article title:', articleData.data.title);
            console.log('   Current score:', articleData.data.composite_score);
        }
    } catch (error) {
        console.error('❌ Article API failed:', error.message);
        return;
    }

    // Test 2: Test SSE endpoint initial connection
    console.log('\n📋 Step 2: Testing SSE endpoint initial state');
    try {
        const sseResponse = await fetch(`${BASE_URL}/api/llm/score-progress/${ARTICLE_ID}`);
        console.log('✅ SSE endpoint accessible:', sseResponse.ok);
        console.log('   Content-Type:', sseResponse.headers.get('content-type'));
        
        // Read first chunk to see initial state
        const reader = sseResponse.body.getReader();
        const decoder = new TextDecoder();
        const { value, done } = await reader.read();
        if (value) {
            const chunk = decoder.decode(value);
            console.log('   Initial SSE data:', chunk);
        }
        reader.releaseLock();
    } catch (error) {
        console.error('❌ SSE endpoint failed:', error.message);
        return;
    }

    // Test 3: Trigger reanalysis
    console.log('\n📋 Step 3: Triggering reanalysis');
    try {
        const reanalysisResponse = await fetch(`${BASE_URL}/api/llm/reanalyze/${ARTICLE_ID}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        const reanalysisData = await reanalysisResponse.json();
        console.log('✅ Reanalysis API response:', reanalysisResponse.ok ? 'SUCCESS' : 'FAILED');
        console.log('   Response data:', reanalysisData);
    } catch (error) {
        console.error('❌ Reanalysis API failed:', error.message);
        return;
    }

    // Test 4: Monitor SSE progress after triggering reanalysis
    console.log('\n📋 Step 4: Monitoring SSE progress updates');
    return new Promise((resolve, reject) => {
        const eventSource = new EventSource(`${BASE_URL}/api/llm/score-progress/${ARTICLE_ID}`);
        const startTime = Date.now();
        let messageCount = 0;
        let lastMessage = null;

        const timeout = setTimeout(() => {
            eventSource.close();
            console.log('\n⏰ Timeout reached (30 seconds)');
            console.log('📊 Summary:');
            console.log(`   Messages received: ${messageCount}`);
            console.log(`   Last message:`, lastMessage);
            resolve();
        }, 30000);

        eventSource.onopen = function(event) {
            console.log('🔗 SSE connection opened');
        };

        eventSource.onmessage = function(event) {
            messageCount++;
            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            console.log(`📨 [${elapsed}s] SSE message ${messageCount}:`, event.data);
            
            try {
                lastMessage = JSON.parse(event.data);
                const status = lastMessage.status || 'unknown';
                const progress = lastMessage.percent || lastMessage.progress || 0;
                const step = lastMessage.step || 'unknown';
                
                console.log(`   Status: ${status}, Progress: ${progress}%, Step: ${step}`);
                
                // Check for completion
                if (status.toLowerCase() === 'complete' || status.toLowerCase() === 'completed' || progress >= 100) {
                    console.log('🎯 Analysis completed!');
                    clearTimeout(timeout);
                    eventSource.close();
                    resolve();
                }
            } catch (parseError) {
                console.log('   Raw message (parse failed):', event.data);
                lastMessage = event.data;
            }
        };

        eventSource.onerror = function(event) {
            console.error('❌ SSE error:', event);
            console.log('   ReadyState:', eventSource.readyState);
            console.log('   EventSource.CONNECTING:', EventSource.CONNECTING);
            console.log('   EventSource.OPEN:', EventSource.OPEN);
            console.log('   EventSource.CLOSED:', EventSource.CLOSED);
            
            if (eventSource.readyState === EventSource.CLOSED) {
                console.log('🔌 SSE connection closed');
                clearTimeout(timeout);
                resolve();
            }
        };
    });
}

// Run the debug workflow
debugReanalysisWorkflow()
    .then(() => {
        console.log('\n🏁 Debug workflow completed');
    })
    .catch(error => {
        console.error('\n💥 Debug workflow failed:', error);
    });

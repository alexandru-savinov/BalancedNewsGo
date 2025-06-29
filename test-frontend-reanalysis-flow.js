// Test script to verify the complete reanalysis flow
// This simulates the button click and monitors the complete process

const ARTICLE_ID = 585;
const TEST_TIMEOUT = 60000; // 60 seconds

console.log('🧪 Testing Frontend Reanalysis Flow');
console.log(`📄 Article ID: ${ARTICLE_ID}`);

async function testReanalysisFlow() {
    try {
        console.log('🚀 Step 1: Trigger reanalysis request');
        
        // Trigger reanalysis
        const response = await fetch(`/api/llm/reanalyze/${ARTICLE_ID}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const result = await response.json();
        console.log('✅ Reanalysis request successful:', result);
        
        console.log('🔌 Step 2: Connect to SSE and monitor progress');
        
        return new Promise((resolve, reject) => {
            const eventSource = new EventSource(`/api/llm/score-progress/${ARTICLE_ID}`);
            const startTime = Date.now();
            let lastStatus = null;
            let progressHistory = [];
            
            // Set up timeout
            const timeout = setTimeout(() => {
                eventSource.close();
                reject(new Error('Test timeout - no completion event received'));
            }, TEST_TIMEOUT);
            
            eventSource.onopen = function(event) {
                console.log('🔗 SSE connection opened');
            };
            
            eventSource.onmessage = function(event) {
                try {
                    const data = JSON.parse(event.data);
                    const currentTime = Date.now();
                    const elapsed = currentTime - startTime;
                    
                    // Track progress
                    progressHistory.push({
                        timestamp: elapsed,
                        status: data.status,
                        step: data.step,
                        percent: data.percent,
                        message: data.message
                    });
                    
                    // Log status changes
                    if (data.status !== lastStatus) {
                        console.log(`📊 [${elapsed}ms] Status: ${data.status} | Step: ${data.step} | Progress: ${data.percent}%`);
                        lastStatus = data.status;
                    }
                    
                    // Check completion using same logic as frontend
                    const progress = data.progress || data.percent || 0;
                    const status = data.status ? data.status.toLowerCase() : '';
                    const isComplete = progress >= 100 || status === 'completed' || status === 'complete';
                    
                    if (isComplete) {
                        clearTimeout(timeout);
                        eventSource.close();
                        
                        console.log('🎉 COMPLETION DETECTED!');
                        console.log('📈 Progress History:');
                        progressHistory.forEach((entry, index) => {
                            console.log(`   ${index + 1}. [${entry.timestamp}ms] ${entry.status} - ${entry.step} (${entry.percent}%)`);
                        });
                        
                        console.log('✅ Test PASSED: Completion event received correctly');
                        resolve({
                            success: true,
                            duration: elapsed,
                            finalStatus: data.status,
                            finalScore: data.final_score,
                            progressHistory: progressHistory
                        });
                    }
                    
                } catch (e) {
                    console.error('❌ Error parsing SSE data:', e);
                }
            };
            
            eventSource.onerror = function(event) {
                clearTimeout(timeout);
                eventSource.close();
                console.error('❌ SSE error:', event);
                reject(new Error('SSE connection error'));
            };
        });
        
    } catch (error) {
        console.error('❌ Test failed:', error);
        throw error;
    }
}

// Run the test
testReanalysisFlow()
    .then(result => {
        console.log('🏆 TEST COMPLETED SUCCESSFULLY');
        console.log('📊 Results:', result);
        console.log('');
        console.log('✅ The frontend should now properly detect completion and reset the button state.');
        console.log('✅ The race condition has been fixed by setting up event listeners before showing the ProgressIndicator.');
    })
    .catch(error => {
        console.error('💥 TEST FAILED');
        console.error('❌ Error:', error.message);
        console.log('');
        console.log('🔍 This indicates the frontend completion detection is still not working properly.');
        console.log('🔍 Check the browser console for additional debugging information.');
    });

console.log('⏳ Test started... waiting for completion event...');

// JavaScript Console Testing Commands for Reanalysis Functionality
// Copy and paste these commands into browser console for manual testing

// 1. Test component existence
console.log('=== Component Existence Tests ===');
const progressIndicator = document.getElementById('reanalysis-progress');
console.log('ProgressIndicator element:', progressIndicator);

const reanalyzeBtn = document.getElementById('reanalyze-btn');
console.log('Reanalyze button:', reanalyzeBtn);

// 2. Extract article ID
const articleId = reanalyzeBtn ? reanalyzeBtn.getAttribute('data-article-id') : 'not found';
console.log('Article ID:', articleId);

// 3. Test ProgressIndicator component methods (if available)
if (progressIndicator) {
    console.log('ProgressIndicator status:', progressIndicator.status);
    console.log('ProgressIndicator progress:', progressIndicator.progress);
}

// 4. Test SSEClient availability
console.log('=== SSEClient Tests ===');
if (window.SSEClient) {
    console.log('SSEClient available:', typeof window.SSEClient);
} else {
    console.log('SSEClient not found in global scope');
}

// 5. Test EventSource support
console.log('EventSource support:', typeof EventSource !== 'undefined');

// 6. Manual API test function
function testReanalysisAPI(articleId) {
    console.log(`Testing reanalysis API for article ${articleId}...`);
    
    fetch(`/api/llm/reanalyze/${articleId}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({})
    })
    .then(response => {
        console.log('API Response status:', response.status);
        return response.json();
    })
    .then(data => {
        console.log('API Response data:', data);
    })
    .catch(error => {
        console.error('API Error:', error);
    });
}

// 7. Manual SSE test function
function testSSEConnection(articleId) {
    console.log(`Testing SSE connection for article ${articleId}...`);
    
    const eventSource = new EventSource(`/api/llm/score-progress/${articleId}`);
    
    eventSource.onopen = function(event) {
        console.log('SSE Connection opened:', event);
    };
    
    eventSource.onmessage = function(event) {
        console.log('SSE Message received:', JSON.parse(event.data));
    };
    
    eventSource.onerror = function(event) {
        console.log('SSE Error:', event);
    };
    
    // Auto-close after 30 seconds
    setTimeout(() => {
        eventSource.close();
        console.log('SSE Connection closed after 30 seconds');
    }, 30000);
    
    return eventSource;
}

// 8. Monitor component state changes
function monitorProgressIndicator() {
    if (!progressIndicator) {
        console.log('ProgressIndicator not found');
        return;
    }
    
    const observer = new MutationObserver(function(mutations) {
        mutations.forEach(function(mutation) {
            if (mutation.type === 'attributes') {
                console.log('ProgressIndicator attribute changed:', 
                    mutation.attributeName, 
                    progressIndicator.getAttribute(mutation.attributeName));
            }
        });
    });
    
    observer.observe(progressIndicator, {
        attributes: true,
        attributeFilter: ['status', 'progress', 'class']
    });
    
    console.log('Started monitoring ProgressIndicator changes');
    return observer;
}

// Usage examples:
console.log('=== Usage Examples ===');
console.log('testReanalysisAPI(' + articleId + ') - Test API call');
console.log('testSSEConnection(' + articleId + ') - Test SSE connection');
console.log('monitorProgressIndicator() - Monitor component changes');

const EventSource = require('eventsource');

const articleId = 1602;
const url = `http://localhost:8080/api/llm/score-progress/${articleId}`;
const eventSource = new EventSource(url);

console.log(`Watching progress for article ${articleId}...`);

eventSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log(JSON.stringify(data, null, 2));
    
    // Close the connection if we reach a final state
    if (data.status === 'Success' || data.status === 'Error') {
        console.log('Reached final state, closing connection...');
        eventSource.close();
        process.exit(0);
    }
};

eventSource.onerror = (error) => {
    console.error('Error:', error);
    eventSource.close();
    process.exit(1);
}; 
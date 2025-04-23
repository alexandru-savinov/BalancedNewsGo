// This is a simple script to debug the API response
const http = require('http');
const fs = require('fs');

// Create a test article
const articleData = JSON.stringify({
  title: "Test Article Debug",
  content: "This is a test article for debugging.",
  url: "https://example.com/debug-" + Date.now(),
  pub_date: new Date().toISOString()
});

const options = {
  hostname: 'localhost',
  port: 8080,
  path: '/api/articles',
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Content-Length': articleData.length
  }
};

const req = http.request(options, (res) => {
  console.log(`STATUS: ${res.statusCode}`);
  console.log(`HEADERS: ${JSON.stringify(res.headers)}`);
  
  let data = '';
  
  res.on('data', (chunk) => {
    data += chunk;
  });
  
  res.on('end', () => {
    console.log('Response data:');
    console.log(data);
    
    try {
      const jsonData = JSON.parse(data);
      console.log('Parsed JSON:');
      console.log(JSON.stringify(jsonData, null, 2));
      
      console.log('Article ID:', jsonData.article_id);
      console.log('ID:', jsonData.id);
      
      // Save the response to a file for inspection
      fs.writeFileSync('api_response.json', JSON.stringify(jsonData, null, 2));
      console.log('Response saved to api_response.json');
    } catch (e) {
      console.error('Error parsing JSON:', e);
    }
  });
});

req.on('error', (e) => {
  console.error(`Problem with request: ${e.message}`);
});

req.write(articleData);
req.end();
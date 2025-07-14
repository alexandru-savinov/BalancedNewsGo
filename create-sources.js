// Create baseline sources for E2E tests
const fetch = require('node-fetch');

async function createSources() {
    console.log('Creating baseline sources...');
    
    try {
        // Source 1: HuffPost
        const source1 = {
            name: "HuffPost",
            channel_type: "rss",
            feed_url: "https://www.huffpost.com/section/front-page/feed",
            category: "left",
            default_weight: 1.0
        };
        
        const response1 = await fetch('http://localhost:8080/api/sources', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(source1)
        });
        const result1 = await response1.json();
        console.log(`✓ Created HuffPost (ID: ${result1.data.id})`);
        
        // Source 2: BBC News (THIS IS WHAT TESTS EXPECT)
        const source2 = {
            name: "BBC News",
            channel_type: "rss", 
            feed_url: "https://feeds.bbci.co.uk/news/rss.xml",
            category: "center",
            default_weight: 1.0
        };
        
        const response2 = await fetch('http://localhost:8080/api/sources', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(source2)
        });
        const result2 = await response2.json();
        console.log(`✓ Created BBC News (ID: ${result2.data.id})`);
        
        // Source 3: MSNBC
        const source3 = {
            name: "MSNBC",
            channel_type: "rss",
            feed_url: "http://www.msnbc.com/feeds/latest", 
            category: "right",
            default_weight: 1.0
        };
        
        const response3 = await fetch('http://localhost:8080/api/sources', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(source3)
        });
        const result3 = await response3.json();
        console.log(`✓ Created MSNBC (ID: ${result3.data.id})`);
        
        // Verify all sources
        const allResponse = await fetch('http://localhost:8080/api/sources');
        const allSources = await allResponse.json();
        console.log(`\nTotal sources: ${allSources.data.total}`);
        
        allSources.data.sources.forEach(source => {
            console.log(`- ID ${source.id}: ${source.name} (${source.category})`);
        });
        
        // Check if BBC News is ID 2
        const bbcNews = allSources.data.sources.find(s => s.id === 2);
        if (bbcNews && bbcNews.name === "BBC News") {
            console.log('\n✓ BBC News correctly has ID 2 - tests should pass!');
        } else {
            console.log('\n✗ BBC News does not have ID 2 - tests will fail!');
        }
        
    } catch (error) {
        console.error('Error creating sources:', error.message);
    }
}

createSources();

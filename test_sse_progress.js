// test_sse_progress.js
// Node.js script to trigger rescoring and wait for SSE completion
const fetch = require('node-fetch');
const EventSource = require('eventsource');
const fs = require('fs');

async function triggerRescore(articleId) {
  const res = await fetch(`http://localhost:8080/api/llm/reanalyze/${articleId}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}'
  });
  if (!res.ok) throw new Error('Failed to trigger rescoring');
}

function waitForSSE(articleId) {
  return new Promise((resolve, reject) => {
    const es = new EventSource(`http://localhost:8080/api/llm/score-progress/${articleId}`);
    es.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.status === 'Success' || data.status === 'Error') {
        es.close();
        resolve(data.status);
      }
    };
    es.onerror = (err) => {
      es.close();
      reject(err);
    };
    setTimeout(() => {
      es.close();
      reject(new Error('Timeout waiting for SSE'));
    }, 40000);
  });
}

function getLatestTestArticleId() {
  // Look for the latest test result file in test-results/
  const dir = __dirname + '/test-results';
  const files = fs.readdirSync(dir).filter(f => f.endsWith('.json'));
  if (!files.length) throw new Error('No test result files found');
  // Sort by modified time descending
  files.sort((a, b) => fs.statSync(dir + '/' + b).mtimeMs - fs.statSync(dir + '/' + a).mtimeMs);
  const latest = dir + '/' + files[0];
  const data = JSON.parse(fs.readFileSync(latest, 'utf8'));
  // Try to find an articleId from the test results (Postman format)
  let articleId = null;
  if (data && data.environment && Array.isArray(data.environment.values)) {
    for (const v of data.environment.values) {
      if (v.key && v.key.match(/^articleId(_TC\d+)?$/) && v.value) {
        articleId = v.value;
        break;
      }
    }
  }
  if (!articleId && data.run && data.run.executions) {
    // Fallback: look for articleId in request URLs
    for (const exec of data.run.executions) {
      const url = exec.request && exec.request.url && exec.request.url.raw;
      if (url) {
        const m = url.match(/articles\/(\d+)/);
        if (m) {
          articleId = m[1];
          break;
        }
      }
    }
  }
  if (!articleId) throw new Error('Could not extract articleId from test results');
  return articleId;
}

async function getArticleScore(articleId) {
  const res = await fetch(`http://localhost:8080/api/articles/${articleId}`);
  if (!res.ok) throw new Error('Failed to fetch article');
  const json = await res.json();
  // Try both .score and .composite_score for compatibility
  return json.data && (json.data.composite_score ?? json.data.score);
}

// Usage: node test_sse_progress.js <articleId>
(async () => {
  let articleId = process.argv[2];
  if (!articleId) {
    try {
      articleId = getLatestTestArticleId();
      console.log('Extracted articleId from test results:', articleId);
    } catch (e) {
      console.error('Usage: node test_sse_progress.js <articleId>');
      console.error('Or ensure a test-results/*.json file with articleId is present.');
      process.exit(1);
    }
  }
  await triggerRescore(articleId);
  let lastStatus = null;
  let finalScore = null;
  await new Promise((resolve, reject) => {
    const es = new EventSource(`http://localhost:8080/api/llm/score-progress/${articleId}`);
    es.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.status && data.status !== lastStatus) {
        lastStatus = data.status;
        console.log('SSE status:', data.status, data.message || '');
      }
      if (data.status === 'Success' || data.status === 'Error') {
        finalScore = data.final_score;
        es.close();
        resolve();
      }
    };
    es.onerror = (err) => {
      es.close();
      reject(err);
    };
    setTimeout(() => {
      es.close();
      reject(new Error('Timeout waiting for SSE'));
    }, 40000);
  });
  if (lastStatus === 'Success') {
    const score = await getArticleScore(articleId);
    console.log('Final SSE status: Success. New article score:', score);
  } else {
    console.log('Final SSE status:', lastStatus);
  }
})();

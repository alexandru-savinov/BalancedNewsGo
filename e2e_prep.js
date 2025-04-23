// e2e_prep.js
// E2E Data Preparation & Management: Feed snapshotting, metadata logging, health monitoring
require('dotenv').config();

const fs = require('fs');
const path = require('path');
const fetch = require('node-fetch').default;
const { v4: uuidv4 } = require('uuid');
const xml2js = require('xml2js');

const FEED_CONFIG = path.join(__dirname, 'configs', 'feed_sources.json');
const LLM_CONFIG = path.join(__dirname, 'configs', 'composite_score_config.json');
const SNAPSHOT_ROOT = path.join(__dirname, 'e2e_snapshots');

function nowIso() {
  return new Date().toISOString();
}

async function fetchFeed(url) {
  // Enforce a hard timeout (10s) even if the server is slow to respond
  const HARD_TIMEOUT_MS = 10000;
  return await Promise.race([
    (async () => {
      try {
        const res = await fetch(url);
        const text = await res.text();
        // Try to parse as XML
        await xml2js.parseStringPromise(text);
        return { status: res.status, ok: res.ok, xml: text, validXml: true };
      } catch (e) {
        return { status: null, ok: false, xml: null, validXml: false, error: e.message };
      }
    })(),
    new Promise(resolve =>
      setTimeout(() => resolve({
        status: null,
        ok: false,
        xml: null,
        validXml: false,
        error: `Timeout after ${HARD_TIMEOUT_MS / 1000}s`
      }), HARD_TIMEOUT_MS)
    )
  ]);
}

async function checkApi() {
  try {
    // Check OpenRouter API health
    const headers = { 'Content-Type': 'application/json' };
    if (process.env.LLM_API_KEY) {
      headers['Authorization'] = `Bearer ${process.env.LLM_API_KEY}`;
    }
    const res = await fetch('https://openrouter.ai/api/v1', {
      method: 'POST',
      headers,
      body: JSON.stringify({ test: true }),
      timeout: 10000
    });
    return { status: res.status, ok: res.ok };
  } catch (e) {
    return { status: null, ok: false, error: e.message };
  }
}

async function main() {
  console.log('=== [E2E PREP] Starting E2E preparation script ===');
  // 1. Load configs
  console.log('[E2E PREP] Loading feed and LLM configs...');
  const feeds = JSON.parse(fs.readFileSync(FEED_CONFIG, 'utf-8')).sources;
  const llmModels = JSON.parse(fs.readFileSync(LLM_CONFIG, 'utf-8')).models;

  // 2. Generate run ID and dirs
  const runId = `${Date.now()}_${uuidv4().slice(0, 8)}`;
  const runDir = path.join(SNAPSHOT_ROOT, runId);
  const feedsDir = path.join(runDir, 'feeds');
  fs.mkdirSync(feedsDir, { recursive: true });
  console.log(`[E2E PREP] Run ID: ${runId}`);
  console.log(`[E2E PREP] Snapshot directory: ${runDir}`);

  // 3. Snapshot feeds
  console.log(`[E2E PREP] Snapshotting ${feeds.length} feeds...`);
  const feedResults = [];
  for (let i = 0; i < feeds.length; ++i) {
    const { category, url } = feeds[i];
    const fname = `${category}_${i}.xml`;
    const outPath = path.join(feedsDir, fname);
    console.log(`[E2E PREP][Feed ${i+1}/${feeds.length}] Fetching [${category}] from ${url} ...`);
    const result = await fetchFeed(url);
    feedResults.push({
      category, url, file: `feeds/${fname}`,
      status: result.status, ok: result.ok, validXml: result.validXml, error: result.error || null
    });
    if (result.xml) {
      fs.writeFileSync(outPath, result.xml, 'utf-8');
      console.log(`[E2E PREP][Feed ${i+1}/${feeds.length}] Saved XML to ${outPath}`);
    } else {
      console.warn(`[E2E PREP][Feed ${i+1}/${feeds.length}] Failed to fetch or parse XML: ${result.error}`);
    }
  }
  console.log('[E2E PREP] Feed snapshotting complete.');

  // 4. Health check LLM APIs
  console.log(`[E2E PREP] Checking health of ${llmModels.length} LLM APIs...`);
  const llmResults = [];
  for (let j = 0; j < llmModels.length; ++j) {
    const model = llmModels[j];
    const { perspective, url } = model;
    console.log(`[E2E PREP][LLM ${j+1}/${llmModels.length}] Checking [${perspective}] at ${url} ...`);
    const result = await checkApi();
    llmResults.push({
      perspective, url, status: result.status, ok: result.ok, error: result.error || null
    });
    if (result.ok) {
      console.log(`[E2E PREP][LLM ${j+1}/${llmModels.length}] OK (status: ${result.status})`);
    } else {
      console.warn(`[E2E PREP][LLM ${j+1}/${llmModels.length}] FAILED: ${result.error || 'Unknown error'}`);
    }
  }
  console.log('[E2E PREP] LLM API health checks complete.');

  // 5. Database health check
  console.log('[E2E PREP] Checking database health...');
  let dbStatus = { path: 'news.db', ok: false, error: null };
  try {
    fs.accessSync(path.join(__dirname, 'news.db'), fs.constants.R_OK);
    dbStatus.ok = true;
    console.log('[E2E PREP] Database is accessible.');
  } catch (e) {
    dbStatus.error = e.message;
    console.warn('[E2E PREP] Database check failed:', e.message);
  }

  // 6. Write metadata
  const metadata = {
    runId,
    timestamp: nowIso(),
    feeds: feeds.map(f => ({ category: f.category, url: f.url })),
    llmApis: llmModels.map(m => ({ perspective: m.perspective, url: m.url }))
  };
  fs.writeFileSync(path.join(runDir, 'metadata.json'), JSON.stringify(metadata, null, 2), 'utf-8');
  console.log('[E2E PREP] Metadata written.');

  // 7. Write health log
  const health = {
    feeds: feedResults,
    llmApis: llmResults,
    database: dbStatus,
    checkedAt: nowIso()
  };
  fs.writeFileSync(path.join(runDir, 'health.json'), JSON.stringify(health, null, 2), 'utf-8');
  console.log('[E2E PREP] Health log written.');

  // 8. Check for any unhealthy services and halt if needed
  const unhealthyFeeds = feedResults.filter(f => !f.ok || !f.validXml);
  const unhealthyLLMs = llmResults.filter(l => !l.ok);
  const dbUnhealthy = !dbStatus.ok;
  if (unhealthyFeeds.length > 0 || unhealthyLLMs.length > 0 || dbUnhealthy) {
    console.error('[E2E PREP] Pre-checks failed:');
    if (unhealthyFeeds.length > 0) {
      console.error('[E2E PREP]   Unhealthy feeds:', unhealthyFeeds.map(f => ({ category: f.category, url: f.url, error: f.error })));
    }
    if (unhealthyLLMs.length > 0) {
      console.error('[E2E PREP]   Unhealthy LLM APIs:', unhealthyLLMs.map(l => ({ perspective: l.perspective, url: l.url, error: l.error })));
    }
    if (dbUnhealthy) {
      console.error('[E2E PREP]   Database unavailable:', dbStatus.error);
    }
    process.exit(2);
  }

  // 8b. Trigger article ingestion job (E2E automation)
  const { spawnSync } = require('child_process');
  console.log('---');
  console.log('[E2E PREP][Ingestion] Starting article ingestion job via Go CLI...');
  // Limit ingestion to 3 articles for faster tests
  const ingestion = spawnSync('go', ['run', 'cmd/fetch_articles/main.go', '--count=3'], { encoding: 'utf-8' });
  if (ingestion.stdout) {
    console.log('[E2E PREP][Ingestion][stdout]:\n' + ingestion.stdout);
  }
  if (ingestion.stderr) {
    console.error('[E2E PREP][Ingestion][stderr]:\n' + ingestion.stderr);
  }
  if (ingestion.error) {
    console.error('[E2E PREP][Ingestion] Failed to start ingestion job:', ingestion.error);
    process.exit(3);
  }
  if (ingestion.status !== 0) {
    console.error(`[E2E PREP][Ingestion] Ingestion job exited with code ${ingestion.status}`);
    process.exit(4);
  }
  console.log('[E2E PREP][Ingestion] Article ingestion job completed successfully.');
  console.log('---');

  // 9. Summary log
  console.log(`=== [E2E PREP] E2E snapshot complete. Run ID: ${runId} ===`);
  console.log(`[E2E PREP] Feeds: ${feedResults.length}, LLM APIs: ${llmResults.length}, DB: ${dbStatus.ok ? 'OK' : 'UNAVAILABLE'}`);
  console.log(`[E2E PREP] Snapshot dir: ${runDir}`);
}

main().catch(e => {
  console.error('E2E prep failed:', e);
  process.exit(1);
});

# **NewsBalancer Go Refactor (Static Templates + Plain-JS Data Fetch)**

---

## 🔍 Executive Snapshot

| Phase | Key Files (Touch) | Outcomes |
|-------|------------------|----------|
| **0 – Safety Net** | `cmd/server/main.go (flag)`, `cmd/server/legacy_handlers.go` | Feature-flag old handlers for instant rollback |
| **1 – Detail View** | `cmd/server/main.go`, `web/article.html`, `web/js/article.js` | `/article/:id` served statically; page hydrates via `/api/articles/:id` |
| **2 – List View** | `cmd/server/main.go`, `web/index.html`, `web/js/list.js` | `/articles` served statically; list hydrates via `/api/articles` |
| **3 – Cleanup** | remove HTMX, prune dead code, tidy imports | Legacy server render code gone |
| **4 – Perf & a11y polish** | headers in `internal/api`, Lighthouse, a11y checks | Caching, bundle trim, screen-reader feedback |
| **5 – Automated Tests** | `tests/api_article_test.go`, `tests/ui_smoke.spec.ts` | Regression suite covers API & UI |

---

## Phase 0 – Rollback Guard

* **Task 0.1 — Add runtime flag**  
  *Add `--legacy-html` CLI flag / `LEGACY_HTML=true` env.*  
  * If flag **on**, register original `articlesHandler` / `articleDetailHandler`.  
  * If flag **off** (default), register new static‐file handlers.

* **Task 0.2 — Segregate handlers**  
  *Move old HTML handlers to `legacy_handlers.go`.*  

* **Test 0**  
  * Start server with flag on → pages render as today.  
  * Start without flag → pages load static HTML.*

---

## Phase 1 – Refactor **Article Detail** (`/article/:id`)

### 1.1 Static File Handler

- **1.1.1** Replace placeholder with:  
  ```go
  router.GET("/article/:id", func(c *gin.Context) {
      c.File("./web/article.html")
  })
  ```

### 1.2 Template Prep (`web/article.html`)

- **1.2.1** Strip `{{ }}` Go tags; insert empty elements:  
  ```html
  <h1 id="article-title"></h1>
  <span id="article-source"></span>
  <span id="article-pubdate"></span>
  <span id="article-fetched"></span>
  <div id="article-content"></div>
  <div class="bias-summary">
     <span id="article-score"></span>
     <span id="article-confidence"></span>
     <div id="bias-slider" class="slider"></div>
  </div>
  ```

### 1.3 Client-Side Script (`web/js/article.js`)

1. **Extract ID**  
   ```js
   const id = location.pathname.split('/').pop();
   ```
2. **Fetch JSON** → `/api/articles/${id}`
3. **Populate DOM** (title, content, meta, bias slider).
4. **Loading / error UI** (`aria-live="polite"`).

### 1.4 Verification Checklist

- Open `/article/1` → sees content.  
- Non-existent `/article/99999` → error message visible, console clean.  
- Network tab shows single GET `/api/articles/1`.  
- Screen reader announces “Loading article…” then updated content.

---

## Phase 2 – Refactor **Articles List** (`/articles`)

### 2.1 Static Handler

```go
router.GET("/articles", func(c *gin.Context) {
    c.File("./web/index.html")
})
```

### 2.2 Template & JS

- **2.2.1** Add list container `<div id="articles"></div>`.
- **2.2.2** Remove HTMX script/link attributes.
- **2.2.3** `web/js/list.js`
  - Build **URL** with `limit`, `offset`, optional `source`, `leaning`.
  - Debounced filter input, pagination buttons.
  - Render each article:  
    ```html
    <h3><a href="/article/${id}">${title}</a></h3>
    <p>${source} • ${date}</p>
    <p>${score} | ${conf}</p>
    ```
  - Lazy‐load ensemble details only when user clicks “Show bias breakdown”.

### 2.3 Verification Checklist

- List populates, filters, paginates without reload.  
- Clicking article navigates & loads detail page.  
- Back button shows list again (reload or bfcache OK).  
- Lighthouse performance ≥ 90, no HTMX requests.

---

## Phase 3 – Cleanup

- **3.1** Delete `articlesHandler`, `articleDetailHandler` code.  
- **3.2** Remove HTMX `<script>` tag & attributes.  
- **3.3** Expose `PubDate`, `CreatedAt` fields in `articleToPostmanSchema`.  
- **3.4** Go vet, `goimports`, `golangci-lint run` → 0 issues.

---

## Phase 4 – Performance & Accessibility

1. **Set caching**  
   - Add `Cache-Control: max-age=60` for `/api/articles` list; `max-age=300` for individual article.  
2. **Bundle trim**  
   - Minify `list.js`, `article.js` (optional Makefile task).  
3. **Accessibility pass**  
   - Use `aria-busy` on containers during fetch.  
   - Ensure contrast ≥ WCAG AA.  
   - Tab order: filters → list → pagination.  
   - VoiceOver/NVDA announce loading & errors (`role="status"`).

---

## Phase 5 – Automated Test Suite

| Layer | New Tests | Tool |
|-------|-----------|------|
| **API** | `TestGetArticles`, `TestGetArticleByID` ensure 200, JSON schema, new `PubDate` field present | Go `testing` + `httptest` |
| **UI smoke** | *Happy path* (list → detail → back), *404 path*, *filter path* | Playwright (typescript) |
| **a11y** | Axe-core audit passes | Playwright plugin |

> **Sample API test skeleton (`tests/api_article_test.go`)**

```go
func TestGetArticleByID(t *testing.T) {
  router := setupRouterWithTestDB()
  w := httptest.NewRecorder()
  req, _ := http.NewRequest("GET", "/api/articles/1", nil)
  router.ServeHTTP(w, req)

  if w.Code != http.StatusOK {
      t.Fatalf("expected 200 got %d", w.Code)
  }
  var resp struct {
      Success bool                   `json:"success"`
      Data    map[string]interface{} `json:"data"`
  }
  if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
      t.Fatal(err)
  }
  if !resp.Success || resp.Data["article_id"].(float64) != 1 {
      t.Fatalf("unexpected response: %+v", resp)
  }
}
```

> **Sample Playwright snippet (`tests/ui_smoke.spec.ts`)**

```ts
test('list → detail flow', async ({ page }) => {
  await page.goto('http://localhost:8080/articles');
  await expect(page.locator('.article-item')).toHaveCount(20);
  await page.locator('.article-item a').first().click();
  await expect(page).toHaveURL(/\/article\/\d+/);
  await expect(page.locator('#article-title')).not.toBeEmpty();
  await page.goBack();
  await expect(page.locator('.article-item')).toHaveCount(20);
});
```

Run UI tests via `npm run test:e2e` in CI.

---

## Phase 6 – Deploy & Monitor

1. **Deploy with `LEGACY_HTML=false` (new paths active).**  
2. **Smoke test in staging** via Playwright.  
3. **Monitor** logs & APM (look for spikes in `/api/articles` latency).  
4. **If issues** → flip `LEGACY_HTML=true` env var and redeploy (instant rollback, no code changes).

---

### ✅ Success Criteria

* End-users see identical article list & detail info.  
* No server-rendered HTML on `/articles` or `/article/:id`.  
* API unit tests & UI smoke suite green in CI.  
* Lighthouse **Performance ≥ 90**, **Accessibility ≥ 90**.  
* Rollback switch verified.

# BalancedNewsGo v1.0 Development Plan

---

## Core Backend Stabilization (Single‑User Mode)
- [ ] **Fix LLM Scoring Logic & Tests** – resolve failing tests in `internal/llm`, adjust composite scoring as needed.
- [ ] **Handle Missing Score Edge‑Case** – return a clear error/null when all perspectives fail.
- [ ] **Single‑User Mode Cleanup** – disable multi‑user code; assume a default global user context.
- [ ] **Feedback Feature (Optional)** – disable or make anonymous if not required for 1.0.
- [ ] **RSS Ingestion Robustness** – handle unreachable feeds, prevent duplicates, keep cron stable.
- [ ] **Progress & SSE Stability** – clear progress map correctly, avoid memory leaks.

## Frontend Redesign & UX Improvements
- [ ] **Revamp Visual Design** – modernize theme, color scheme, typography.
- [ ] **Improve Layout & Responsiveness** – refine templates for desktop & mobile.
- [ ] **Enhance User Interactions** – add loading indicators, dynamic filtering without page reloads.
- [ ] **Navigation & UX Polish** – intuitive nav, active states, accessible elements.
- [ ] **Testing UI Usability** – cross‑browser/device manual tests, iterate on feedback.
- [ ] **Frontend Code Refactor** – clean up JS/CSS, remove unused legacy code.

## Testing & Coverage Improvements
- [ ] **Fix Failing Test Suites** – get all tests green, update/complete collections.
- [ ] **Expand Unit Test Coverage** – target ≥80 % for core packages.
- [ ] **Integration Testing with Mock LLM** – simulate end‑to‑end flow using mock service.
- [ ] **Frontend/Browser Testing** – validate HTML rendering via headless checks.
- [ ] **Continuous Integration Setup** – run tests & coverage in CI on every PR.
- [ ] **Test Coverage Monitoring** – generate & review coverage reports.

## Documentation & Developer Guides
- [ ] **Update README for v1.0** – accurate status, quick‑start, usage.
- [ ] **Ensure Configuration Docs are Current** – sync `.env.example`, remove obsolete vars.
- [ ] **API Documentation (Swagger)** – regenerate & publish latest spec.
- [ ] **Code Comments and Cleanup** – add GoDoc comments, remove outdated TODOs.
- [ ] **User Guide (Optional)** – explain bias scores, site features.
- [ ] **Contribution Guidelines** – add `CONTRIBUTING.md`, coding standards.

## Code Maintenance & Best Practices
- [ ] **Remove Legacy/Dead Code** – delete deprecated modes, scripts.
- [ ] **Refactor for Clarity** – break up large files/functions for maintainability.
- [ ] **Apply Linting & Format Checks** – enforce `golangci‑lint`, `gofmt`.
- [ ] **Ensure Efficient & Safe Concurrency** – run `go test -race`, address issues.
- [ ] **Environment & Config Management** – confirm flexible config file paths/env usage.
- [ ] **Dockerization** – finalise Dockerfile & compose for production.
- [ ] **Logging and Monitoring** – streamline logs, expose metrics endpoint.
- [ ] **Security Audit** – update deps, sanitize inputs, check for secrets in logs.

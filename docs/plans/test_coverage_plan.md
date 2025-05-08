# NewsBalancer Test Coverage Todo List

> **Coverage command run on April 30, 2025:**
> - Database layer: 80.6% (from `go tool cover -func=db_cov.out`)
> - LLM Component: 58.2% (from `go tool cover -func=llm_cov.out`)
> - Score Manager: 56.8% (from `go tool cover -func=score_manager_cov.out`)
> - API Layer: 11.0% (from `go tool cover -func=coverage.out`)
> - API Documentation: 0.0% (expected, from `go tool cover -func=api_coverage.out`)
> - Overall System: 11.0% (statements) (from `go test -coverprofile=coverage.out ./internal/api/`)
> 
> **Test presence summary:**
> - Go unit/integration tests: Found for LLM, DB, API, RSS modules (see internal/*/ and internal/tests/unit/)
> - API/Integration tests: Postman/Newman collections and results in postman/ and test-results/
> - E2E/Frontend: Playwright specs in tests-examples/ and tests/
> 
> **Update April 30, 2025:**
> - Database layer has excellent coverage at 80.6%, with most core functions between 75-100% coverage
> - LLM and Score Manager components have moderate coverage (58.2% and 56.8% respectively)
> - API layer still needs significant improvement at 11.0% coverage
> - Comprehensive automated tests for the "Score article" API endpoint (manual score) are now present in internal/api/api_test.go and all scenarios pass
> - The file `internal/api/api_handler_legacy_test.go` contains only handler-level and progress/SSE tests. True end-to-end and workflow tests are performed via Postman collections and Playwright specs.

# Detailed Test Implementation Plan

## 1. Unit Tests
- **Goal:** ≥90% coverage for all core business logic, including edge cases and error handling.
- **Files:** internal/llm/score_calculator.go, internal/llm/composite_score_utils.go, internal/llm/ensemble.go, internal/db/db.go, internal/api/errors.go, internal/api/handlers.go
- **Tasks:**
  - [x] Score calculation: boundaries, normalization, confidence aggregation, model name handling (see internal/tests/unit/README.md)
  - [x] DB operations: CRUD, transaction, error/rollback, connection pool (see internal/db/db_test.go)
  - [x] API handler validation: required fields, malformed JSON, extra fields, out-of-range values (see internal/api/api_test.go)
  - [ ] LLM client: error handling, retries, fallback, malformed responses (see internal/llm/llm_test.go)
  - [x] Ensemble logic: Partial coverage (~91% for composite scoring, but only ~18.6% for ensemble.go)
  - [ ] ScoreManager: transactionality, progress, cache invalidation, error handling (see internal/llm/score_manager.go)

## 2. Integration Tests
- **Goal:** ≥80% coverage for all component interactions and API flows.
- **Files:** internal/api/api.go, internal/llm/score_manager.go, internal/db/db.go, postman/collections
- **Tasks:**
  - [x] Article creation, retrieval, update, feedback, summary, ensemble endpoints (see postman/backup/essential_rescoring_tests.json)
  - [x] Manual scoring: valid, invalid, edge, and error cases (see postman/backup/essential_rescoring_tests.json)
  - [ ] Rescoring progress (SSE): progress tracking is 86.2% covered, but disconnects and error states need work (see test_sse_progress.js)
  - [x] ScoreManager: Good progress with cache and invalidation (80%), but overall only 66.8% covered
  - [x] DB state transitions: Database layer has excellent 80.6% coverage

> **Note:** The file `internal/api/api_handler_legacy_test.go` contains only handler-level and progress/SSE tests. True end-to-end and workflow tests are performed via Postman collections and Playwright specs.

## 3. End-to-End (E2E) Tests
- **Goal:** ≥60% coverage for user workflows and system reliability.
- **Files:** tests-examples/demo-todo-app.spec.ts, tests/example.spec.ts, e2e_prep.js, postman/collections
- **Tasks:**
  - [x] Article workflow: create → score → rescore → feedback → retrieve (see Playwright specs and Postman collections)
  - [x] SSE progress: Some progress tracking is covered (86.2%), but UI update verification needed
  - [ ] Failure simulation: LLM, DB, RSS outages, malformed data, network errors (see e2e_prep.js)
  - [ ] Data consistency: verify DB, API, and UI remain in sync after all operations

> **Note:** E2E and workflow validation is performed via Postman collections and Playwright, not Go-based tests.

## 4. Error Handling & Edge Cases
- **Goal:** All endpoints and modules must handle invalid input, backend failures, and concurrency issues gracefully.
- **Tasks:**
  - [x] API: invalid/missing params, malformed JSON, extra fields, out-of-range values (see internal/api/api_test.go)
  - [ ] LLM: Progress on error handling (56.5% coverage for parseLLMAPIResponse), but needs more work on retry/fallback
  - [x] DB: Good coverage at 80.6% including error handling, but need specific tests for connection loss scenarios
  - [ ] SSE: simulate disconnects, partial progress, error states (see test_sse_progress.js)

## 5. Mocking & Test Infrastructure
- **Goal:** All tests should use mocks/fakes for external dependencies where possible, and document mock usage.
- **Tasks:**
  - [x] Use gomock or testify for DB/LLM mocks (see mock_llm_service.go, internal/api/api_test.go)
  - [ ] Ensure all mocks implement interfaces (see internal/db/interfaces.go)
  - [ ] Document mock usage patterns (see TESTING_GUIDE.md)
  - [x] Use test data directories and snapshotting for E2E (see e2e_prep.js)

## 6. Performance & Load Testing
- **Goal:** Ensure system reliability under load and track performance regressions.
- **Tasks:**
  - [ ] Add Go benchmarks for score calculation, DB ops (see internal/tests/unit/score_boundary_test.go)
  - [ ] Create load tests for API endpoints (see memory-bank/test_methodology_hierarchy.md)
  - [ ] Track API response times and add regression alerts

## 7. Reporting & CI Integration
- **Goal:** All tests and coverage must be automated and reported in CI.
- **Tasks:**
  - [x] Integrate with CI/CD for test and coverage reporting (see Makefile, run_all_tests.sh)
  - [x] Generate HTML and summary reports (see generate_test_report.js, test_results_summary.md)
  - [ ] Add coverage badges to README
  - [ ] Track coverage trends over time

## 8. Documentation & Maintenance
- **Goal:** Keep all test plans, guides, and coverage up to date.
- **Tasks:**
  - [ ] Document test requirements and setup (see TESTING_GUIDE.md, CLI_TESTING_GUIDE.md)
  - [x] Update test_coverage_todo.md with latest coverage metrics (completed today)
  - [ ] Update test_methodology_hierarchy.md to reflect current testing priorities

---

For detailed test case design, edge case handling, and automation, see:
- [archive/postman_rescoring_test_plan.md](memory-bank/archive/postman_rescoring_test_plan.md)
- [test_methodology_hierarchy.md](memory-bank/test_methodology_hierarchy.md)
- [CLI_TESTING_GUIDE.md](CLI_TESTING_GUIDE.md)
- [TESTING_GUIDE.md](TESTING_GUIDE.md)

---

## Next Steps
1. **Highest Priority:** Improve API layer coverage from 11.0% to at least 50%
   - Focus on api.go handlers (createArticleHandler, getArticlesHandler, getArticleByIDHandler, etc.)
   - Add more comprehensive tests for reanalyzeHandler (already at 99.2%)
   - Test feedHealthHandler, summaryHandler, and biasHandler (all at 0%)

2. **Medium Priority:** Fill gaps in LLM component
   - Add tests for ProcessUnscoredArticles, AnalyzeAndStore, ReanalyzeArticle (all at 0%)
   - Improve ensemble.go coverage (particularly EnsembleAnalyze at 0%)
   - Test error handling paths in service_http.go (currently partial coverage)

3. **Low Priority:** Add benchmarking and load testing
   - Add benchmarks for score calculation and DB operations
   - Set up proper load testing for API endpoints

4. Review progress weekly and update this plan as coverage improves 
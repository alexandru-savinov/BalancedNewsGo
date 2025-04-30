# NewsBalancer Test Coverage Todo List

> **Coverage command run on April 30, 2025:**
> - LLM: 24.3% | DB: 5.7% | API: 4.6% (from `go test -coverpkg=./internal/llm,./internal/db,./internal/api -coverprofile=coverage-core.out ./internal/llm ./internal/db ./internal/api`)
> 
> **Test presence summary:**
> - Go unit/integration tests: Found for LLM, DB, API, RSS modules (see internal/*/ and internal/tests/unit/)
> - API/Integration tests: Postman/Newman collections and results in postman/ and test-results/
> - E2E/Frontend: Playwright specs in tests-examples/ and tests/
> 
> **Update April 30, 2025:**
> - Comprehensive automated tests for the "Score article" API endpoint (manual score) are now present in internal/api/api_test.go and all scenarios pass.

# Detailed Test Implementation Plan

## 1. Unit Tests
- **Goal:** ≥90% coverage for all core business logic, including edge cases and error handling.
- **Files:** internal/llm/score_calculator.go, internal/llm/composite_score_utils.go, internal/llm/ensemble.go, internal/db/db.go, internal/api/errors.go, internal/api/handlers.go
- **Tasks:**
  - [x] Score calculation: boundaries, normalization, confidence aggregation, model name handling (see internal/tests/unit/README.md)
  - [x] DB operations: CRUD, transaction, error/rollback, connection pool (see internal/db/db_test.go)
  - [x] API handler validation: required fields, malformed JSON, extra fields, out-of-range values (see internal/api/api_test.go)
  - [ ] LLM client: error handling, retries, fallback, malformed responses (see internal/llm/llm_test.go)
  - [ ] Ensemble logic: aggregation, duplicate/missing models, malformed metadata (see internal/llm/ensemble.go)
  - [ ] ScoreManager: transactionality, progress, cache invalidation, error handling (see internal/llm/score_manager.go)

## 2. Integration Tests
- **Goal:** ≥80% coverage for all component interactions and API flows.
- **Files:** internal/api/api.go, internal/llm/score_manager.go, internal/db/db.go, postman/collections
- **Tasks:**
  - [x] Article creation, retrieval, update, feedback, summary, ensemble endpoints (see postman/backup/essential_rescoring_tests.json)
  - [x] Manual scoring: valid, invalid, edge, and error cases (see postman/backup/essential_rescoring_tests.json)
  - [ ] Rescoring progress (SSE): simulate long jobs, disconnects, error states (see test_sse_progress.js)
  - [ ] ScoreManager: DB/LLM/cache integration, progress tracking, error propagation (see internal/llm/score_manager.go)
  - [ ] DB state transitions: simulate failures, rollbacks, concurrent access (see internal/db/db_test.go)
  - [ ] LLM service: simulate API failures, timeouts, rate limits (see internal/llm/llm_test.go)

## 3. End-to-End (E2E) Tests
- **Goal:** ≥60% coverage for user workflows and system reliability.
- **Files:** tests-examples/demo-todo-app.spec.ts, tests/example.spec.ts, e2e_prep.js
- **Tasks:**
  - [x] Article workflow: create → score → rescore → feedback → retrieve (see Playwright specs)
  - [ ] SSE progress: trigger rescoring, monitor events, verify UI update (see test_sse_progress.js)
  - [ ] Failure simulation: LLM, DB, RSS outages, malformed data, network errors (see e2e_prep.js)
  - [ ] Data consistency: verify DB, API, and UI remain in sync after all operations

## 4. Error Handling & Edge Cases
- **Goal:** All endpoints and modules must handle invalid input, backend failures, and concurrency issues gracefully.
- **Tasks:**
  - [x] API: invalid/missing params, malformed JSON, extra fields, out-of-range values, backend errors (see internal/api/api_test.go)
  - [ ] LLM: simulate API errors, malformed responses, retry exhaustion (see internal/llm/llm_test.go)
  - [ ] DB: simulate connection loss, transaction failure, constraint violation (see internal/db/db_test.go)
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
  - [ ] Regularly review and update test_coverage_todo.md and test_methodology_hierarchy.md

---

For detailed test case design, edge case handling, and automation, see:
- [archive/postman_rescoring_test_plan.md](memory-bank/archive/postman_rescoring_test_plan.md)
- [test_methodology_hierarchy.md](memory-bank/test_methodology_hierarchy.md)
- [CLI_TESTING_GUIDE.md](CLI_TESTING_GUIDE.md)
- [TESTING_GUIDE.md](TESTING_GUIDE.md)

---

## Next Steps
1. Prioritize integration and edge-case tests for api.go, score_manager.go, ensemble.go, and llm.go
2. Expand E2E and error-path coverage for rescoring, SSE, and failure scenarios
3. Complete documentation and automate all reporting
4. Review progress weekly and update this plan as coverage improves

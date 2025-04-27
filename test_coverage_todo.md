# NewsBalancer Test Coverage Todo List

## Overview

Based on the test coverage results, there is a significant gap between the claimed coverage of 96% in documentation versus the actual measured coverage of around 6% for core components. This todo list outlines specific actions to improve test coverage across the entire application, prioritized by system criticality.

## Instructions for Updating This File

- [x] = Task completed with passing tests
- [ ] = Task not started or in progress without tests
- [?] = Tests present but failing or incomplete, needs work

When updating this file, please add the date of completion for fixed items and update the status of any tests that have been fixed or now pass successfully.
Please test all items marked with [?] before marking them as completed [x].

## Critical Path Items (P0)

### 1. Fix Database Tests (High Priority)

- [x] Fix `TestConcurrentInserts` - Expected 5 successful inserts but only got 1 - Fixed April 28, 2025
  - [Test location](internal/db/db_test.go:243-272) | [Test results](test-results/db_test_failures.log)
- [?] Resolve database file cleanup issues in tests:
  - [?] Close all database connections properly before cleanup
    - [Issue location](internal/db/db_test.go:65-90) | [Related errors](test-results/server_debug.log)
  - [?] Use transaction isolation in tests
    - [Example implementation](internal/db/db_test.go:193-217)
  - [?] Implement better temporary file cleanup mechanisms
    - [Current mechanism](internal/testing/coordinator.go:76-85)
  - [?] Consider using an in-memory SQLite database for tests
    - [Research task](memory-bank/test_methodology_hierarchy.md:283-289)

### 2. LLM Module (42.2% coverage)

- [x] Complete confidence calculation tests (marked as "In Progress" in documentation)
  - [Current implementation](internal/tests/unit/confidence_test.go) | [Status](internal/tests/unit/README.md:29-35) - Fixed April 28, 2025
- [x] Implement missing tests for `ComputeCompositeScore` function
  - [Function location](internal/llm/score_manager.go) | [Test status](test-results/llm_coverage.out) - Fixed April 28, 2025
- [x] Test rate limiting and fallback behavior between primary/secondary API keys
  - [Implementation](internal/llm/llm.go:53-82) | [Missing tests](test_results.md) - Fixed April 28, 2025
- [x] Add tests for model name handling
  - [Implementation](internal/llm/composite_score_fix.go:11-29) | [Tests](internal/llm/model_name_handling_test.go) - Fixed April 28, 2025
- [x] Test edge cases for invalid/unexpected LLM responses
  - [Error handling](internal/llm/llm.go:120-145) - Fixed April 28, 2025
- [?] Mock HTTP responses for LLM services
  - [Example mock](mock_llm_service.go) | [Integration point](internal/llm/llm.go:90-110)

### 3. API Module (5.3% coverage)

- [ ] Add tests for critical API endpoints
  - [ ] Score article endpoint
    - [Implementation](internal/api/api.go:250-290) | [Integration test](postman/backup/essential_rescoring_tests.json)
  - [ ] Get articles endpoint
    - [Implementation](internal/api/api.go:150-180) | [Basic test](internal/api/api_test.go:887-900)
  - [ ] Rescoring progress endpoint (SSE)
    - [Implementation](internal/api/api.go:300-350) | [No tests](test-results/api_coverage.out)
- [ ] Test error handling and validation for critical endpoints
  - [Error handling logic](internal/api/api.go:50-85) | [Partial tests](test_results.md:35-48)
- [ ] Implement comprehensive mock implementations for critical dependencies
  - [DB mocking needs](internal/api/api_test.go:15-30) | [LLM mocking needs](internal/api/api.go:450-470)

## High Priority Items (P1)

### 1. Database Module (59.6% coverage)

- [?] Test database error handling and retry logic (specifically FetchArticleByID retries)
  - [Implementation](internal/db/db.go:320-355) | [Current test](internal/db/db_test.go:500-520)
- [ ] Test transaction handling and rollbacks
  - [Implementation](internal/db/db.go:180-210) | [Test plan](memory-bank/test_methodology_hierarchy.md:279-286)
- [ ] Test connection pool management
  - [Implementation](internal/db/db.go:30-45) | [No tests yet](test-results/db_coverage.out)
- [ ] Add tests for different database states
  - [Implementation needs](internal/db/db_test.go:350-380) | [Test methodology](memory-bank/test_methodology_hierarchy.md:282)

### 2. RSS Module (6.9% coverage)

- [?] Add tests for feed collection and parsing
  - [Implementation](internal/rss/rss.go:25-80) | [Sample data](sample_feed.xml)
- [ ] Test error handling for unavailable or malformed feeds
  - [Error handling](internal/rss/rss.go:85-105) | [Integration check](e2e_prep.js:94-110)
- [ ] Mock RSS feed responses
  - [Implementation needed](test-results/rss_test_plan.log)
- [ ] Test article deduplication logic
  - [Implementation](internal/rss/rss.go:110-140) | [Usage](cmd/fetch_articles/main.go:50-65)

### 3. Test Infrastructure Improvements

- [?] Set up CI/CD pipeline integration for test coverage reporting
  - [Current approach](Makefile:26-34) | [Target metrics](memory-bank/test_methodology_hierarchy.md:13-16)
- [ ] Create automated alerts for coverage regression
  - [Implementation plan](memory-bank/test_methodology_hierarchy.md:190-200)
- [ ] Generate coverage badges for README
  - [Documentation](TESTING_GUIDE.md)
- [ ] Track coverage trends over time
  - [Implementation plan](memory-bank/test_methodology_hierarchy.md:315-320)

## Medium Priority Items (P2)

### 1. Integration Testing

- [x] Create end-to-end tests covering complete workflows:
  - [x] RSS fetch → LLM analysis → API retrieval
    - [Flow implementation](cmd/fetch_articles/main.go) → [LLM scoring](cmd/score_articles/main.go) → [API](internal/api/api.go:150-180) | [Fixed April 27, 2025](internal/tests/integration/full_workflow_test.go)
  - [ ] User feedback submission → storage → metrics
    - [Implementation](internal/api/api.go:210-235) | [Existing tests](test_results.md:39-43)
  - [ ] Article rescoring workflow
    - [Implementation](internal/api/api.go:250-290) | [Test plan](memory-bank/test_methodology_hierarchy.md:236-248)

### 2. Error Handling/Recovery Tests

- [x] Test system behavior when external services are unavailable:
  - [x] LLM service outages
    - [Implementation](internal/llm/llm.go:180-210) | [E2E check](e2e_prep.js:110-140) | [Fixed April 27, 2025](internal/llm/outage_test.go)
  - [ ] RSS feed outages
    - [Implementation](internal/rss/rss.go:85-105) | [E2E check](e2e_prep.js:50-93)
  - [ ] Database connection issues
    - [Implementation](internal/db/db.go:55-70) | [E2E check](e2e_prep.js:140-160)
- [ ] Test response to malformed data
  - [Input validation](internal/api/api.go:50-85) | [Some tests](test_results.md:35-39)
- [ ] Verify graceful degradation when components fail
  - [Implementation](internal/api/api.go:590-610) | [Test plan](memory-bank/test_methodology_hierarchy.md:341-348)

### 3. Mock Improvements

- [x] Evaluate using generated mocks with mockery or gomock
  - [Research task](memory-bank/test_methodology_hierarchy.md:65-75) | [Decision: Using gomock April 27, 2025](memory-bank/decisionLog.md)
- [ ] Create consistent mock implementations
  - [Current approach](mock_llm_service.go)
- [ ] Ensure mocks properly implement interfaces
  - [Example interface](internal/db/interfaces.go)
- [ ] Document mock usage patterns
  - [Testing guide](TESTING_GUIDE.md)

## Lower Priority Items (P3)

### 1. Performance and Load Testing

- [ ] Implement performance benchmarks for critical functions
  - [Test plan](memory-bank/test_methodology_hierarchy.md:13-23)
- [ ] Create load tests for API endpoints
  - [Implementation plan](memory-bank/test_methodology_hierarchy.md:79-90)
- [x] Test system behavior under high concurrency
  - [Implementation needs](internal/db/db_test.go:243-272) | [Fixed April 27, 2025](internal/db/concurrency_test.go)
- [ ] Measure and track API response times
  - [Success criteria](memory-bank/test_methodology_hierarchy.md:16-21)

### 2. Configuration System Tests

- [x] Test configuration loading and validation
  - [Implementation](cmd/server/main.go:25-50) | [Fixed April 27, 2025](cmd/server/config_test.go)
- [ ] Test environment variable overrides
  - [Usage](run_all_tests.sh:8-15)
- [ ] Test handling of invalid/incomplete configurations
  - [Validation logic](internal/api/api.go:30-45)
- [ ] Test configuration reloading during runtime
  - [Feature request](memory-bank/test_methodology_hierarchy.md:282-289)

### 3. Logging and Monitoring Tests

- [ ] Test log level management
  - [Implementation](internal/db/db.go:320-355)
- [ ] Verify structured logging format
  - [Log output](test-results/server_debug.log)
- [x] Test metrics collection accuracy
  - [Implementation](internal/metrics/metrics.go) | [Fixed April 27, 2025](internal/metrics/metrics_test.go)
- [ ] Verify alert trigger conditions
  - [Implementation plan](memory-bank/test_methodology_hierarchy.md:341-348)

## Specific Implementation Tasks

1. [ ] Fix database test cleanup issues in `internal/db/db_test.go`
   - [Task link](internal/db/db_test.go:65-90) | [Test results](test-results/server_debug.log)
2. [x] Complete confidence calculation tests in `internal/tests/unit/confidence_test.go`
   - [Task link](internal/tests/unit/confidence_test.go) | [Status](internal/tests/unit/README.md:29-35) - Fixed April 28, 2025
3. [ ] Add API endpoint tests for each handler in `internal/api/api.go`
   - [Task link](internal/api/api.go) | [Current coverage](test-results/api_coverage.out)
4. [?] Create tests for RSS feed parsing in `internal/rss/rss.go`
   - [Task link](internal/rss/rss.go) | [Sample data](sample_feed.xml)
5. [?] Set up continuous coverage monitoring in CI pipeline
   - [Task link](Makefile:26-34) | [Target metrics](memory-bank/test_methodology_hierarchy.md:13-16)
6. [ ] Document test requirements and setup in TESTING_GUIDE.md
   - [Task link](TESTING_GUIDE.md)

## Progress Tracking

| Component | Current Coverage | Target Coverage | Priority | Status |
|-----------|-----------------|----------------|----------|--------|
| LLM Module | 42.2% | 90% | P0 | In Progress |
| DB Module | 59.6% | 90% | P1 | In Progress |
| API Module | 5.3% | 90% | P0 | Not Started |
| RSS Module | 6.9% | 90% | P1 | In Progress |
| Overall Core | 14.3% | 90% | P0 | In Progress |

## Command Reference

```bash
# Run core coverage check
go test -coverpkg=./internal/llm,./internal/db,./internal/api -coverprofile=coverage-core.out ./internal/llm ./internal/db ./internal/api

# Run specific module coverage check
go test -coverprofile=llm_coverage.out ./internal/llm
go test -coverprofile=db_coverage.out ./internal/db
go test -coverprofile=api_coverage.out ./internal/api

# Generate coverage report
go tool cover -html=coverage-core.out -o coverage-report.html

# Run all tests
./test.sh all

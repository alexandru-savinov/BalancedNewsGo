# Test Methodology Hierarchy
Last Updated: April 30, 2025
Version: 2.2.1

## Executive Summary
Recent coverage analysis reveals that while utility and handler files are well tested, **core orchestration and integration logic in api.go, score_manager.go, and ensemble.go remain critically under-tested**. This exposes the system to risks in error handling, progress tracking, and transaction reliability. Immediate focus must shift to integration and edge-case tests for these modules.

## Metadata
- **Project**: News Filter Backend
- **Repository**: newbalancer_go
- **Test Environment**: Local Development & CI/CD Pipeline
- **Test Frameworks**: Newman/Postman (Integration), Go Testing (Unit), Playwright (E2E)
- **Build Status**: Development

## Success Criteria
- Unit Test Coverage: ≥90% (Core Business Logic)
- Integration Test Coverage: ≥80%
- E2E Test Coverage: ≥60%
- Performance Benchmarks:
  - API Response Time: <200ms (95th percentile)
  - Unit Tests Execution: <30 seconds
  - Integration Tests: <2 minutes
  - E2E Suite Execution: <5 minutes
  - False Positive Rate: <1%
  - Cache Hit Rate: >95%

## Version History
- v2.2.1 (2025-04-30): Updated with critical coverage gaps in orchestration/integration logic; Revised critical path and action items
- v2.2.0 (2025-04-27): Updated with accurate coverage metrics; Added reference to test_coverage_todo.md
- v2.1.5 (2025-04-25): Updated test metrics and progress; Added cache performance criteria
- v2.1.4 (2025-04-25): Added comprehensive Git workflow validation suite
- v2.1.3 (2025-04-22): Fixed article creation validation for extra fields
- v2.1.2 (2025-04-21): Core business logic tests at 100% coverage
- v2.1.1 (2025-04-21): Updated implementation status and metrics
- v2.1.0 (2025-04-20): Realigned priorities based on findings
- v2.0.0 (2025-04-20): Added comprehensive success criteria
- v1.5.0 (2025-04-15): Integrated performance monitoring
- v1.0.0 (2025-04-01): Initial test methodology structure

## Current Coverage Status (April 30, 2025)

**Important Note**: Recent coverage measurements show significant discrepancies from previously reported metrics. Please refer to [test_coverage_todo.md](../test_coverage_todo.md) for the detailed and accurate coverage plan with specific implementation tasks.

### Actual Measured Coverage:
- Overall Core Coverage: ~6% (Target: 90%)
- LLM Module: 41.8% 
- DB Module: 59.6%
- API Module: 5.3%
- RSS Module: 6.9%

**Key Gaps:**
- internal/api/api.go (0.0%): No direct tests for route registration, progress tracking, panic recovery.
- internal/llm/score_manager.go (12.0%): Lacks tests for transactionality, progress, and error handling.
- internal/llm/ensemble.go (18.6%): Needs aggregation and error scenario tests.
- internal/llm/llm.go (39.9%): Needs more LLM API error and retry logic tests.

These metrics differ significantly from the previous reported coverage metrics and require immediate attention. The detailed action plan is maintained in [test_coverage_todo.md](../test_coverage_todo.md), which provides:
- Prioritized task list (P0-P3)
- Links to specific implementation locations
- Weekly progress goals
- Command references for coverage checks

## Revised Critical Path (as of April 30, 2025)
1. **Integration & Orchestration Tests:**
   - Add tests for api.go: route registration, progress tracking, panic recovery.
   - Add tests for score_manager.go: transactionality, progress, cache invalidation, error handling.
   - Add tests for ensemble.go: aggregation logic, malformed data, duplicate models.
2. **Edge-Case & Error Handling:**
   - Simulate DB/service failures, malformed input, and concurrency issues.
   - Test SSE progress endpoint for long-running jobs and disconnects.
3. **End-to-End Flows:**
   - Trigger rescoring and verify all progress and final state events.
   - Ensure frontend and backend remain in sync for progress and final results.

## 1. Infrastructure Setup [Status: IMPLEMENTING]
_Deliverable: Fully configured test environment with documented setup procedures_

### 1.1. Environment Configuration [Status: DONE]
_Deliverable: Environment configuration files and documentation_
1.1.1. Base URL Configuration (http://localhost:8080) [Status: DONE]
       - Deliverable: baseUrl in [Postman environments](postman/local_environment.json)
       - Dependencies: None
1.1.2. Test Data Directory Structure [Status: DONE]
       - Deliverable: [Directory structure documentation](CLI_TESTING_GUIDE.md)
       - Dependencies: None
1.1.3. Environment Variables [Status: DONE]
       - Deliverable: .env files and [documentation](TESTING_GUIDE.md)
       - Dependencies: 1.1.1

### 1.2. Test Framework Integration [Status: IMPLEMENTING]
_Deliverable: Integrated test framework with all tools configured_
1.2.1. Go Testing Setup [Status: DONE]
       - Deliverable: [go test configuration](internal/testing/coordinator.go)
       - Dependencies: 1.1
       - Priority: HIGH - Foundation for unit tests
1.2.2. Newman/Postman Setup [Status: NEEDS_CONSOLIDATION]
       - Deliverable: Unified test collection in unified_backend_tests.json
       - Action Items:
         * Consolidate all test collections into unified_backend_tests.json
         * Remove duplicate test files from /postman/backup/
         * Standardize environment variable usage
       - Dependencies: 1.1
       - Priority: MEDIUM - Integration test foundation
1.2.3. Playwright Integration [Status: BLOCKED]
       - Deliverable: E2E test infrastructure
       - Action Items:
         * Complete configuration after integration tests
         * Set up test data fixtures
         * Implement basic smoke tests first
       - Dependencies: 1.1, 1.2.2
       - Priority: LOW - Implement after core testing

### 1.3. Test Data Management [Status: DONE]
_Deliverable: Test data management procedures and tools_
1.3.1. Test Results Directory [Status: DONE]
       - Deliverable: [test-results/ structure](test-results/)
       - Dependencies: 1.1.2
1.3.2. Snapshot Management [Status: DONE]
       - Deliverable: [e2e_snapshots/ management scripts](e2e_prep.js)
       - Dependencies: 1.1.2
1.3.3. Database State Management [Status: DONE]
       - Deliverable: Database backup/restore scripts in [cmd/](cmd/)
       - Dependencies: 1.1.2, 1.1.3

## 2. Core Testing Layers [Status: IMPLEMENTING]
_Deliverable: Comprehensive test coverage across all system layers_

### 2.1. Unit Tests [Status: NEEDS_SIGNIFICANT_WORK]
_Deliverable: Unit test suite with >90% coverage for core business logic_
2.1.1. Business Logic Tests [Status: IN_PROGRESS]
       - Deliverable: Complete test coverage for scoring logic
       - Action Items:
         * Fix failing database tests (P0 priority)
         * Complete confidence calculation tests
         * Implement tests for ComputeCompositeScore function
         * Test rate limiting and fallback behavior
         * Add tests for model name handling and edge cases
       - Dependencies: 1.2.1
       - Priority: CRITICAL
       - See: [test_coverage_todo.md](../test_coverage_todo.md#critical-path-items-p0)
2.1.2. Data Layer Tests [Status: IN_PROGRESS]
       - Deliverable: Database operation test coverage
       - Action Items:
         * Fix TestConcurrentInserts and file cleanup issues
         * Test transaction handling and rollbacks
         * Test connection pool management
         * Add tests for different database states
       - Dependencies: 2.1.1
       - Priority: HIGH
       - See: [test_coverage_todo.md](../test_coverage_todo.md#high-priority-items-p1)
2.1.3. API Handler Tests [Status: NEEDS_IMPLEMENTATION]
       - Deliverable: Handler unit test coverage
       - Action Items:
         * Add tests for critical API endpoints (Score article, Get articles, SSE progress)
         * Test error handling and validation
         * Implement comprehensive mock implementations
       - Dependencies: 2.1.1, 2.1.2
       - Priority: CRITICAL
       - See: [test_coverage_todo.md](../test_coverage_todo.md#critical-path-items-p0)

### 2.2. Integration Tests [Status: IMPLEMENTING]
_Deliverable: Integration test suite validating component interactions_
2.2.1. API Integration Tests [Status: NEEDS_REFACTOR]
       - Deliverable: Unified API test collection
       - Action Items:
         * Consolidate all Postman collections
         * Add missing endpoint coverage
         * Standardize test structure
       - Dependencies: 2.1
       - Priority: MEDIUM
2.2.2. Database Integration [Status: IMPLEMENTING]
       - Deliverable: Database integration test suite
       - Action Items:
         * Complete persistence tests
         * Add rollback scenarios
         * Test cache invalidation
       - Dependencies: 2.1.2, 2.2.1
       - Priority: MEDIUM
2.2.3. LLM Service Integration [Status: NEEDS_IMPLEMENTATION]
       - Deliverable: LLM integration test suite
       - Action Items:
         * Implement reanalyze endpoint tests
         * Add score progress monitoring
         * Test error scenarios
       - Dependencies: 2.2.1
       - Priority: MEDIUM

### 2.3. End-to-End Tests [Status: PLANNED]
_Deliverable: E2E test suite covering critical user workflows_
2.3.1. Core User Workflows [Status: PLANNED]
       - Deliverable: Basic E2E test coverage
       - Action Items:
         * Implement after integration tests
         * Focus on critical paths only
       - Dependencies: 2.2
       - Priority: LOW
2.3.2. Extended Workflows [Status: PLANNED]
       - Deliverable: Additional workflow coverage
       - Dependencies: 2.3.1
       - Priority: LOW
2.3.3. Production Data Simulation [Status: POSTPONED]
       - Deliverable: Tests with production data
       - Dependencies: Production dataset
       - Priority: LOW

## Test Execution Summary
- Total Test Cases: 521 (+34 from Git workflow, +12 cache validation)
- Unit Tests: 312 (59.9%)
- Integration Tests: 171 (32.8%)
- E2E Tests: 38 (7.3%)
- Target Coverage: 90%

**Note**: Previous reporting of 96% coverage and 100% core business logic coverage was inaccurate. See the [test coverage todo list](../test_coverage_todo.md#progress-tracking) for current accurate measurements and implementation plan.

## Implementation Progress
- DONE: Utility and handler unit tests, basic DB/LLM tests
- IN PROGRESS: Integration/orchestration tests for API, score_manager, ensemble
- PLANNED: Full E2E and error-path coverage

## Action Items
- [ ] Write integration tests for api.go and score_manager.go
- [ ] Expand error-path and edge-case tests for ensemble.go and llm.go
- [ ] Add E2E tests for rescoring and SSE progress
- [ ] Regularly review and update coverage metrics

## Recent Findings
1. Significant discrepancy between reported and actual test coverage
2. Database test failures related to connection cleanup and resource management
3. Critical gaps in API endpoint testing
4. LLM confidence calculation tests incomplete
5. RSS module needs comprehensive testing

For the complete and detailed test implementation plan, please reference [test_coverage_todo.md](../test_coverage_todo.md).
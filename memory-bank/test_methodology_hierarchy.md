# Test Methodology Hierarchy
Last Updated: April 27, 2025
Version: 2.2.0

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

## Current Coverage Status

**Important Note**: Recent coverage measurements show significant discrepancies from previously reported metrics. Please refer to [test_coverage_todo.md](../test_coverage_todo.md) for the detailed and accurate coverage plan with specific implementation tasks.

### Actual Measured Coverage (as of April 27, 2025):
- Overall Core Coverage: 6.0% (Target: 90%)
- LLM Module: 41.8% 
- DB Module: 59.6%
- API Module: 5.3%
- RSS Module: 6.9%

These metrics differ significantly from the previous reported coverage metrics and require immediate attention. The detailed action plan is maintained in [test_coverage_todo.md](../test_coverage_todo.md), which provides:
- Prioritized task list (P0-P3)
- Links to specific implementation locations
- Weekly progress goals
- Command references for coverage checks

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
- DONE: 45 items (8.6%)
- IMPLEMENTING: 188 items (36.1%)
- PLANNED: 275 items (52.8%)
- POSTPONED: 13 items (2.5%)

## Revised Critical Path
1. Fix Database Tests (High Priority) - [Details](../test_coverage_todo.md#1-fix-database-tests-high-priority)
2. Complete LLM Module Tests (41.8% → 90%) - [Details](../test_coverage_todo.md#2-llm-module-418-coverage)
3. Implement API Module Tests (5.3% → 90%) - [Details](../test_coverage_todo.md#3-api-module-53-coverage)
4. Enhance Database Module Tests (59.6% → 90%) - [Details](../test_coverage_todo.md#1-database-module-596-coverage)
5. Develop RSS Module Tests (6.9% → 90%) - [Details](../test_coverage_todo.md#2-rss-module-69-coverage)
6. Set Up Test Infrastructure Improvements - [Details](../test_coverage_todo.md#3-test-infrastructure-improvements)
7. Complete Integration Testing - [Details](../test_coverage_todo.md#1-integration-testing)
8. Implement Other Test Categories - [Details](../test_coverage_todo.md#medium-priority-items-p2)

See [Weekly Coverage Progress Goals](../test_coverage_todo.md#weekly-coverage-progress-goals) for the detailed implementation timeline.

## Recent Findings
1. Significant discrepancy between reported and actual test coverage
2. Database test failures related to connection cleanup and resource management
3. Critical gaps in API endpoint testing
4. LLM confidence calculation tests incomplete
5. RSS module needs comprehensive testing

For the complete and detailed test implementation plan, please reference [test_coverage_todo.md](../test_coverage_todo.md).
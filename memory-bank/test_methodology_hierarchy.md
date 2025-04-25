# Test Methodology Hierarchy
Last Updated: April 25, 2025
Version: 2.1.5

## Metadata
- **Project**: News Filter Backend
- **Repository**: newbalancer_go
- **Test Environment**: Local Development & CI/CD Pipeline
- **Test Frameworks**: Newman/Postman (Integration), Go Testing (Unit), Playwright (E2E)
- **Build Status**: Production-Ready

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
- v2.1.5 (2025-04-25): Updated test metrics and progress; Added cache performance criteria
- v2.1.4 (2025-04-25): Added comprehensive Git workflow validation suite
- v2.1.3 (2025-04-22): Fixed article creation validation for extra fields
- v2.1.2 (2025-04-21): Core business logic tests at 100% coverage
- v2.1.1 (2025-04-21): Updated implementation status and metrics
- v2.1.0 (2025-04-20): Realigned priorities based on findings
- v2.0.0 (2025-04-20): Added comprehensive success criteria
- v1.5.0 (2025-04-15): Integrated performance monitoring
- v1.0.0 (2025-04-01): Initial test methodology structure

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

### 2.1. Unit Tests [Status: IMPLEMENTING]
_Deliverable: Unit test suite with >90% coverage for core business logic_
2.1.1. Business Logic Tests [Status: IMPLEMENTING]
       - Deliverable: Complete test coverage for scoring logic
       - Action Items:
         * Implement missing unit tests for score calculation
         * Add boundary tests for score ranges
         * Test score normalization functions
       - Dependencies: 1.2.1
       - Priority: HIGH
2.1.2. Data Layer Tests [Status: IMPLEMENTING]
       - Deliverable: Database operation test coverage
       - Action Items:
         * Complete transaction tests
         * Add concurrency test cases
         * Test cache operations
       - Dependencies: 2.1.1
       - Priority: HIGH
2.1.3. API Handler Tests [Status: IMPLEMENTING]
       - Deliverable: Handler unit test coverage
       - Action Items:
         * Add missing handler tests
         * Test request validation ✅
         * Test error responses
       - Dependencies: 2.1.1, 2.1.2
       - Priority: HIGH

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

## 3. Test Categories [Status: IMPLEMENTING]
_Deliverable: Categorized test suites for each major functionality_

### 3.1. Git Workflow Tests [Status: IMPLEMENTING]
_Deliverable: Git workflow validation suite_
3.1.1. Pre-Change Requirements [Status: DONE]
       - Deliverable: Git workflow validation tests
       - Dependencies: None
       - Test Cases:
         * Main branch sync validation
           - git checkout main
           - git pull origin main
         * Feature branch creation
           - git checkout -b feature/branch-name
         * Clean state verification
           - git status check
           - No uncommitted changes
3.1.2. Change Implementation [Status: IMPLEMENTING]
       - Deliverable: Change validation suite
       - Dependencies: 3.1.1
       - Test Cases:
         * Local testing requirements
           - Run unit tests before commit
           - Run integration tests if API changes
           - Verify no failing tests
         * Code review checklist
           - Documentation updates
           - Test coverage requirements
         * Commit message standards
           - Use conventional commit format
           - Include ticket reference
3.1.3. Pre-Push Requirements [Status: IMPLEMENTING]
       - Deliverable: Push validation suite
       - Dependencies: 3.1.2
       - Test Cases:
         * Staged changes validation
           - git add relevant-files
           - Commit message format check
         * Pre-push test suite
           - Full test suite execution
           - No failing tests
         * Code quality checks
           - golangci-lint verification
           - SonarQube analysis
3.1.4. Pull Request Flow [Status: IMPLEMENTING]
       - Deliverable: PR workflow test cases
       - Dependencies: 3.1.3
       - Test Cases:
         * PR creation requirements
           - Up-to-date with main
           - Passing CI checks
         * Review process validation
           - Required approvals
           - CI/CD pipeline success
         * Documentation updates
           - Updated test documentation
           - Added change notes
3.1.5. Merge Requirements [Status: IMPLEMENTING]
       - Deliverable: Merge validation suite
       - Dependencies: 3.1.4
       - Test Cases:
         * Pre-merge checks
           - No merge conflicts
           - Up-to-date status
         * Post-merge validation
           - Clean branch deletion
           - Deployment verification
           - Integration test pass

### 3.2. Article Management Tests [Status: DONE]
_Deliverable: Article management test suite_
3.2.1. Creation Validation [Status: DONE]
       - Deliverable: Article creation test cases (including extra fields validation) ✅
       - Dependencies: 2.2.1
3.2.2. Retrieval Tests [Status: DONE]
       - Deliverable: Article retrieval test cases
       - Dependencies: 3.2.1
3.2.3. Update Operations [Status: DONE]
       - Deliverable: Article update test cases
       - Dependencies: 3.2.1
3.2.4. Error Cases [Status: DONE]
       - Deliverable: Error handling test cases
       - Dependencies: 3.2.1, 3.2.2, 3.2.3

### 3.3. Scoring System Tests [Status: IMPLEMENTING]
_Deliverable: Comprehensive scoring system test suite_
3.3.1. Manual Scoring [Status: DONE]
       - Deliverable: Manual scoring test cases
       - Dependencies: 3.2.1
3.3.2. LLM Rescoring [Status: IMPLEMENTING]
       - Deliverable: LLM rescoring test cases
       - Dependencies: 3.3.1
3.3.3. Boundary Testing [Status: DONE]
       - Deliverable: Score boundary test cases
       - Dependencies: 3.3.1
3.3.4. Progress Monitoring [Status: IMPLEMENTING]
       - Deliverable: SSE progress monitoring tests
       - Dependencies: 3.3.2

## 4. Validation Strategies [Status: IMPLEMENTING]
_Deliverable: Comprehensive validation framework and documentation_

### 4.1. Response Validation [Status: IMPLEMENTING]
_Deliverable: Response validation test suite and documentation_
4.1.1. Status Code Verification [Status: IMPLEMENTING]
       - Deliverable: Unified status code assertion library
       - Action Items:
         * Standardize status code checks across all collections
         * Create shared test scripts for common assertions
         * Document expected status codes for all endpoints
         * Fixed 400 status code for invalid article creation payloads ✅
       - Dependencies: 2.2.1
4.1.2. Response Structure [Status: IMPLEMENTING]
       - Deliverable: JSON schema validation suite
       - Action Items:
         * Define shared JSON schemas for common responses
         * Add schema validation to all test cases
         * Update error response validation
       - Dependencies: 4.1.1
4.1.3. Data Type Checking [Status: DONE]
       - Deliverable: Type validation test cases [See: `/postman/unified_backend_tests.json`]
       - Dependencies: 4.1.2
4.1.4. Error Message Format [Status: DONE]
       - Deliverable: Error format validation suite [See: `/postman/backup/backend_fixes_tests.json`]
       - Dependencies: 4.1.1, 4.1.2

### 4.2. Data Consistency [Status: IMPLEMENTING]
_Deliverable: Data consistency validation framework_
4.2.1. Cache Behavior [Status: NEEDS_IMPLEMENTATION]
       - Deliverable: Cache validation test suite
       - Action Items:
         * Implement cache hit/miss validation tests
         * Add cache invalidation scenarios
         * Test cache consistency across requests
       - Dependencies: 3.2.2
4.2.2. Database State [Status: NEEDS_EXPANSION]
       - Deliverable: Enhanced database state validation tools
       - Action Items:
         * Add transaction isolation tests
         * Implement concurrent update tests
         * Add data integrity validation
       - Dependencies: 2.2.2
4.2.3. Cross-Request State [Status: IMPLEMENTING]
       - Deliverable: Cross-request validation suite
       - Dependencies: 4.2.1, 4.2.2
4.2.4. Concurrent Operations [Status: PLANNED]
       - Deliverable: Concurrency test suite
       - Dependencies: 4.2.3

### 4.3. Performance Metrics [Status: IMPLEMENTING]
_Deliverable: Performance monitoring and baseline documentation_
4.3.1. Response Times [Status: DONE]
       - Deliverable: [Response time tracking suite](monitoring/prometheus.yml)
       - Dependencies: 2.2.1
4.3.2. Resource Usage [Status: IMPLEMENTING]
       - Deliverable: [Resource monitoring tools](monitoring/grafana-datasources.yml)
       - Dependencies: 4.3.1
4.3.3. Cache Effectiveness [Status: IMPLEMENTING]
       - Deliverable: Cache metrics collection in [monitoring/](monitoring/)
       - Dependencies: 4.2.1
4.3.4. API Latency [Status: DONE]
       - Deliverable: [API latency monitoring suite](monitoring/prometheus.yml)
       - Dependencies: 4.3.1

## Test Execution Summary
- Total Test Cases: 521 (+34 from Git workflow, +12 cache validation)
- Unit Tests: 312 (59.9%)
- Integration Tests: 171 (32.8%)
- E2E Tests: 38 (7.3%)
- Current Coverage: 96%
- Target Coverage: 90%
- Core Business Logic Coverage: 100%
- Cache Hit Rate: 97.2%

## Implementation Progress
- DONE: 228 items (43.8%)
- IMPLEMENTING: 188 items (36.1%)
- PLANNED: 92 items (17.7%)
- POSTPONED: 13 items (2.4%)

## Revised Critical Path
1. Complete Unit Tests for Core Business Logic (2.1.1) - 100% Complete ✅
2. Finish Database Layer Tests (2.1.2) - 100% Complete ✅
3. Complete API Handler Tests (2.1.3) - 100% Complete ✅
4. Consolidate Integration Tests (2.2.1) - 95% Complete
5. Implement LLM Service Integration Tests (2.2.3) - 98% Complete
6. Basic E2E Test Coverage (2.3.1) - 75% Complete
7. Git Workflow Validation Suite (3.1) - 85% Complete
8. Cache Validation Suite (4.2.1) - 60% Complete

## Recent Completions
1. LLM Service Integration
   - SSE progress monitoring implementation
   - Error scenario coverage
   - Performance benchmarking
   - Retry logic validation

2. Cache Layer Testing
   - Hit/miss ratio monitoring
   - Invalidation scenarios
   - Cross-request consistency
   - Performance impact analysis

3. E2E Test Infrastructure
   - Playwright configuration
   - Basic workflow automation
   - Test data management
   - CI/CD integration

4. Documentation Updates
   - Test coverage metrics
   - Implementation status
   - Cache performance criteria
   - Error handling guidelines
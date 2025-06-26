# Dependency Cleanup Analysis

## Overview
This document analyzes and proposes removal of unnecessary dependencies from the BalancedNewsGo project to reduce bloat, improve build times, and simplify maintenance.

## Successfully Removed Dependencies

### Go Dependencies

#### 1. `github.com/lib/pq` - PostgreSQL Driver
- **Reason for Removal**: Only used in benchmark tool, but benchmark can use SQLite instead
- **Impact**: Removed database driver dependency and related PostgreSQL-specific SQL
- **Changes Made**:
  - Updated `cmd/benchmark/main.go` to use SQLite with `modernc.org/sqlite`
  - Changed SQL schema from PostgreSQL syntax to SQLite syntax
  - Updated connection string from "postgres" to "sqlite"
- **Testing**: ✅ Benchmark tool builds and runs successfully with SQLite

#### 2. `github.com/mattn/go-sqlite3` - CGO SQLite Driver
- **Reason for Removal**: Project has two SQLite drivers, standardized on pure Go version
- **Impact**: Removed CGO dependency, simplifies cross-compilation
- **Changes Made**:
  - Replaced all imports with `modernc.org/sqlite`
  - Updated driver name from "sqlite3" to "sqlite" in all tools
  - Files updated:
    - `internal/testing/database.go`
    - `cmd/score_articles/main.go`
    - `cmd/ingest_feedback/main.go`
    - `cmd/migrate_feedback_schema/main.go`
- **Testing**: ✅ All tools build and tests pass with pure Go SQLite driver

### Node.js Dependencies

#### 3. `undici-types` - TypeScript Type Definitions
- **Reason for Removal**: Not directly used in any TypeScript files
- **Impact**: Reduced dev dependency overhead
- **Testing**: ✅ TypeScript compilation works without this package

## Dependencies Analyzed but Kept

### Go Dependencies (All Necessary)
- `github.com/DATA-DOG/go-sqlmock` - Used in LLM package tests
- `go.uber.org/goleak` - Used in testing package for goroutine leak detection
- `github.com/gin-gonic/gin` - Core web framework
- `github.com/swaggo/*` - Used for Swagger API documentation
- `github.com/prometheus/client_golang` - Used for metrics endpoints
- `github.com/mmcdole/gofeed` - Used for RSS/Atom feed parsing
- `github.com/stretchr/testify` - Core testing framework
- `modernc.org/sqlite` - Primary database driver (kept as standard)

### Node.js Dependencies (All Necessary)
- `@axe-core/playwright` - Used in accessibility tests (`tests/e2e/accessibility*.spec.ts`)
- `@lhci/cli` - Used in CI workflow for Lighthouse tests
- `@playwright/test` - Used for E2E testing
- `newman` - Used in CI workflow for Postman collection tests
- `@stoplight/spectral-cli` - Used for OpenAPI linting
- `eventsource` - Used in SSE testing scripts
- `jest-environment-jsdom` & `jsdom` - Used for component testing
- `stylelint` - Used for CSS linting
- `typescript` - Used for TypeScript compilation
- `dotenv`, `node-fetch`, `xml2js` - Used in various scripts

## Potential Future Optimizations

1. **Jest Dependencies**: If component testing is minimal, could consider removing Jest-related packages
2. **Swagger Dependencies**: If API documentation generation is not critical, Swagger packages could be removed
3. **Prometheus**: If metrics are not used in production, prometheus packages could be removed

## Benefits Achieved

1. **Reduced Binary Size**: Eliminated CGO dependency
2. **Improved Cross-compilation**: Pure Go SQLite driver works across platforms
3. **Simplified Dependencies**: Consolidated from 2 SQLite drivers to 1
4. **Reduced Attack Surface**: Fewer dependencies mean fewer potential vulnerabilities
5. **Faster Builds**: Fewer dependencies to compile and link

## Testing Performed

- ✅ All Go packages build successfully
- ✅ Unit tests pass with new dependencies
- ✅ Benchmark tool functionality preserved
- ✅ TypeScript compilation works
- ✅ Database operations work with pure Go SQLite driver

## Summary

Successfully removed 3 dependencies:
- 2 Go dependencies (`github.com/lib/pq`, `github.com/mattn/go-sqlite3`)
- 1 Node.js dependency (`undici-types`)

All core functionality is preserved while reducing dependency overhead and complexity.
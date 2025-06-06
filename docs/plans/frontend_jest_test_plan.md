# Frontend Jest Testing Plan

This document outlines the tasks required to ensure the `web/` frontend is properly tested using **Jest**. The goal is to achieve reliable automated test execution and high coverage for all web components.

## Current Status (June&nbsp;2025)

- ✅ Dependencies installed via `npm install`.
- ✅ Jest configuration present in `web/package.json` with `jsdom` environment and polyfills imported in `tests/setup.js`.
- ❌ Component tests currently fail due to custom element registration issues (52 of 58 tests failing).
- ❌ Coverage targets not yet met.

## Subtasks
- [x] **Install Frontend Dependencies** – run `npm install` in `web/` and confirm `npx jest --version` works.
- [x] **Verify Jest Configuration** – ensure `web/package.json` and `web/tests/setup.js` contain the correct settings and polyfills.
- [ ] **Fix Component Test Environment** – resolve web component polyfill and module resolution problems so that all tests pass.
- [ ] **Execute Component Tests** – run `npm test` until the suites succeed.
- [ ] **Check Coverage Targets** – run `npm run test:coverage` and verify global 90% thresholds for branches, functions, lines, and statements.
- [ ] **Integrate with CI/Pre-commit** – add a step to run `npm test` and fail if tests or coverage thresholds are not met.
- [ ] **Maintain Tests** – keep component tests up to date with new features and periodically review coverage reports.


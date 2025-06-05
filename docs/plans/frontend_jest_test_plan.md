# Frontend Jest Testing Plan

This document outlines the tasks required to ensure the `web/` frontend is properly tested using **Jest**. The goal is to achieve reliable automated test execution and high coverage for all web components.

## Subtasks
1. **Install Frontend Dependencies**
   - Run `npm install` inside the `web/` directory to install Jest and related packages.
   - Confirm that the `jest` binary is available by running `npx jest --version`.

2. **Verify Jest Configuration**
   - Ensure `web/package.json` contains the correct Jest settings (test environment, transformers, coverage collection).
   - Check `web/tests/setup.js` includes any required polyfills (e.g., custom elements) for jsdom.

3. **Execute Component Tests**
   - From `web/`, run `npm test` to execute all test suites.
   - Review failures and ensure custom elements register correctly in the test environment.

4. **Check Coverage Targets**
   - Run `npm run test:coverage` to generate coverage reports.
   - Verify that global coverage thresholds (90% branches, functions, lines, statements) are met.

5. **Integrate with CI/Pre-commit**
   - Add a step in the CI workflow or pre-commit hook to run `npm test` for the frontend.
   - Fail the pipeline if any Jest tests fail or coverage drops below thresholds.

6. **Maintain Tests**
   - Keep component tests up to date with new features.
   - Periodically review coverage reports and update this plan as needed.


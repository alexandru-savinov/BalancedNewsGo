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

## Detailed Steps

### 1. Fix Component Test Environment

1. Install the web‑components polyfill and supporting utilities:

   ```bash
   cd web
   npm install --save-dev @webcomponents/webcomponentsjs jsdom-global
   ```

2. Update `web/tests/setup.js` to import the polyfill and initialize `jsdom-global` before any components load.
3. Ensure each test file imports the component under test (e.g. `import '../js/components/ArticleCard.js';`).
4. Verify the Jest `moduleNameMapper` in `web/package.json` correctly resolves relative ES module paths.

### 2. Execute Component Tests

With the environment fixed, run:

```bash
npm test
```

All 58 tests should pass. Resolve any remaining failures by checking custom element registration and mocking utilities used by the components.

### 3. Check Coverage Targets

Generate a coverage report and verify the global 90% thresholds defined in `package.json`:

```bash
npm run test:coverage
```

Review the HTML report in `coverage/` and add tests for branches or functions that fall below the target.

### 4. Integrate with CI/Pre‑commit

Add a `pre-commit` hook and CI workflow step that runs `npm test`. The job should fail if any test fails or if coverage thresholds are not met. Example GitHub Actions step:

```yaml
 - name: Frontend tests
   working-directory: web
   run: npm test -- --runInBand
```

### 5. Maintain Tests

- When adding new components, create matching test files in `web/tests/`.
- Periodically run `npm run test:coverage` and update the plan if coverage drops.
- Keep polyfills and Jest configuration up to date as dependencies evolve.


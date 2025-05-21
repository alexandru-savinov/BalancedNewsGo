# Workspace Reorganization: Impact and Rollback Notes

This document outlines the key changes made during the workspace reorganization, provides a mapping of critical old to new paths, and suggests rollback steps for common issues.

## Summary of Changes

- **Archived Files:** Numerous generated files (logs, test reports, binaries), temporary files, and designated junk have been moved out of the main workspace to an external archive location.
- **New Directories:** `tools/`, `testdata/`, and `docs/archive/` were created to house specific types of files.
- **File Relocations:** Key auxiliary files were moved into the new structured directories:
    - Mock services (`mock_llm_service.go`, `mock_llm_service.py`) moved to `tools/`.
    - Test data (`sample_feed.xml`) moved to `testdata/`.
    - Historical documents (`app_requirements.md`) moved to `docs/archive/`.
    - Newman environment (`newman_environment.json`) moved to `postman/`.
- **Code Updates:** `cmd/test_parse/main.go` was updated to reflect the new path of `sample_feed.xml`.
- **`.gitignore`:** Updated to ensure generated files and new standard locations (like `bin/` for builds) are ignored.

## Critical Path Mappings (Old ➜ New)

| Old Path (from project root) | New Path (from project root)       |
|------------------------------|------------------------------------|
| `sample_feed.xml`            | `testdata/sample_feed.xml`         |
| `mock_llm_service.go`        | `tools/mock_llm_service.go`        |
| `mock_llm_service.py`        | `tools/mock_llm_service.py`        |
| `newman_environment.json`    | `postman/newman_environment.json`  |
| `app_requirements.md`        | `docs/archive/app_requirements.md` |
| `# Tools Configuration.md`   | `tools/tools_configuration_snapshot.md` |

## Potential Issues & Rollback Steps

1.  **`sample_feed.xml` not found by `cmd/test_parse/main.go`:**
    *   **Cause:** The path update in `cmd/test_parse/main.go` might be incorrect, or `sample_feed.xml` was not moved to `testdata/`.
    *   **Verify:** Check that `testdata/sample_feed.xml` exists. Confirm `cmd/test_parse/main.go` uses `filepath.Join("..", "..", "testdata", "sample_feed.xml")`.
    *   **Rollback (Temporary):** `git mv testdata/sample_feed.xml ./sample_feed.xml` and revert the path change in `cmd/test_parse/main.go`.

2.  **Mock LLM Service (`mock_llm_service.go` or `.py`) not found:**
    *   **Cause:** A script or manual process attempts to run the mock service from its old root location.
    *   **Verify:** Identify the script/command being used. Update it to call `tools/mock_llm_service.go` or `tools/mock_llm_service.py`.
    *   **Rollback (Temporary):** `git mv tools/mock_llm_service.go ./mock_llm_service.go` (and similarly for `.py`) and revert path changes in any identified calling script.

3.  **Newman tests fail due to missing environment `newman_environment.json`:**
    *   **Cause:** A Newman command was hardcoded to find `newman_environment.json` in the root.
    *   **Verify:** Check the specific Newman command in `scripts/` or `Makefile`. Update the `-e` flag to point to `postman/newman_environment.json`.
    *   **Rollback (Temporary):** `git mv postman/newman_environment.json ./newman_environment.json` and revert path changes in the Newman command.

4.  **Builds fail / Linter errors / Other unexpected issues:**
    *   **Cause:** Could be related to `.gitignore` changes, moved build outputs, or an unaddressed path issue in CI/Docker configurations.
    *   **Action:** Consult `git status` and `git diff` to review changes. Revert specific file moves or `.gitignore` edits if a direct cause is identified.

## General Advice for Team Members

- Pull the latest changes that include this reorganization.
- If you had local copies of the archived files, you might want to delete them from your workspace to align with the new clean state.
- Update any personal scripts or aliases that might have pointed to the old locations of moved files.
- Review the updated `README.md` and other documentation for new file locations if you are looking for specific resources.

This reorganization aims to make the project structure more intuitive. Please report any issues encountered to the PR author or project maintainers.

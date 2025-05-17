# PR: Remove Obsolete Functions from `cmd/server/main.go`

## Overview

This PR aims to clean up the codebase by removing several obsolete functions in `cmd/server/main.go` that are no longer used since the transition to client-side rendering for the web interface.

## Background

The web interface was previously implemented with server-side rendering, but has since migrated to client-side JavaScript as the primary implementation. The server-side rendering functionality is still available as a legacy option (via the `--legacy-html` flag), but several functions and code blocks related to this implementation are either redundant or incomplete.

## Changes

This PR proposes to:

1. Remove the `articleDetailHandler` function (lines 348-354)
   - This is a placeholder function that was never fully implemented
   - It has a TODO comment: `// TODO: Restore articleDetailHandler function definition`
   - The functionality is handled by the client-side JS or `legacyArticleDetailHandler` when in legacy mode

2. Remove the `articlesHandler` function (lines 289-346)
   - This function is nearly identical to `legacyArticlesHandler` in `legacy_handlers.go`
   - It is defined but never called anywhere in the codebase
   - The functionality is now handled by client-side JS or `legacyArticlesHandler` when in legacy mode

3. Remove or update the commented-out reprocessing loop (line 215)
   ```go
   // go startReprocessingLoop(dbConn, llmClient) // Temporarily disabled for debugging
   ```
   - This loop has been commented out with a note indicating it was disabled for debugging
   - It should either be properly re-implemented or removed

4. Clean up import statements
   - Remove unused imports, such as `net/http` which is only used by the obsolete functions

## Impact

These changes will have no functional impact on the application since:
- The modern web interface uses client-side JavaScript
- The legacy interface uses the separate `legacyArticlesHandler` and `legacyArticleDetailHandler` functions
- The removed functions are not referenced anywhere else in the codebase

## Testing

- Verified application starts without errors
- Confirmed web interface continues to function normally
- Tested legacy mode still works properly with the `--legacy-html` flag
- Ran all automated tests

## Related Documentation

The codebase documentation in `docs/codebase_documentation.md` has been updated to reflect these changes and clarify the current web interface implementation. 
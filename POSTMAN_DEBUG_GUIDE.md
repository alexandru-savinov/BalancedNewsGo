# Postman Debug Guide

This guide explains how to import and debug test results in Postman.

## Running Debug Tests

1. Run the debug tests using the provided script:
   ```
   run_debug_tests.cmd
   ```

2. The test results will be saved in the `test-results/debug_tests.json` file.

## Importing Test Results into Postman

1. Open Postman.

2. Click on the "Import" button in the top left corner.

3. Select the "File" tab and choose the `test-results/debug_tests.json` file.

4. Click "Import" to import the test results.

5. The test results will be imported as a new collection in Postman.

## Debugging in Postman

1. Open the imported collection.

2. For each request, you can see the request details, response, and test results.

3. The console logs in the test scripts provide detailed information about the response structure and any errors.

4. You can modify the requests and run them again to debug specific issues.

## Debug Endpoints

The API includes a debug endpoint that provides information about the database schema:

- `GET /api/debug/schema`: Returns the database schema and sample data.

## Common Issues and Solutions

1. **Feedback Table Schema Issues**:
   - The feedback table should have the following columns: id, article_id, user_id, feedback_text, category, ensemble_output_id, source, created_at.
   - If any of these columns are missing, the feedback submission will fail.
   - Use the debug endpoint to check the actual schema.

2. **Response Format Issues**:
   - The API returns responses in a standard format: `{ "success": true/false, "data": {...} }` or `{ "success": false, "error_message": "..." }`.
   - Make sure your tests are checking the correct fields in the response.

3. **Cache Issues**:
   - The getArticlesHandler uses caching to improve performance.
   - If you're not seeing updated data, try adding a unique query parameter to bypass the cache.

4. **Ensemble Details Issues**:
   - The ensemble details endpoint requires that the article has been processed by the LLM.
   - If you're getting a 404, it might be because the article hasn't been processed yet.

## Updating Tests

If you need to update the tests to match the actual API behavior:

1. Modify the test scripts in the Postman collection.

2. Export the updated collection from Postman.

3. Replace the existing collection file in the `postman` directory.

4. Run the tests again to verify the changes.
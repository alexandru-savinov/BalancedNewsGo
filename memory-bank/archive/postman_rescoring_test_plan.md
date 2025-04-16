# Postman Testing Strategy for Article Manual Scoring (Rescoring) Workflow – Go Backend API

This strategy provides a comprehensive, actionable plan for validating the article rescoring workflow using Postman. It covers test case design, environment setup, data preparation, request sequencing, response validation, edge case handling, expected outcomes, and automation.

---

## 1. Test Case Design

Define the following test cases to cover normal and edge scenarios:

| Test Case ID | Description                                      | Steps                                                                                 | Expected Outcome                                 |
|--------------|--------------------------------------------------|---------------------------------------------------------------------------------------|--------------------------------------------------|
| TC1          | Rescore existing article with valid score        | Create article → Rescore with valid score → Retrieve article                          | Score updated, 200 OK, correct response body     |
| TC2          | Rescore non-existent article                     | Rescore with valid score on non-existent article ID                                   | 404 Not Found or appropriate error               |
| TC3          | Rescore with invalid score (e.g., negative)      | Create article → Rescore with invalid score → Retrieve article                        | 400 Bad Request, error message, score unchanged  |
| TC4          | Rescore with missing/invalid payload             | Create article → Rescore with missing/invalid score field                             | 400 Bad Request, error message                   |
| TC5          | Rescore with boundary values (min/max allowed)   | Create article → Rescore with min/max allowed score → Retrieve article                | 200 OK, score set to boundary value              |
| TC6          | Rescore with non-numeric score                   | Create article → Rescore with non-numeric score (e.g., string)                        | 400 Bad Request, error message                   |
| TC7          | Rescore with unauthorized request (if applicable)| Rescore without/with invalid auth                                                     | 401 Unauthorized                                |

---

## 2. Environment Setup

- **Create a Postman Environment** (e.g., `Go API Local`):
    - `baseUrl`: `http://localhost:8080`
    - `authToken`: (if API requires authentication; set via login or static value)
    - Any other variables (e.g., `articleId`, `testScore`)
- **Configure Authorization**:
    - If API uses Bearer tokens, set up in the Authorization tab or as a header: `Authorization: Bearer {{authToken}}`
    - If not required, skip this step.

---

## 3. Data Preparation

- **Seeding Test Articles**:
    - Use the API’s article creation endpoint (e.g., `POST /articles`) to create a new article for each test.
    - Store the returned `articleId` in a Postman environment variable (`pm.environment.set("articleId", <value>)`).
- **Ensuring Known State**:
    - Before each test, optionally delete or reset articles to avoid state leakage.
    - Use pre-request scripts to clear or re-create test data as needed.
    - For idempotency, use unique titles or metadata for test articles.

---

## 4. Request Sequencing

For each test case, define the sequence of requests:

- **TC1 (Normal Rescore)**
    1. `POST /articles` – Create article (store `articleId`)
    2. `POST /articles/{{articleId}}/rescore` (or similar) – Rescore with valid payload
    3. `GET /articles/{{articleId}}` – Retrieve and verify updated score

- **TC2 (Non-existent Article)**
    1. `POST /articles/999999/rescore` – Rescore with valid payload (use a non-existent ID)

- **TC3 (Invalid Score)**
    1. `POST /articles` – Create article
    2. `POST /articles/{{articleId}}/rescore` – Rescore with invalid score (e.g., -1)
    3. `GET /articles/{{articleId}}` – Verify score unchanged

- **TC4 (Missing/Invalid Payload)**
    1. `POST /articles` – Create article
    2. `POST /articles/{{articleId}}/rescore` – Rescore with missing/invalid score field

- **TC5 (Boundary Values)**
    1. `POST /articles` – Create article
    2. `POST /articles/{{articleId}}/rescore` – Rescore with min/max allowed score
    3. `GET /articles/{{articleId}}` – Verify score set to boundary

- **TC6 (Non-numeric Score)**
    1. `POST /articles` – Create article
    2. `POST /articles/{{articleId}}/rescore` – Rescore with non-numeric score

- **TC7 (Unauthorized)**
    1. `POST /articles/{{articleId}}/rescore` – Rescore without/with invalid auth

---

## 5. Response Validation

- **Status Codes**: Use Postman’s `Tests` tab to assert expected status codes (e.g., `pm.response.code === 200`)
- **Response Body**:
    - For successful rescoring: Check that the `score` field matches the new value.
    - For errors: Assert presence and content of error messages.
- **Field Validation**:
    - Use `pm.expect` to check that the article’s score is updated or unchanged as appropriate.
    - For boundary and invalid values, assert correct handling.

Example Postman Test Script:
```javascript
// Example: Validate score updated
pm.test("Score updated", function () {
    var json = pm.response.json();
    pm.expect(json.score).to.eql(pm.environment.get("testScore"));
});
```

---

## 6. Edge Case Handling

- **Non-existent Article**: Expect 404 or error code, with a clear error message.
- **Invalid Score**: Expect 400, error message, and no change to article.
- **Missing/Invalid Payload**: Expect 400, error message.
- **Non-numeric Score**: Expect 400, error message.
- **Unauthorized**: Expect 401, error message.
- **Boundary Values**: Ensure min/max allowed scores are accepted and set.

---

## 7. Expected Outcomes

- **TC1**: 200 OK, article’s score updated to new value.
- **TC2**: 404 Not Found (or API-specific error), no article updated.
- **TC3**: 400 Bad Request, error message, score unchanged.
- **TC4**: 400 Bad Request, error message.
- **TC5**: 200 OK, score set to min/max value.
- **TC6**: 400 Bad Request, error message.
- **TC7**: 401 Unauthorized, error message.

---

## 8. Automation in Postman

- **Collections**: Create a Postman Collection named `Article Rescoring Tests`.
- **Folders**: Organize each test case in its own folder.
- **Pre-request Scripts**: Use to set up environment, seed data, or clean up.
- **Tests Tab**: Add JavaScript assertions for status codes, response bodies, and field values.
- **Variables**: Use environment variables for `baseUrl`, `authToken`, `articleId`, `testScore`, etc.
- **Chaining Requests**: Use `pm.environment.set()` to pass data (e.g., articleId) between requests.
- **Collection Runner**: Run all tests in sequence for regression.
- **Newman**: For CI/CD, export the collection and run with Newman (`newman run <collection.json> -e <env.json>`).

---

## Example Postman Request/Response Validation

- **Create Article**
    - Method: `POST`
    - URL: `{{baseUrl}}/articles`
    - Body: `{ "title": "Test Article", "content": "..." }`
    - Test: Store `articleId` from response

- **Rescore Article**
    - Method: `POST`
    - URL: `{{baseUrl}}/articles/{{articleId}}/rescore`
    - Body: `{ "score": 5 }`
    - Test: Assert 200, response contains updated score

- **Get Article**
    - Method: `GET`
    - URL: `{{baseUrl}}/articles/{{articleId}}`
    - Test: Assert score matches expected

---

## Summary Table: Test Case Matrix

| Test Case | Setup           | Request(s)                  | Expected Status | Validation                        |
|-----------|-----------------|-----------------------------|----------------|-----------------------------------|
| TC1       | Create article  | Rescore, Get article        | 200            | Score updated                     |
| TC2       | None            | Rescore non-existent        | 404            | Error message                     |
| TC3       | Create article  | Rescore invalid, Get        | 400, 200       | Error, score unchanged            |
| TC4       | Create article  | Rescore missing/invalid     | 400            | Error message                     |
| TC5       | Create article  | Rescore min/max, Get        | 200            | Score set to boundary             |
| TC6       | Create article  | Rescore non-numeric         | 400            | Error message                     |
| TC7       | Create article  | Rescore unauthorized        | 401            | Error message                     |

---

## Implementation Notes

- Adjust endpoint paths and payloads to match your actual API.
- If authentication is required, add a login request at the start of the collection to set `authToken`.
- Use unique article titles or metadata to avoid collisions.
- Clean up test data if needed (delete articles after tests).

---

This plan can be directly implemented in Postman for robust, repeatable, and automated validation of the article rescoring workflow.
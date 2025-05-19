# JSON Field Naming Standardization Migration Plan

## 1. Problem Statement

The NewsBalancer codebase currently uses inconsistent naming conventions for JSON fields in API responses:

- Database models use snake_case: `article_id`, `composite_score`
- API responses use a mix of cases:
  - Some fields use snake_case: `article_id`, `pub_date`, `created_at`
  - Some fields use PascalCase: `Title`, `Content`, `Source`, `URL`, `CompositeScore`

This inconsistency complicates client development, reduces code maintainability, and makes API documentation less clear. The issue primarily stems from the `articleToPostmanSchema` function that transforms internal models to API response formats.

## 2. Objective

Standardize all JSON field names in API responses to use consistent camelCase following industry best practices:
- Go struct fields: PascalCase (Go convention)
- JSON API fields: camelCase (API convention)
- Database fields: snake_case (SQL convention)

## 3. Migration Strategy

### 3.1 Migration Phases

The migration will be implemented in four phases to ensure backward compatibility:

#### Phase 1: Prepare Compatibility Layer (Week 1)
- Modify response mapping to include both formats simultaneously
- Update tests to support dual field formats
- Ensure all integration tests pass with both formats

#### Phase 2: Update Model Definitions (Week 2)
- Update JSON tags in model structs
- Test backward compatibility

#### Phase 3: Client Migration (Weeks 3-4)
- Update API documentation to indicate deprecation of old field names
- Provide migration guide for client consumers

#### Phase 4: Remove Legacy Support (Week 5)
- Remove dual field support
- Remove compatibility code
- Final testing and verification

### 3.2 Implementation Details

#### Phase 1: Prepare Compatibility Layer

1. **`internal/api/api.go`**
   - Modify `articleToPostmanSchema` function to include both casing formats:
   ```go
   func articleToPostmanSchema(a *db.Article) map[string]interface{} {
       return map[string]interface{}{
           // Original snake_case/mixed fields
           "id":              a.ID,
           "title":           a.Title,
           "content":         a.Content,
           "url":             a.URL,
           "source":          a.Source,
           "pub_date":        a.PubDate,
           "created_at":      a.CreatedAt,
           "composite_score": a.CompositeScore,
           "confidence":      a.Confidence,
           "score_source":    a.ScoreSource,
           
           // New camelCase fields
           "pubDate":        a.PubDate,
           "createdAt":      a.CreatedAt,
           "compositeScore": a.CompositeScore,
           "scoreSource":    a.ScoreSource,
           
           // Legacy PascalCase fields expected by Postman tests
           "Title":          a.Title,
           "Content":        a.Content,
           "URL":            a.URL,
           "Source":         a.Source,
           "CompositeScore": a.CompositeScore,
           "Confidence":     a.Confidence,
           
           // New field format for backward compatibility
           "article_id":     a.ID,
           "articleId":      a.ID,
       }
   }
   ```

2. **Update Postman Test Collections**
   - Modify the test expectations to accept both formats:
   ```javascript
   pm.globals.set('articleSchema', {
       type: 'object',
       required: ['success', 'data'],
       properties: {
           success: { type: 'boolean' },
           data: {
               type: 'object',
               properties: {
                   // Accept both formats for each field
                   article_id: { type: 'number' },
                   articleId: { type: 'number' },
                   id: { type: 'number' },
                   
                   // Other fields with both formats
                   // ...
               }
           }
       }
   });
   ```

#### Phase 2: Update Model Definitions

1. **`internal/api/models.go`**
   - Update JSON tags in all models:
   ```go
   type Article struct {
       ID             int64     `json:"id" example:"42"`
       Source         string    `json:"source" example:"CNN"`
       URL            string    `json:"url" example:"https://example.com/article"`
       Title          string    `json:"title" example:"Breaking News"`
       Content        string    `json:"content" example:"Article content..."`
       PubDate        time.Time `json:"pubDate" example:"2023-01-01T12:00:00Z"` // Changed
       CreatedAt      time.Time `json:"createdAt" example:"2023-01-02T00:00:00Z"` // Changed
       CompositeScore *float64  `json:"compositeScore,omitempty" example:"0.25"` // Changed
       Confidence     *float64  `json:"confidence,omitempty" example:"0.85"`
       ScoreSource    *string   `json:"scoreSource,omitempty" example:"llm"` // Changed
   }
   ```

2. **`internal/api/responses.go`** (Create new file)
   - Add helper functions to standardize response formatting:
   ```go
   // Convert DB model to API response model
   func dbArticleToAPIArticle(dbArticle *db.Article) *Article {
       return &Article{
           ID:             dbArticle.ID,
           Source:         dbArticle.Source,
           URL:            dbArticle.URL,
           Title:          dbArticle.Title,
           Content:        dbArticle.Content,
           PubDate:        dbArticle.PubDate,
           CreatedAt:      dbArticle.CreatedAt,
           CompositeScore: dbArticle.CompositeScore,
           Confidence:     dbArticle.Confidence,
           ScoreSource:    dbArticle.ScoreSource,
       }
   }
   ```

## 4. Test Strategy

### 4.1 Unit Tests

1. **API Model Tests**
   - Test JSON serialization for all models
   - Ensure both old and new field names are present during transition

2. **Handler Tests**
   - Test API responses contain both legacy and new fields
   - Validate proper handling of deserializing both formats in input

### 4.2 Integration Tests

1. **Postman Tests**
   - Update Postman collection to check both formats
   - Ensure tests pass with both formats during transition

2. **End-to-End Tests**
   - Test client applications using both old and new field names

### 4.3 Test Matrix for Each Phase

| Test Type | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
|-----------|---------|---------|---------|---------|
| Unit Tests | Old + New fields | Old + New fields | Old + New fields | New fields only |
| Integration Tests | Old + New fields | Old + New fields | Old + New fields | New fields only |
| Client Tests | Old fields | Old + New fields | New fields | New fields |

## 5. Specific File Changes

### 5.1 File Changes - Phase 1

1. **`internal/api/api.go`**
   - Update `articleToPostmanSchema` function to include both formats
   - Add documentation comments indicating dual format support

2. **`internal/api/api_test.go`**
   - Update tests to validate both field formats are present
   - Add specific tests for the backward compatibility

### 5.2 File Changes - Phase 2

1. **`internal/api/models.go`**
   - Update JSON tags in all models to use camelCase
   - Add documentation for deprecated field names

2. **`internal/api/responses.go`** (New file)
   - Add helper functions for consistent API response formatting
   - Add documentation explaining the new response formats

### 5.3 File Changes - Phase 3

1. **API Documentation**
   - Update Swagger documentation to indicate deprecated fields
   - Add migration guide for API consumers

### 5.4 File Changes - Phase 4

1. **`internal/api/api.go`**
   - Remove legacy field formats from `articleToPostmanSchema`
   - Update all references to use only camelCase

2. **`internal/api/api_test.go`**
   - Remove backward compatibility tests
   - Update all tests to expect only camelCase fields

## 6. Rollback Plan

If issues are discovered during any phase:

1. **Immediate Rollback**
   - Revert code changes to previous phase
   - Deploy emergency fix
   - Notify affected clients

2. **Root Cause Analysis**
   - Identify what caused the issue
   - Develop targeted fix
   - Add specific tests to prevent regression

3. **Retry Migration**
   - Update plan based on lessons learned
   - Proceed with more granular steps if needed

## 7. Success Criteria

The migration will be considered successful when:

1. All API responses use consistent camelCase field names
2. All tests pass with the new field naming convention
3. No client-reported issues due to field naming changes
4. API documentation is updated to reflect the new conventions
5. Codebase is more maintainable with consistent patterns 
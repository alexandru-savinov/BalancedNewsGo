# 🐛 LLM Reanalysis Timeout and Retry Logic Issues

## 📋 Issue Summary

**Priority:** High  
**Component:** LLM Integration  
**Affects:** Reanalysis functionality, User experience  

The JavaScript reanalysis workflow appears successful in the UI but fails silently due to LLM API timeout issues. Users see "Analysis complete" but receive cached results instead of fresh analysis.

## 🔍 Problem Description

### Current Behavior
1. User clicks "Request Reanalysis" → UI shows "Processing..."
2. SSE stream reports "Complete" in 0.4 seconds (suspiciously fast)
3. Bias analysis timestamp remains unchanged (cached results)
4. Fresh reanalysis attempts fail with timeout errors

### Root Cause
```json
{
  "error": "scoring with model openai/gpt-4.1-nano failed: Post \"https://openrouter.ai/api/v1/chat/completions\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)"
}
```

### Impact
- ❌ **Silent failures**: Users think reanalysis worked when it didn't
- ❌ **Stale data**: No fresh analysis performed, cached results returned
- ❌ **Poor UX**: Inconsistent behavior between "success" and actual failure
- ❌ **Production blocker**: Core functionality unreliable

## 🧪 E2E Test Results

**Test Environment:** Windows 11, Go server localhost:8080, Article ID 587

| Test Phase | Status | Details |
|------------|--------|---------|
| **Frontend Workflow** | ✅ PASS | Button trigger, SSE connection, UI completion |
| **API Endpoints** | ✅ PASS | POST `/api/llm/reanalyze/587` returns 200 |
| **LLM Integration** | ❌ FAIL | OpenRouter API timeout after 30s |
| **Error Handling** | ⚠️ PARTIAL | Silent failure, returns cached data |

## 🔧 Technical Analysis

### Existing Retry Mechanisms (Inconsistent)
- **Ensemble level**: 2 retries (ensemble.go:19)
- **API wrapper**: 3 retries (wrapper/client.go:37)
- **Service level**: Rate limit handling only
- **Timeout**: 30s (too aggressive for LLM APIs)

### Missing Components
- ❌ **Exponential backoff**: Immediate retries overwhelm failing services
- ❌ **Unified retry config**: Inconsistent retry counts across components
- ❌ **Error classification**: No distinction between retryable vs non-retryable errors
- ❌ **Proper timeout**: 30s insufficient for LLM API latency
- ❌ **Circuit breaker**: Continues retrying when service is down

## 🛠️ Proposed Solution

### 1. Unified Retry Configuration
```go
type RetryConfig struct {
    MaxAttempts       int           `default:"3"`
    BaseDelay         time.Duration `default:"2s"`
    MaxDelay          time.Duration `default:"30s"`
    Timeout           time.Duration `default:"90s"`
    BackoffMultiplier float64       `default:"2.0"`
}
```

### 2. Exponential Backoff Implementation
- **Retry delays**: 2s → 4s → 8s (capped at 30s)
- **Context cancellation**: Support for request cancellation
- **Error classification**: Retry only on retryable errors

### 3. Error Classification
**Retryable errors:**
- Timeout, connection reset, service unavailable
- HTTP 502, 503, 504
- LLM API timeout errors

**Non-retryable errors:**
- Authentication failures (401)
- Rate limits (429) - handled separately
- Invalid requests (400)

### 4. Files to Modify
- `internal/llm/config.go` - Add retry configuration
- `internal/llm/retry.go` - New retry executor
- `internal/llm/service_http.go` - Update service calls
- `internal/llm/ensemble.go` - Remove duplicate retry logic
- `internal/llm/llm.go` - Increase timeout to 90s

## 🎯 Implementation Plan

### Phase 1: Critical Fixes (20 minutes)
- [ ] Increase LLM timeout from 30s to 90s
- [ ] Add exponential backoff to existing retry loops
- [ ] Unify retry configuration across components

### Phase 2: Enhanced Logic (30 minutes)
- [ ] Implement RetryExecutor with error classification
- [ ] Update service layer to use new retry logic
- [ ] Remove duplicate retry loops in ensemble

### Phase 3: Configuration (15 minutes)
- [ ] Add retry configuration to config files
- [ ] Add monitoring/logging for retry attempts
- [ ] Test with various error scenarios

**Total Estimated Time:** ~65 minutes

## 🧪 Testing Plan

### Unit Tests
```bash
# Test retry logic with mocked failures
go test ./internal/llm -v -run TestRetryLogic

# Test timeout handling
go test ./internal/llm -v -run TestTimeoutHandling

# Test error classification
go test ./internal/llm -v -run TestErrorClassification
```

### Integration Tests
```bash
# Test full reanalysis workflow
go test ./internal/api -v -run TestReanalysisWorkflow

# Test with simulated network issues
go test ./internal/llm -v -run TestNetworkFailures
```

### E2E Tests

#### Manual Browser Testing
1. **Fresh Analysis Test**
   ```
   1. Navigate to http://localhost:8080/article/587
   2. Note current bias timestamp
   3. Click "Request Reanalysis"
   4. Wait for completion
   5. Verify timestamp updated (not cached)
   ```

2. **Error Handling Test**
   ```
   1. Disconnect internet during reanalysis
   2. Verify error message displayed in UI
   3. Verify previous score preserved
   4. Reconnect and retry - should succeed
   ```

3. **Timeout Test**
   ```
   1. Configure short timeout (5s) for testing
   2. Trigger reanalysis
   3. Verify timeout error handling
   4. Verify retry attempts logged
   ```

#### Automated E2E Testing
```powershell
# Run E2E test script
.\test-sse-monitor.ps1 -ArticleId 587 -TimeoutSeconds 120

# Expected results:
# - Fresh analysis with new timestamp
# - Proper error handling on failures
# - Retry attempts logged
# - No silent failures
```

### Performance Testing
```bash
# Test concurrent reanalysis requests
for i in {1..5}; do
  curl -X POST http://localhost:8080/api/llm/reanalyze/587 &
done
wait

# Verify:
# - No race conditions
# - Proper retry behavior under load
# - Resource cleanup after failures
```

### Error Scenario Testing
1. **Network Timeout**: Simulate slow network
2. **Service Unavailable**: Mock 503 responses
3. **Rate Limiting**: Test backup key fallback
4. **Invalid Responses**: Test malformed JSON handling
5. **Context Cancellation**: Test request cancellation

## ✅ Success Criteria

| Requirement | Test Method | Expected Result |
|-------------|-------------|-----------------|
| **No silent failures** | E2E reanalysis test | Error displayed in UI, no false success |
| **Fresh analysis** | Timestamp comparison | New timestamp on each reanalysis |
| **Timeout handling** | 90s timeout test | No premature failures |
| **Retry logic** | Network simulation | 3 retry attempts with exponential backoff |
| **Error preservation** | Failure test | Previous score preserved on error |
| **Performance** | Load testing | Stable under concurrent requests |

## 📊 Definition of Done

- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] E2E tests show fresh analysis (new timestamps)
- [ ] Error scenarios properly handled and displayed
- [ ] No silent failures in reanalysis workflow
- [ ] Performance stable under load
- [ ] Documentation updated with new retry behavior
- [ ] Monitoring/logging added for retry attempts

## 🔗 Related Issues

- Frontend reanalysis workflow (completed)
- SSE progress tracking (completed)
- Error handling improvements (this issue)

## 📝 Additional Notes

This issue blocks production deployment of the reanalysis feature. The frontend workflow is solid, but backend reliability must be ensured before release.

**Priority justification:** Core functionality failure with silent errors affecting user trust and data accuracy.

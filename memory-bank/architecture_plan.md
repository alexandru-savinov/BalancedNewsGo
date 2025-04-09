# Architectural Review & Refined Project Plan for NewsBalancer

---

## 1. Alignment with Objectives

- Aggregate diverse news sources.
- Analyze articles from **Left, Center, Right** perspectives using LLMs.
- Provide a **balanced, explainable ranking**.
- Prioritize **robust, deterministic bias detection**.
- Deliver a **modular MVP** with clear upgrade paths.

---

## 2. Architecture Overview

- **Backend:** Go (Gin, sqlx, cron, resty), SQLite.
- **LLM Integration:** API calls with prompt templates for each perspective.
- **Bias Detection:** Robust querying per perspective, outlier filtering, averaging.
- **Database:** Stores articles and multiple LLM scores.
- **API:** Supports filtering/sorting by political leaning.
- **Frontend:** Minimal UI with political indicators, filters, feedback.
- **Infrastructure:** Async jobs, caching, Dockerized deployment.

---

## 3. Detailed Step-by-Step Plan

### Phase 1: System Stabilization

- Warm up LLM APIs.
- Standardize prompts for **Left, Center, Right**.
- Fix known variability sources.
- Implement health checks and logging.

### Phase 2: Robust Multi-Perspective Bias Detection

- For each article and each perspective:
  - Use the **10-attempts, 5-valid-responses** method.
  - Discard highest/lowest, average middle 3.
  - Log all attempts and errors.
- Store scores separately in the database.
- Add deterministic tests for each perspective.
- Validate on a diverse dataset.

### Phase 3: Ranking Algorithm

- Define composite score combining:
  - Left, Center, Right bias scores.
  - Source diversity.
  - Recency, relevance.
  - User feedback (future).
- Document scoring logic.
- Prototype and validate.

### Phase 4: Backend & API

- Extend database schema for multi-perspective scores.
- Implement async batch processing of articles.
- Expose API endpoints with filtering/sorting.
- Add caching of LLM responses.

### Phase 5: Frontend Integration

- Display articles with political indicators.
- Add filters and sorting options.
- Collect user feedback on bias and ranking.

### Phase 6: Quality Control

- Enforce linting and code reviews.
- Maintain high test coverage.
- Use CI/CD for automated testing.
- Monitor system health and performance.

### Phase 7: Deployment & Feedback Loop

- Deploy MVP with Docker.
- Collect user feedback.
- Plan iterative improvements.

---

## 4. Risk Assessment & Mitigation

| **Risk** | **Impact** | **Likelihood** | **Mitigation** |
|--------------------------|----------|--------------|------------------------|
| LLM variability          | High     | High         | Robust querying, prompt tuning |
| API costs                | High     | High         | Caching, batching, async jobs |
| Prompt engineering       | High     | Medium       | Iterative refinement, validation |
| Performance bottlenecks  | Medium   | Medium       | Async processing, profiling |
| Data diversity           | Medium   | Medium       | Expand/categorize sources |
| Integration bugs         | Medium   | Medium       | Incremental integration, tests |
| User acceptance          | Medium   | Medium       | Feedback loop, UI improvements |

---

## 5. Quality Control Measures

- **Robust querying** to stabilize LLM outputs.
- **Deterministic tests** for bias detection.
- **Code linting** and reviews.
- **Automated CI/CD** pipelines.
- **Monitoring** and logging.
- **Documentation** of prompts, scoring, API.

---

## 6. Contingency Plans

- **LLM failures:** Retry, fallback to cached results.
- **Cost overruns:** Optimize prompts, reduce frequency.
- **Bias detection issues:** Revert to previous heuristics, increase samples.
- **Data source failures:** Use backups, manual updates.
- **Test failures:** Isolate, prioritize fixes.

---

## 7. Future Enhancements

- Confidence metrics and explanations.
- Advanced UI visualizations.
- Multiple LLM providers.
- User accounts and personalization.
- Topic clustering and analytics.

---

## Summary

This refined plan fully aligns with the project’s requirements and roadmap, emphasizing **robust, multi-perspective bias detection** as the foundation for a balanced, explainable ranking system. It incorporates risk mitigation, quality control, and extensibility to ensure a successful MVP and a clear path for future growth.
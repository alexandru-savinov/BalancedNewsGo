<!-- Metadata -->
Last updated: April 9, 2025
Author: Roo AI Assistant

# Changelog
- **2025-04-09:** Added comprehensive continuous bias scoring & ensemble redesign, metadata, and changelog.
- **Earlier:** Initial architecture plan with phased implementation, risk assessment, and future enhancements.

---

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

## 8. Continuous Bias Scoring & Ensemble Architecture (2025 Redesign)

### Overview

This redesign transitions from discrete bias labels to a **continuous, fine-grained bias score** framework, integrated with a multi-model, multi-prompt ensemble, advanced prompt engineering, diverse sampling, comprehensive storage, and systematic outlier analysis.

---

### Continuous Bias Scoring

- **Range:** -1.0 (strongly left) to 1.0 (strongly right), in 0.1 increments.
- **Interpretation:**  
  - Strongly Left: -1.0 to -0.6  
  - Moderately Left: -0.5 to -0.2  
  - Center: -0.1 to 0.1  
  - Moderately Right: 0.2 to 0.5  
  - Strongly Right: 0.6 to 1.0
- **Example:**  
  "The billionaire class exploits workers" → -0.8 (strongly left)  
  "Government overreach burdens taxpayers" → 0.7 (strongly right)  
  "Experts debate the policy's impact" → 0.0 (neutral)

---

### Prompt Engineering

- Explicitly request a **numerical score** and **detailed explanation**.
- Provide **few-shot examples** covering the full spectrum.
- Use **multiple prompt variants** focusing on different cues.
- Discourage neutral defaults unless truly balanced.

---

### Multi-Model, Multi-Prompt Ensemble

- **Models:** GPT-3.5, GPT-4, Claude, fine-tuned classifiers.
- **Prompts:** Varied phrasings, examples, focus areas.
- **Outputs:** Bias score, explanation, confidence.
- **Aggregation:** Mean, weighted mean, variance.
- **Uncertainty:** High variance or low confidence flags ambiguity.

---

### Diversity-Enforcing Data Pipeline

- **Sources:** Left, right, center, fringe, international.
- **Topics:** Politics, economy, culture, social issues.
- **Time:** Recent and historical.
- **Deduplication:** Remove redundant content.
- **Partisan cues:** Preserve or highlight.
- **Balanced sampling:** Enforce representation.

---

### Processing Pipeline

1. Fetch diverse, deduplicated articles.
2. Generate multiple prompt variants.
3. Query multiple models.
4. Collect scores, explanations, confidences.
5. Aggregate results, compute variance.
6. Threshold scores into categories.
7. Quantify uncertainty.
8. Store all data.
9. Detect and flag outliers.
10. Visualize and report.

---

### Comprehensive Storage & Outlier Analysis

- **Store:**  
  - Article metadata and content  
  - Prompts and models used  
  - Raw LLM responses  
  - Parsed scores, explanations, confidences  
  - Aggregated results, variance  
  - Processing timestamps, system config
- **Detect outliers:**  
  - Extreme scores deviating from ensemble mean  
  - High disagreement across ensemble  
  - Sudden temporal shifts  
  - Source-specific anomalies
- **Visualize:**  
  - Scatter plots, heatmaps, time series  
  - Highlight flagged cases
- **Benefits:**  
  - Transparency, bias diagnosis, quality control, continuous improvement

---

### Visualization & User Experience

- Bias distribution histograms and density plots.
- Temporal trends with anomaly markers.
- Source/topic breakdowns.
- Interactive dashboards with outlier highlights.

---

### Feedback Loop

- Validate against labeled datasets.
- Track precision, recall, bias-specific metrics.
- Refine prompts, models, weights.
- Use flagged outliers to guide improvements.

---

### Summary

This integrated redesign enables **granular, nuanced, transparent, and trustworthy** political bias detection, supporting robust analytics, continuous validation, and iterative refinement.
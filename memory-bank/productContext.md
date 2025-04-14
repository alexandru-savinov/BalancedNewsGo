<!-- Metadata -->
Last updated: April 10, 2025  
Author: Roo AI Assistant

# Changelog
- **2025-04-10:** Inserted project brief from `projectbrief.md` at the top. Part of memory-bank restructuring.
- **2025-04-10:** Documented article page UI/UX redesign with interactive bias visualization, image optimization, and layout improvements.
- **2025-04-09:** Added metadata and changelog sections. Planned expansion with personas, user stories, KPIs.
- **Earlier:** Initial product context with user needs, problem statement, and value proposition.

---

# Project Brief: BalancedNewsGo

BalancedNewsGo is a news aggregation platform designed to provide users with a balanced, multi-perspective view of current events. It leverages Large Language Models (LLMs) to analyze news articles from diverse sources, summarize differing viewpoints, and highlight potential biases. The goal is to help users break out of echo chambers and develop a more nuanced understanding of the news.

---

# Product Context

## User Needs
- Access diverse perspectives on news topics
- Avoid echo chambers and filter bubbles
- Quickly understand differing viewpoints
- Identify potential bias in reporting

## Problem Statement
Most news aggregators reinforce existing biases by surfacing similar viewpoints. Users struggle to get a balanced, multi-perspective understanding of current events, leading to polarization and misinformation.

## Value Proposition
BalancedNewsGo aggregates news from multiple sources, uses LLMs to analyze and summarize diverse perspectives, and highlights potential biases. This empowers users to make more informed opinions based on a comprehensive view of the news landscape.

---

## Developer & Tester Debugging Needs (April 2025)

To ensure model reliability and accelerate iteration, the UI now prioritizes **transparent, info-rich debugging features**:

- **Expose raw data:** Article IDs, sources, fetch/scoring timestamps, fallback status, raw composite scores, average confidence, model count.
- **Visualize model behavior:** Bias slider with color-coded zones, model disagreement highlights, detailed tooltips including parse status.
- **Default expanded debug info:** Raw model outputs, parse success/failure, aggregation stats.
- **Feedback tagging:** Options to report parse errors, model disagreement, low confidence, or fallback usage, improving data collection for model refinement.
- **Full detail views:** Article detail modal/page with full text, raw responses, parse status, timestamps, download JSON, retry parse/re-score.
- **Transparency on fallback triggers, API responses, parse failures, aggregation methods, and timestamps.**
- **Color cues and inline indicators** for quick status assessment.
- **Accessibility:** ARIA labels, keyboard navigation, high contrast, semantic HTML to support all team members.
- **Minimal JS/SCSS:** Ensures maintainability and reduces debugging complexity.

These features enable developers and testers to **quickly identify issues, understand model outputs, and iterate on prompts, parsing, and aggregation logic** with minimal friction.

---

## [2025-04-13 17:22:55] - Added comprehensive Postman-based rescoring workflow test plan

A detailed Postman testing strategy for the article manual scoring (rescoring) workflow has been developed and saved to `memory-bank/postman_rescoring_test_plan.md`. This plan covers test case design, environment setup, data preparation, request sequencing, response validation, edge case handling, expected outcomes, and automation. It enables robust, repeatable, and automated validation of the rescoring workflow for the Go backend API.
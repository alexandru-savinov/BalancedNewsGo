# Frontend Rewrite Plan: React & TypeScript (Simplified & API-First)

**Date:** 2025-04-12

## 1. Introduction

*   **Goal:** Rewrite the existing htmx-based frontend for BalancedNewsGo using React and TypeScript.
*   **Focus:** Deliver core value faster by prioritizing modularity, robust state management, performance, UI/UX consistency (essential debugging features), and core user journeys. Authentication and non-essential features are deferred initially.

## 2. Proposed Architecture

*   **Technology Stack:**
    *   UI Library: React 18+
    *   Language: TypeScript
    *   Build Tool/Dev Server: Vite
    *   Routing: React Router v6
    *   State Management: Zustand
    *   Styling: CSS Modules (Recommended for potentially faster initial setup)
    *   API Client: Axios or Fetch API wrapper
*   **Structure:** Single Page Application (SPA).
*   **Diagram (High-Level Flow):**
    ```mermaid
    graph TD
        A[Browser Request] --> B(Vite Dev Server / Static Files);
        B --> C{React App (main.tsx)};
        C --> D[React Router];
        D --> E{Layout Component};
        E --> F[Page Component (e.g., HomePage, ArticlePage)];
        F --> G[Feature Components (e.g., ArticleList, ArticleCard)];
        G --> H[Reusable UI Components (e.g., Button, BiasSlider)];
        G --> I{Zustand Store};
        G --> J[API Service Layer];
        J --> K(Backend API);
        I -.-> G;
        K -.-> J;
    ```

## 3. State Management

*   **Recommendation:** Zustand
*   **Justification:** Simplicity (less boilerplate), modularity (slices/stores), performance, and sufficiency for the project's core needs.

## 4. Component Structure

*   **Approach:** Feature-based directories combined with Atomic Design principles for reusable components.
*   **Proposed Directory Structure:**
    ```
    src/
    ├── App.tsx            # Root component, router setup
    ├── main.tsx           # Application entry point
    ├── index.css          # Global styles
    ├── assets/            # Static assets (images, fonts)
    ├── components/        # Reusable UI (Atoms, Molecules, Organisms)
    │   ├── common/        # Button.tsx, Input.tsx, Loader.tsx, Modal.tsx, Tooltip.tsx, BiasSlider.tsx
    │   └── layout/        # Header.tsx, MainLayout.tsx
    ├── features/          # Feature-specific components, hooks, and logic
    │   ├── articles/      # ArticleList.tsx, ArticleCard.tsx, ArticleDetailView.tsx, FeedbackForm.tsx, useArticleData.ts
    │   └── dashboard/     # DashboardPage.tsx, FilterControls.tsx (basic)
    ├── hooks/             # Global custom hooks (e.g., useApi.ts)
    ├── pages/             # Top-level route components (HomePage.tsx, NotFoundPage.tsx)
    ├── services/          # API interaction logic
    │   ├── apiClient.ts   # Axios/Fetch instance setup
    │   ├── articleService.ts
    │   ├── feedbackService.ts
    │   └── sseService.ts    # For score progress
    ├── store/             # Zustand store configuration
    │   ├── index.ts       # Main store setup
    │   └── slices/        # articleSlice.ts, uiSlice.ts
    ├── styles/            # Global styles, theme variables (using CSS Modules)
    └── types/             # TypeScript interfaces and types (api.d.ts, app.d.ts)
    ```

## 5. Implementation Plan (Simplified Critical Journeys)

*   **Backend Consideration (API-First):**
    *   **Consolidate Article Data:** *Before starting Phase 2*, review and potentially modify the backend API endpoint `/api/articles/:id` to return main article data *along with* essential bias/ensemble details (composite score, confidence, model scores for slider). This simplifies frontend logic and aligns with API-first. Requires backend changes.
    *   **Authentication Deferral:** Authentication is deferred from the initial implementation. Backend auth endpoints (`/login`, `/logout`, `/user`) will need to be implemented *before* frontend auth work begins in a later phase.
*   **Phase 1: Setup & Core Layout**
    *   Initialize project (Vite, React-TS).
    *   Install dependencies (React Router, Zustand, Axios/Fetch, CSS Modules setup).
    *   Configure TypeScript.
    *   Set up basic project structure, layout components, router, Zustand store, API client.
*   **Phase 2: Dashboard & Basic Article Display**
    *   Implement `articleService.ts` for `/api/articles`.
    *   Create `articleSlice.ts` in Zustand.
    *   Develop `DashboardPage.tsx`, `ArticleList.tsx`, `ArticleCard.tsx`.
    *   Display essential article info (Title, Source, Date, Composite Score).
    *   Implement basic filtering controls if straightforward.
    *   *(Defer "Refresh Feeds" button initially)*.
*   **Phase 3: Detailed View & Essential Debugging UI**
    *   Enhance `ArticleCard.tsx` or create `ArticleDetailView.tsx`.
    *   Fetch detailed data via the (potentially consolidated) `/api/articles/:id` endpoint. Store in Zustand.
    *   Implement `BiasSlider.tsx` (core visual representation).
    *   Implement basic `Tooltip.tsx` for scores.
    *   Display essential debugging info: Composite Score, Confidence, Source, Timestamps, Model Scores. *(Defer raw model outputs, extensive stats)*.
*   **Phase 4: Scoring & Feedback (Core Entity Creation)**
    *   Implement "Score Article" button calling `/api/llm/reanalyze/:id`.
    *   Implement SSE handling (`sseService.ts`) for `/api/llm/score-progress/:id` and display progress.
    *   Implement `FeedbackForm.tsx` and `feedbackService.ts` to POST to `/api/feedback`.
*   **Phase 5: Authentication** *(Deferred)*

## 6. UI/UX Adherence Strategy

*   Prioritize replicating core layout, article display, essential metadata, bias slider, and feedback mechanism based on `web/index.html` and Memory Bank descriptions.
*   Defer non-essential visual elements or complex debug displays initially.

## 7. Performance Optimization Strategies

*   Leverage Vite defaults.
*   Implement route-based code splitting (`React.lazy`).
*   Use Zustand selectors effectively.
*   Apply basic memoization (`React.memo`) where obvious.
*   *(Defer complex profiling/optimization)*.

## 8. Cypress Testing (Post-Implementation)

*   Focus E2E tests on implemented core journeys: Dashboard display/filtering, viewing article details (essential debug info), triggering scoring/progress, submitting feedback.
*   *(Authentication tests deferred)*.
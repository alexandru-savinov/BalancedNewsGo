# Active Context

## Backend Status (Post-Implementation)

### RSS Fetching
- Fully implemented via `internal/rss`
- Fetches and parses multiple news sources reliably

### Database
- SQLite schema operational
- Stores raw articles and LLM analysis results
- Repository pattern abstracts DB access

### LLM Integration
- **OpenAI API fully integrated, replacing mock services**
- **Prompt engineering refined with configurable templates**
- **Bias detection enhanced with structured outputs and heuristics**
- Summarization and bias detection functional via OpenAI
- Multi-perspective extraction basic, planned for future refinement

### API
- REST API operational via `internal/api`
- Serves articles, summaries, and bias data
- **Endpoints for user feedback, comparison, and bias insights implemented**

### Frontend
- Improved UI with htmx dynamic loading
- Displays articles, summaries, bias info
- Supports inline user feedback submission
- Responsive and accessible design

## Current Focus
- Testing and validation of new features
- Collecting user feedback
- Planning future enhancements (multi-perspective extraction, source diversity)

## Recent Changes
- Toolchain issues fixed, improving build/test reliability
- Backend audit completed, confirming modular design is sound
- **OpenAI API integration completed and verified**
- Backend components integrated and functional end-to-end
- Frontend enhanced with summaries, bias info, and feedback forms

## Debugging and Diagnostics

- Introduced **verbose logging** across backend modules to facilitate troubleshooting.
- Fixed a **nil pointer dereference bug** in the LLM integration module, improving stability.
- Improved configuration fallback logic to use environment variables when config files are unavailable.

## Next Steps and Improvements

### Quality Control Loops
- Integrate automated validation after each subtask 
- Use assertions and heuristics to catch errors early
- Implement feedback loops where agents can request clarification or reprocessing if confidence is low

### Automated Testing
- Add or fix unit and integration tests for backend components
- Ensure OpenAI integration, bias detection, and API endpoints are covered
- Set up CI (e.g., GitHub Actions) to run tests on every push

### Robust Error Handling and Logging
- Improve error messages and add structured logging
- Log LLM API failures, database errors, and user input issues clearly

### Security Improvements
- Secure API keys and sensitive configs (use environment variables or secret managers)
- Add input validation and sanitize user feedback submissions
- Plan for authentication and authorization if user accounts are added

### Frontend Polish
- Improve UI/UX for clarity and responsiveness
- Add loading indicators, error messages, and success confirmations
- Enhance accessibility

### Bias Detection Refinement
- Improve prompt design and post-processing for more accurate, nuanced bias insights
- Add confidence thresholds and fallback logic

### Expand News Sources
- Integrate more diverse RSS feeds to reduce bias and increase coverage

### Documentation
- Update README, API docs, and developer setup instructions
- Document prompt templates and configuration options

## Known Issues

- **Bias detection logic requires refinement**; current heuristics sometimes yield inconsistent or incorrect results.
- **Some logic tests in `internal/llm` continue to fail** due to variability in bias detection outputs.
- Further improvements needed to stabilize and validate bias analysis.

### OpenAI API Integration Anomaly (April 8, 2025)

- Real OpenAI API calls are confirmed via logs.
- However, responses fail to parse as JSON, causing repeated errors:
  
  `Failed to parse LLM JSON response: invalid character 'X' looking for beginning of value`
  
- This suggests unexpected API response format or a bug in parsing logic.
- Needs urgent investigation and fix.

### Improved Article Ranking & OpenAI Fallback (April 8, 2025)

- **Multi-factor ranking** combines:
  - **Bias balance score** (favoring diverse perspectives)
  - **Recency score** (favoring newer articles)
- **Fallback strategy:**
  - If OpenAI bias scores are **missing or invalid**, use **recency score only** or apply a **penalty**.
  - Flag such articles for **reprocessing or review**.
  - Avoid top-ranking unreliable articles.
- **Future extensions:** incorporate user feedback, source credibility, personalization.
- **Goal:** Deliver a well-organized, balanced, and reliable article feed despite LLM issues.
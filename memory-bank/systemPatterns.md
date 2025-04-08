# System Patterns

## Architecture Overview
- Modular Go backend with clear separation of concerns:
  - **RSS Module (`internal/rss`)**: Fetches and parses feeds from multiple sources
  - **Database Module (`internal/db`)**: Handles SQLite persistence for articles and LLM outputs
  - **LLM Module (`internal/llm`)**: Interfaces with OpenAI API or local models, abstracts provider differences
  - **API Module (`internal/api`)**: Exposes REST endpoints for frontend consumption
- SQLite database for lightweight, local storage
- Integration with LLMs (OpenAI API or local models) via adapter pattern
- REST API layer serving JSON
- Frontend using HTML, JavaScript, and htmx for interactivity

## Backend Data Flow
1. **Fetch**: RSS module retrieves articles from diverse news sources
2. **Store Raw**: Raw articles saved in SQLite via database module
3. **Analyze**: LLM module processes articles to generate summaries, extract perspectives, and identify potential bias
4. **Store Analysis**: Summaries and metadata linked to original articles in database
5. **Serve**: API module exposes endpoints to retrieve articles, summaries, perspectives
6. **Display**: Frontend fetches and dynamically displays balanced news content

## Key Design Patterns
- **Repository Pattern**: Abstracts database access
- **Adapter Pattern**: Supports multiple LLM providers seamlessly
- **RESTful API**: Clean separation between backend and frontend
- **Progressive Enhancement**: htmx augments static HTML with dynamic updates

## Component Relationships
- RSS module feeds data into the database
- LLM module reads raw articles from database, writes back analysis
- API module queries database for both raw and analyzed data
- Frontend interacts solely with API module

## Notes from Backend Audit
- Modular design confirmed effective
- Adapter pattern for LLMs simplifies provider switching
- Data flow is clear and maintainable
- Opportunities:
  - Improve LLM prompt engineering for richer multi-perspective output
  - Expand API endpoints (e.g., user feedback, comparison views)
  - Enhance bias detection logic in LLM processing
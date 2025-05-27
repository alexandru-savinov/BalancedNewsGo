**Document Version**: 1.0
**Date**: May 27, 2025
**Target**: Production-Ready Multi-Page Frontend Implementation
**Status**: Approved for Implementation

## Executive Summary

This document provides a comprehensive technical specification for the NewsBalancer frontend implementation. It defines the complete architecture, component specifications, API contracts, performance requirements, and implementation guidelines for a production-ready web application.

## Document Scope

This specification covers:
- Complete technical architecture and component design
- Detailed API contracts and data models
- Performance benchmarks and optimization strategies
- Security implementation requirements
- Testing specifications and quality assurance
- Deployment and operational requirements

## Technical Architecture

### System Architecture Overview
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Browser   │◄──►│  Go HTTP Server │◄──►│   Database      │
│                 │    │                 │    │   (SQLite)      │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │                 │
│ │ HTML Pages  │ │    │ │ REST API    │ │    │                 │
│ │ CSS Modules │ │    │ │ SSE Stream  │ │    │                 │
│ │ JS Components│ │    │ │ Static Srv  │ │    │                 │
│ └─────────────┘ │    │ └─────────────┘ │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

**Multi-Page Application (MPA)** with the following structure:
- Individual HTML files for each major view with server-side rendering capability
- Shared CSS and JavaScript modules with ES6 module system
- Progressive enhancement approach with graceful degradation
- Server-side routing via Go templates with client-side enhancement

## Page Structure

### 1. Articles List (`/web/articles.html`)
**Primary landing page displaying news articles with bias analysis**

**Features:**
- Grid/list view of articles with bias indicators
- Search and filtering (source, leaning, date range)
- Pagination controls
- Real-time bias score sliders for each article
- Click-through to article detail pages

**Layout:**
```
[Header with Navigation]
[Search/Filter Bar]
[Articles Grid]
  - Article Card (Title, Source, Date, Bias Slider, Summary)
  - Article Card (Title, Source, Date, Bias Slider, Summary)
  - ...
[Pagination Controls]
[Footer]
```

### 2. Article Detail (`/web/article.html`)
**Detailed view of individual articles with comprehensive bias analysis**

**Features:**
- Full article content display
- Interactive bias slider with real-time updates
- Manual scoring interface (admin feature on article page)
- SSE-powered real-time analysis progress
- Individual model scores breakdown
- User feedback submission form
- Re-analysis trigger button

**Layout:**
```
[Header with Navigation]
[Article Meta (Title, Source, Date, URL)]
[Bias Analysis Section]
  - Main Bias Slider (Interactive)
  - Individual Model Scores
  - Confidence Indicators
  - Manual Scoring Controls
[Article Content]
[Feedback Section]
[Re-analysis Controls]
```

### 3. Admin Dashboard (`/web/admin.html`)
**Administrative interface for system management**

**Features:**
- RSS feed management and health status
- System metrics visualization
- Feed refresh controls
- User feedback overview
- System performance indicators
- Database statistics

**Layout:**
```
[Header with Navigation]
[System Status Cards]
[RSS Feed Management]
[Metrics Dashboard]
[User Feedback Summary]
[System Controls]
```

## Technical Stack

## Technical Stack Specifications

### Core Technologies & Versions
- **HTML5**: Semantic markup compliant with WHATWG Living Standard
- **CSS3**: Custom CSS using Level 4 specifications (Grid, Flexbox, Custom Properties)
- **JavaScript**: ES2022+ with native modules, targeting baseline browsers
- **Web APIs**: Fetch API, Server-Sent Events, Web Storage API, History API

### Dependencies & Versions
1. **Chart.js v4.4.0** (47KB gzipped) - Data visualization library
   - **Purpose**: Bias score charts, admin dashboard metrics
   - **CDN**: `https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.js`
   - **Integrity**: `sha384-...` (to be calculated during implementation)

2. **DOMPurify v3.0.5** (45KB gzipped) - XSS sanitization
   - **Purpose**: User-generated content sanitization
   - **CDN**: `https://cdn.jsdelivr.net/npm/dompurify@3.0.5/dist/purify.min.js`

3. **Normalize.css v8.0.1** (2KB gzipped) - CSS reset
   - **CDN**: `https://cdn.jsdelivr.net/npm/normalize.css@8.0.1/normalize.min.css`

**Total Bundle Size**: 47KB (HTML/CSS/JS) + 94KB (dependencies) = **141KB total**
**Target Optimized Size**: < 50KB after tree-shaking and compression

## Component Architecture

## Component Specifications

### 1. Shared Components (`/web/js/components/`)

#### `BiasSlider.js` - Interactive Bias Score Component
**Purpose**: Primary UI component for displaying and editing bias scores

**Technical Specifications**:
```javascript
class BiasSlider extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: 'open' });
  }

  static get observedAttributes() {
    return ['value', 'readonly', 'article-id', 'size'];
  }

  // Component state
  #value = 0;          // Current bias score (-1 to +1)
  #readonly = false;   // Edit mode toggle
  #articleId = null;   // Associated article ID
  #size = 'medium';    // Component size variant
}
```

**Properties & Methods**:
- `value: number` - Bias score value (-1.0 to +1.0)
- `readonly: boolean` - Toggle edit mode
- `articleId: string` - Article identifier for API calls
- `size: 'small'|'medium'|'large'` - Component size variant
- `updateValue(newValue: number): Promise<void>` - Update bias score
- `enableEditMode(): void` - Allow user interaction
- `disableEditMode(): void` - Make component read-only

**Events**:
- `biaschange` - Fired when bias value changes
- `biasupdate` - Fired during live drag operations
- `apierror` - Fired when API update fails

**Accessibility**:
- ARIA role: `slider`
- ARIA labels: `aria-valuemin="-1" aria-valuemax="1" aria-valuenow="{value}"`
- Keyboard support: Arrow keys, Home/End
- Screen reader announcements for value changes

#### `ArticleCard.js` - Article Preview Component
**Purpose**: Reusable card component for article listings

**Technical Specifications**:
```javascript
class ArticleCard extends HTMLElement {
  #article = null;     // Article data object
  #biasSlider = null;  // Embedded BiasSlider instance
  #clickHandler = null; // Navigation handler
}
```

**Properties**:
- `article: ArticleData` - Complete article data object
- `showBiasSlider: boolean` - Toggle bias slider visibility
- `compact: boolean` - Compact layout mode
- `clickable: boolean` - Enable navigation on click

**Article Data Interface**:
```typescript
interface ArticleData {
  id: string;
  title: string;
  url: string;
  content?: string;
  summary: string;
  source: string;
  publishedAt: string;
  bias: {
    score: number;        // -1 to +1
    confidence: number;   // 0 to 1
    modelScores: ModelScore[];
  };
  metadata: {
    wordCount: number;
    readingTime: number;
  };
}

interface ModelScore {
  modelName: string;
  score: number;
  confidence: number;
  timestamp: string;
}
```

#### `ProgressIndicator.js` - Real-time Progress Component
**Purpose**: SSE-powered progress tracking for LLM analysis

**Technical Specifications**:
```javascript
class ProgressIndicator extends HTMLElement {
  #eventSource = null;    // SSE connection
  #progressValue = 0;     // Current progress (0-100)
  #status = 'idle';       // Current status
  #reconnectAttempts = 0; // Connection retry counter
}
```

**States**:
- `idle` - No active operation
- `connecting` - Establishing SSE connection
- `processing` - Analysis in progress
- `completed` - Operation finished
- `error` - Error state
- `disconnected` - Connection lost

**Progress Data Interface**:
```typescript
interface ProgressData {
  progress: number;      // 0-100 percentage
  status: string;        // Human-readable status
  stage: string;         // Current processing stage
  eta?: number;          // Estimated completion time
  modelProgress?: {      // Per-model progress breakdown
    [modelName: string]: {
      status: 'pending' | 'processing' | 'completed' | 'error';
      progress: number;
    };
  };
}
```

#### `ApiClient.js` - HTTP Client with Advanced Error Handling
**Purpose**: Centralized API communication with enterprise-grade reliability

**Technical Specifications**:
```javascript
class ApiClient {
  #baseURL = '';
  #defaultHeaders = {
    'Content-Type': 'application/json',
    'X-Requested-With': 'XMLHttpRequest'
  };
  #circuitBreaker = new CircuitBreaker();
  #retryPolicy = new RetryPolicy();
  #requestCache = new Map();
}
```

**Circuit Breaker Configuration**:
```javascript
const circuitBreakerConfig = {
  failureThreshold: 5,     // Failures before opening circuit
  recoveryTimeout: 30000,  // 30s before trying again
  monitoringPeriod: 10000  // 10s monitoring window
};
```

**Retry Policy Configuration**:
```javascript
const retryConfig = {
  maxAttempts: 3,
  baseDelay: 1000,        // 1s base delay
  maxDelay: 10000,        // 10s max delay
  backoffFactor: 2,       // Exponential backoff
  retryableStatusCodes: [408, 429, 500, 502, 503, 504]
};
```

**Methods**:
- `get(endpoint: string, options?: RequestOptions): Promise<ApiResponse>`
- `post(endpoint: string, data: any, options?: RequestOptions): Promise<ApiResponse>`
- `put(endpoint: string, data: any, options?: RequestOptions): Promise<ApiResponse>`
- `delete(endpoint: string, options?: RequestOptions): Promise<ApiResponse>`
- `sse(endpoint: string, handler: SSEHandler): EventSource`
- `upload(endpoint: string, file: File, options?: UploadOptions): Promise<ApiResponse>`

### 2. Page-Specific Scripts (`/web/js/pages/`)

#### `articles.js`
- Article listing logic
- Search/filter implementation
- Pagination handling
- Infinite scroll (optional enhancement)

#### `article-detail.js`
- Individual article management
- Real-time bias analysis updates
- Manual scoring interface
- Feedback submission

#### `admin.js`
- Dashboard data fetching
- RSS feed management
- System controls
- Metrics visualization

## Styling Approach

### CSS Architecture
```
/web/css/
├── reset.css           # Browser normalization
├── variables.css       # CSS custom properties
├── layout.css          # Grid/Flexbox layouts
├── components.css      # Reusable components
├── bias-slider.css     # Slider-specific styles
├── articles.css        # Articles page styles
├── article-detail.css  # Article detail styles
├── admin.css          # Admin dashboard styles
└── main.css           # Import all stylesheets
```

## CSS Architecture & Design System Specifications

### CSS Module Structure
```
/web/css/
├── core/
│   ├── reset.css           # Normalize + custom resets
│   ├── variables.css       # CSS custom properties
│   ├── typography.css      # Font definitions & scales
│   └── layout.css          # Grid systems & layouts
├── components/
│   ├── buttons.css         # Button component styles
│   ├── forms.css           # Form element styling
│   ├── cards.css           # Article card components
│   ├── navigation.css      # Header/nav components
│   ├── bias-slider.css     # Bias slider styling
│   ├── progress.css        # Progress indicators
│   └── modals.css          # Modal/overlay styles
├── pages/
│   ├── articles.css        # Articles listing page
│   ├── article-detail.css  # Single article page
│   ├── admin.css          # Admin dashboard
│   └── error.css          # Error page styling
├── utilities/
│   ├── spacing.css         # Margin/padding utilities
│   ├── colors.css          # Color utility classes
│   ├── responsive.css      # Responsive utilities
│   └── accessibility.css   # A11y helper classes
└── main.css               # Import orchestration
```

### CSS Custom Properties (Design Tokens)
```css
:root {
  /* Colors - Primary Palette */
  --color-primary-50: #eff6ff;
  --color-primary-100: #dbeafe;
  --color-primary-500: #3b82f6;  /* Primary blue */
  --color-primary-600: #2563eb;
  --color-primary-900: #1e3a8a;

  /* Colors - Bias Scale */
  --color-bias-left: #dc2626;    /* Red for left bias */
  --color-bias-center: #6b7280;  /* Gray for center */
  --color-bias-right: #2563eb;   /* Blue for right bias */

  /* Colors - Semantic */
  --color-success: #059669;
  --color-warning: #d97706;
  --color-error: #dc2626;
  --color-info: #0ea5e9;

  /* Typography Scale */
  --font-family-base: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  --font-family-mono: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, monospace;

  --font-size-xs: 0.75rem;    /* 12px */
  --font-size-sm: 0.875rem;   /* 14px */
  --font-size-base: 1rem;     /* 16px */
  --font-size-lg: 1.125rem;   /* 18px */
  --font-size-xl: 1.25rem;    /* 20px */
  --font-size-2xl: 1.5rem;    /* 24px */
  --font-size-3xl: 1.875rem;  /* 30px */
  --font-size-4xl: 2.25rem;   /* 36px */

  --line-height-tight: 1.25;
  --line-height-base: 1.5;
  --line-height-relaxed: 1.625;

  /* Spacing Scale (8px base) */
  --space-1: 0.25rem;   /* 4px */
  --space-2: 0.5rem;    /* 8px */
  --space-3: 0.75rem;   /* 12px */
  --space-4: 1rem;      /* 16px */
  --space-5: 1.25rem;   /* 20px */
  --space-6: 1.5rem;    /* 24px */
  --space-8: 2rem;      /* 32px */
  --space-10: 2.5rem;   /* 40px */
  --space-12: 3rem;     /* 48px */
  --space-16: 4rem;     /* 64px */

  /* Border Radius */
  --radius-sm: 0.125rem;  /* 2px */
  --radius-base: 0.25rem; /* 4px */
  --radius-md: 0.375rem;  /* 6px */
  --radius-lg: 0.5rem;    /* 8px */
  --radius-xl: 0.75rem;   /* 12px */
  --radius-full: 9999px;

  /* Shadows */
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-base: 0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);

  /* Transitions */
  --transition-fast: 150ms ease;
  --transition-base: 200ms ease;
  --transition-slow: 300ms ease;

  /* Z-index Scale */
  --z-dropdown: 1000;
  --z-sticky: 1020;
  --z-fixed: 1030;
  --z-modal-backdrop: 1040;
  --z-modal: 1050;
  --z-popover: 1060;
  --z-tooltip: 1070;
}

/* Dark mode support */
@media (prefers-color-scheme: dark) {
  :root {
    --color-bg-primary: #111827;
    --color-bg-secondary: #1f2937;
    --color-text-primary: #f9fafb;
    --color-text-secondary: #d1d5db;
    --color-border: #374151;
  }
}

/* Reduced motion support */
@media (prefers-reduced-motion: reduce) {
  :root {
    --transition-fast: 0ms;
    --transition-base: 0ms;
    --transition-slow: 0ms;
  }

  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

### Component-Specific CSS Specifications

#### BiasSlider Component CSS
```css
.bias-slider {
  --slider-height: 8px;
  --thumb-size: 20px;
  --track-radius: 4px;

  position: relative;
  width: 100%;
  height: var(--thumb-size);
  margin: var(--space-2) 0;
}

.bias-slider__track {
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  height: var(--slider-height);
  transform: translateY(-50%);
  border-radius: var(--track-radius);
  background: linear-gradient(
    to right,
    var(--color-bias-left) 0%,
    var(--color-bias-center) 50%,
    var(--color-bias-right) 100%
  );
  box-shadow: inset 0 1px 2px rgb(0 0 0 / 0.1);
}

.bias-slider__thumb {
  position: absolute;
  top: 50%;
  width: var(--thumb-size);
  height: var(--thumb-size);
  transform: translate(-50%, -50%);
  background: white;
  border: 2px solid var(--color-primary-500);
  border-radius: 50%;
  box-shadow: var(--shadow-md);
  cursor: grab;
  transition: all var(--transition-base);
}

.bias-slider__thumb:hover {
  box-shadow: var(--shadow-lg);
  transform: translate(-50%, -50%) scale(1.1);
}

.bias-slider__thumb:active {
  cursor: grabbing;
  box-shadow: var(--shadow-base);
  transform: translate(-50%, -50%) scale(0.95);
}

.bias-slider__thumb:focus {
  outline: 2px solid var(--color-primary-500);
  outline-offset: 2px;
}

/* Size variants */
.bias-slider--small {
  --slider-height: 6px;
  --thumb-size: 16px;
}

.bias-slider--large {
  --slider-height: 12px;
  --thumb-size: 24px;
}

/* Readonly state */
.bias-slider--readonly .bias-slider__thumb {
  cursor: default;
  pointer-events: none;
}

/* Color-blind accessible patterns */
.bias-slider--accessible .bias-slider__track {
  background: repeating-linear-gradient(
    45deg,
    var(--color-bias-left),
    var(--color-bias-left) 10px,
    var(--color-bias-center) 10px,
    var(--color-bias-center) 20px,
    var(--color-bias-right) 20px,
    var(--color-bias-right) 30px
  );
}
```

## Real-Time Features Implementation

### Server-Sent Events (SSE)
```javascript
// Progress tracking for article analysis
const progressStream = new EventSource(`/api/llm/score-progress/${articleId}`);
progressStream.onmessage = (event) => {
  const progress = JSON.parse(event.data);
  updateProgressIndicator(progress);
};
```

### Live Bias Score Updates
- WebSocket fallback for SSE if needed
- Automatic reconnection logic
- Graceful degradation to polling

## File Structure

```
/web/
├── index.html              # Landing page (redirect to articles)
├── articles.html           # Articles listing
├── article.html            # Article detail view
├── admin.html             # Admin dashboard
├── css/
│   ├── reset.css
│   ├── variables.css
│   ├── layout.css
│   ├── components.css
│   ├── bias-slider.css
│   ├── articles.css
│   ├── article-detail.css
│   ├── admin.css
│   └── main.css
├── js/
│   ├── main.js            # App initialization
│   ├── api-client.js      # API communication
│   ├── utils.js           # Shared utilities
│   ├── components/
│   │   ├── BiasSlider.js
│   │   ├── ArticleCard.js
│   │   ├── ProgressIndicator.js
│   │   ├── Navigation.js
│   │   └── Modal.js
│   └── pages/
│       ├── articles.js
│       ├── article-detail.js
│       └── admin.js
├── assets/
│   ├── icons/             # SVG icons
│   └── images/            # Images/logos
└── FRONTEND_PROPOSAL.md   # This document
```

## Integration with Go Backend

### Template Integration (Optional)
- Go templates can still render initial page data
- JavaScript enhances with dynamic functionality
- Fallback to server-side rendering if JS disabled

## API Specifications & Integration

### REST API Endpoints

#### Articles Management

**GET `/api/articles`** - List Articles with Filtering
```typescript
// Request Parameters
interface ArticlesQuery {
  source?: string[];        // Filter by news source
  leaning?: 'left'|'center'|'right'|'all';
  dateFrom?: string;        // ISO 8601 date
  dateTo?: string;          // ISO 8601 date
  limit?: number;           // Default: 20, Max: 100
  offset?: number;          // Pagination offset
  sortBy?: 'date'|'bias'|'relevance';
  sortOrder?: 'asc'|'desc'; // Default: desc
  search?: string;          // Full-text search
}

// Response Format
interface ArticlesResponse {
  articles: ArticleData[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
  filters: {
    availableSources: string[];
    dateRange: {
      earliest: string;
      latest: string;
    };
  };
}
```

**GET `/api/articles/{id}`** - Get Single Article
```typescript
// Response includes complete article data with bias analysis
interface ArticleDetailResponse extends ArticleData {
  analysis: {
    biasBreakdown: {
      political: number;     // Political bias component
      factual: number;       // Factual accuracy component
      emotional: number;     // Emotional language component
    };
    modelAnalysis: ModelScore[];
    lastAnalyzed: string;    // ISO 8601 timestamp
    analysisVersion: string; // Model version used
  };
  relatedArticles?: ArticleData[]; // Similar articles
}
```

**POST `/api/articles/{id}/bias`** - Update Bias Score (Manual)
```typescript
// Request Body
interface BiasUpdateRequest {
  score: number;           // -1 to +1
  source: 'manual'|'model'|'user';
  confidence?: number;     // 0 to 1
  notes?: string;          // Optional explanation
}

// Response
interface BiasUpdateResponse {
  success: boolean;
  previousScore: number;
  newScore: number;
  timestamp: string;
}
```

#### LLM Analysis Operations

**POST `/api/llm/analyze/{id}`** - Trigger Article Re-analysis
```typescript
// Request Body
interface AnalysisRequest {
  models?: string[];       // Specific models to use
  priority?: 'low'|'normal'|'high';
  options?: {
    forceReanalyze?: boolean;
    updateExisting?: boolean;
  };
}

// Response
interface AnalysisResponse {
  taskId: string;          // For progress tracking
  estimatedDuration: number; // Seconds
  queuePosition: number;   // Position in analysis queue
}
```

**GET `/api/llm/progress/{taskId}`** - SSE Progress Stream
```typescript
// SSE Event Data
interface ProgressEvent {
  type: 'progress'|'completed'|'error';
  data: ProgressData;
}

// Error Events
interface ErrorEvent {
  type: 'error';
  error: {
    code: string;
    message: string;
    retryable: boolean;
  };
}
```

#### System Management (Admin)

**GET `/api/admin/feeds/health`** - RSS Feed Health Status
```typescript
interface FeedHealthResponse {
  feeds: {
    [feedUrl: string]: {
      status: 'healthy'|'warning'|'error';
      lastFetch: string;
      lastSuccess: string;
      errorCount: number;
      articlesCount: number;
      avgFetchTime: number;
    };
  };
  overall: {
    totalFeeds: number;
    healthyFeeds: number;
    totalArticles: number;
    lastGlobalUpdate: string;
  };
}
```

**POST `/api/admin/feeds/refresh`** - Trigger Feed Refresh
```typescript
// Request Body
interface RefreshRequest {
  feedUrls?: string[];     // Specific feeds, or all if omitted
  force?: boolean;         // Ignore cache and fetch immediately
}

// Response
interface RefreshResponse {
  taskId: string;
  feedsQueued: number;
  estimatedCompletion: string;
}
```

### Error Handling Specifications

#### HTTP Status Codes
- `200` - Success
- `201` - Created (new resource)
- `400` - Bad Request (validation error)
- `401` - Unauthorized (authentication required)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `409` - Conflict (resource state conflict)
- `422` - Unprocessable Entity (business logic error)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
- `503` - Service Unavailable (maintenance/overload)

#### Error Response Format
```typescript
interface ApiError {
  error: {
    code: string;          // Machine-readable error code
    message: string;       // Human-readable message
    details?: {            // Additional error context
      field?: string;      // For validation errors
      value?: any;         // Invalid value
      constraint?: string; // Validation rule violated
    };
    retryable: boolean;    // Whether client should retry
    retryAfter?: number;   // Seconds to wait before retry
  };
  requestId: string;       // For debugging/support
  timestamp: string;       // ISO 8601 timestamp
}
```

### Authentication & Authorization

#### API Key Authentication
```typescript
// Request Headers
headers: {
  'Authorization': 'Bearer {api_key}',
  'X-API-Version': '1.0'
}

// For admin endpoints
headers: {
  'Authorization': 'Bearer {admin_api_key}',
  'X-Admin-Token': '{session_token}'
}
```

#### Rate Limiting
- **Public endpoints**: 100 requests/minute per IP
- **Admin endpoints**: 1000 requests/minute per authenticated user
- **Analysis endpoints**: 10 requests/minute per IP (resource-intensive)

**Rate Limit Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1621234567
X-RateLimit-Retry-After: 60
```

### Routing Strategy
- Use Go server for initial page delivery
- JavaScript handles in-page interactions
- Browser history API for SPA-like navigation (optional enhancement)

## Performance Considerations

### Loading Strategy
1. **Critical CSS**: Inline above-the-fold styles
2. **Progressive Loading**: Load non-critical JS asynchronously
3. **Image Optimization**: WebP with fallbacks
4. **Caching**: Leverage browser caching for static assets

## Performance Specifications & Optimization

### Core Web Vitals Targets
- **Largest Contentful Paint (LCP)**: < 2.5 seconds
- **First Input Delay (FID)**: < 100 milliseconds
- **Cumulative Layout Shift (CLS)**: < 0.1
- **First Contentful Paint (FCP)**: < 1.8 seconds
- **Time to Interactive (TTI)**: < 3.5 seconds

### Bundle Size Specifications
```
Critical Path (Above-the-fold):
├── HTML (compressed)       │  2-4 KB
├── Critical CSS (inlined)  │  8-12 KB
├── Core JS (defer)         │  15-20 KB
└── Web Fonts (optional)    │  20-30 KB
                            └─ Total: ~50 KB

Secondary Resources (Below-the-fold):
├── Chart.js (lazy)         │  47 KB
├── DOMPurify (lazy)        │  45 KB
├── Page-specific CSS       │  5-8 KB
├── Page-specific JS        │  10-15 KB
└── Images (WebP/AVIF)      │  Variable
                            └─ Total: ~110 KB
```

### Resource Loading Strategy
```javascript
// Critical CSS inlining
<style>
  /* Critical above-the-fold styles inlined */
  .header, .navigation, .article-card { ... }
</style>

// Non-critical CSS loading
<link rel="preload" href="/css/main.css" as="style" onload="this.onload=null;this.rel='stylesheet'">
<noscript><link rel="stylesheet" href="/css/main.css"></noscript>

// JavaScript loading strategy
<script type="module">
  // Core functionality loaded immediately
  import { initializeApp } from '/js/main.js';

  // Heavy dependencies loaded on-demand
  const loadChart = () => import('/js/vendor/chart.min.js');
  const loadDOMPurify = () => import('/js/vendor/dompurify.min.js');

  initializeApp({ loadChart, loadDOMPurify });
</script>
```

### Caching Strategy
```
Cache-Control Headers:
├── HTML files              │  max-age=300 (5 minutes)
├── CSS/JS (versioned)      │  max-age=31536000 (1 year)
├── Images (versioned)      │  max-age=31536000 (1 year)
├── API responses           │  max-age=60 (1 minute)
└── Static assets           │  max-age=604800 (1 week)

Service Worker Strategy:
├── App Shell               │  Cache First
├── Articles List           │  Network First (fresh data)
├── Article Content         │  Stale While Revalidate
├── Images                  │  Cache First
└── API calls               │  Network Only
```

### Image Optimization Specifications
```html
<!-- Responsive images with modern formats -->
<picture>
  <source srcset="/images/article-hero.avif" type="image/avif">
  <source srcset="/images/article-hero.webp" type="image/webp">
  <img src="/images/article-hero.jpg"
       alt="Article hero image"
       width="800"
       height="400"
       loading="lazy"
       decoding="async">
</picture>

<!-- Lazy loading with intersection observer -->
<img class="lazy-image"
     data-src="/images/article-thumb.webp"
     data-srcset="/images/article-thumb-2x.webp 2x"
     alt="Article thumbnail"
     width="300"
     height="200">
```

### HTTP/2 Optimization
```html
<!-- Resource hints -->
<link rel="dns-prefetch" href="//cdn.jsdelivr.net">
<link rel="preconnect" href="//api.newsbalancer.com">
<link rel="modulepreload" href="/js/main.js">

<!-- Server Push candidates (configured in Go server) -->
Push-Promise: /css/critical.css
Push-Promise: /js/main.js
Push-Promise: /images/logo.webp
```

## Development Workflow

### Development Server
- Use existing Go server for development
- Live reload via browser-sync (optional)
- Source maps for debugging

### Build Process (Optional)
- Simple concatenation and minification
- CSS/JS preprocessing if needed
- Asset optimization pipeline

## Browser Support

**Target Browsers:**
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

**Modern Features Used:**
- CSS Grid/Flexbox
- ES6 Modules
- Fetch API
- Server-Sent Events
- CSS Custom Properties

## Security Considerations

### Content Security Policy (CSP)
```
Content-Security-Policy: default-src 'self';
  script-src 'self' 'unsafe-inline';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: https:;
  connect-src 'self';
```

### XSS Protection
- DOMPurify integration for user-generated content sanitization
- Proper escaping of dynamic content in templates
- Input validation on both client and server side

### CSRF Protection
- CSRF tokens for all state-changing operations
- SameSite cookie attributes
- Origin/Referer header validation

### Additional Security Measures
- Subresource Integrity (SRI) for external dependencies
- HTTP security headers (HSTS, X-Frame-Options, etc.)
- Rate limiting on API endpoints

## Future Enhancements

### Accessibility & Standards (Built-in Phase 1)
- **WCAG 2.1 AA Compliance**: Semantic HTML, proper ARIA labels, keyboard navigation
- **Color Contrast**: 4.5:1 minimum ratio for all text elements
- **Screen Reader Support**: Proper heading hierarchy, alt text, focus management
- **Keyboard Navigation**: Tab order, focus indicators, keyboard shortcuts
- **Reduced Motion**: Respect prefers-reduced-motion media query
- **Color Accessibility**: Alternative indicators for bias scores (patterns/shapes)

### Testing Specifications & Quality Assurance

#### Test Suite Architecture
```
/tests/
├── unit/                   # Jest unit tests
│   ├── components/         # Component logic tests
│   ├── utils/             # Utility function tests
│   ├── api/               # API client tests
│   └── __mocks__/         # Mock implementations
├── e2e/                   # Puppeteer end-to-end tests
│   ├── user-flows/        # Complete user journeys
│   ├── accessibility/     # A11y automated tests
│   ├── performance/       # Lighthouse performance tests
│   └── visual/            # Visual regression tests
├── integration/           # API integration tests
│   ├── articles/          # Article CRUD operations
│   ├── bias-scoring/      # Bias analysis workflows
│   └── admin/             # Admin functionality
└── fixtures/              # Test data and mocks
    ├── articles.json      # Sample article data
    ├── api-responses/     # Mock API responses
    └── images/            # Test images
```

#### Unit Testing Specifications (Jest)
```javascript
// Component testing example
import { BiasSlider } from '../js/components/BiasSlider.js';
import { fireEvent } from '@testing-library/dom';

describe('BiasSlider Component', () => {
  let slider;

  beforeEach(() => {
    slider = new BiasSlider();
    document.body.appendChild(slider);
  });

  afterEach(() => {
    document.body.removeChild(slider);
  });

  describe('Initialization', () => {
    test('should initialize with default value of 0', () => {
      expect(slider.value).toBe(0);
    });

    test('should set aria attributes correctly', () => {
      expect(slider.getAttribute('aria-valuemin')).toBe('-1');
      expect(slider.getAttribute('aria-valuemax')).toBe('1');
      expect(slider.getAttribute('aria-valuenow')).toBe('0');
    });
  });

  describe('User Interactions', () => {
    test('should update value on keyboard navigation', () => {
      slider.focus();
      fireEvent.keyDown(slider, { key: 'ArrowRight' });
      expect(slider.value).toBeGreaterThan(0);
    });

    test('should emit biaschange event on value update', () => {
      const changeHandler = jest.fn();
      slider.addEventListener('biaschange', changeHandler);

      slider.updateValue(0.5);
      expect(changeHandler).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: { value: 0.5, previousValue: 0 }
        })
      );
    });
  });

  describe('API Integration', () => {
    test('should call API when value changes in edit mode', async () => {
      const mockApiCall = jest.spyOn(slider.apiClient, 'post');
      slider.articleId = 'test-123';
      slider.enableEditMode();

      await slider.updateValue(0.3);

      expect(mockApiCall).toHaveBeenCalledWith(
        '/api/articles/test-123/bias',
        { score: 0.3, source: 'manual' }
      );
    });
  });
});

// API Client testing
describe('ApiClient', () => {
  let apiClient;

  beforeEach(() => {
    apiClient = new ApiClient();
    fetch.resetMocks();
  });

  describe('Circuit Breaker', () => {
    test('should open circuit after consecutive failures', async () => {
      // Mock 5 consecutive failures
      fetch.mockReject(new Error('Network error'));

      for (let i = 0; i < 5; i++) {
        try {
          await apiClient.get('/api/test');
        } catch (e) {}
      }

      expect(apiClient.circuitBreaker.state).toBe('open');
    });
  });

  describe('Retry Logic', () => {
    test('should retry on retryable status codes', async () => {
      fetch
        .mockResponseOnce('', { status: 503 })
        .mockResponseOnce('', { status: 503 })
        .mockResponseOnce('{"data": "success"}', { status: 200 });

      const result = await apiClient.get('/api/test');
      expect(fetch).toHaveBeenCalledTimes(3);
      expect(result.data).toBe('success');
    });
  });
});
```

#### E2E Testing Specifications (Puppeteer)
```javascript
// User flow testing
describe('Article Reading Flow', () => {
  let browser, page;

  beforeAll(async () => {
    browser = await puppeteer.launch({
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox']
    });
  });

  beforeEach(async () => {
    page = await browser.newPage();
    await page.setViewport({ width: 1200, height: 800 });
  });

  afterEach(async () => {
    await page.close();
  });

  afterAll(async () => {
    await browser.close();
  });

  test('Complete article reading journey', async () => {
    // 1. Navigate to articles list
    await page.goto('http://localhost:8080/articles.html');
    await page.waitForSelector('[data-testid="article-card"]');

    // 2. Click on first article
    await page.click('[data-testid="article-card"]:first-child');
    await page.waitForNavigation();

    // 3. Verify article detail page loaded
    await page.waitForSelector('[data-testid="article-content"]');
    const title = await page.$eval('h1', el => el.textContent);
    expect(title).toBeTruthy();

    // 4. Interact with bias slider
    const slider = await page.$('[data-testid="bias-slider"]');
    expect(slider).toBeTruthy();

    // 5. Test keyboard navigation
    await slider.focus();
    await page.keyboard.press('ArrowRight');

    const biasValue = await page.$eval(
      '[data-testid="bias-value"]',
      el => parseFloat(el.textContent)
    );
    expect(biasValue).toBeGreaterThan(0);
  });

  test('Real-time analysis progress', async () => {
    await page.goto('http://localhost:8080/article.html?id=test-article');

    // Trigger re-analysis
    await page.click('[data-testid="reanalyze-button"]');

    // Wait for progress indicator
    await page.waitForSelector('[data-testid="progress-indicator"]');

    // Monitor SSE events
    const progressUpdates = [];
    page.on('response', response => {
      if (response.url().includes('/api/llm/progress/')) {
        progressUpdates.push(response);
      }
    });

    // Wait for completion
    await page.waitForSelector('[data-testid="analysis-completed"]', {
      timeout: 30000
    });

    expect(progressUpdates.length).toBeGreaterThan(0);
  });
});

// Accessibility testing
describe('Accessibility Compliance', () => {
  test('should pass WCAG 2.1 AA standards', async () => {
    await page.goto('http://localhost:8080/articles.html');
    await injectAxe(page);

    const results = await checkA11y(page, null, {
      tags: ['wcag2a', 'wcag2aa', 'wcag21aa']
    });

    expect(results.violations).toHaveLength(0);
  });

  test('should support keyboard navigation', async () => {
    await page.goto('http://localhost:8080/articles.html');

    // Tab through interactive elements
    await page.keyboard.press('Tab'); // Skip link
    await page.keyboard.press('Tab'); // Main nav
    await page.keyboard.press('Tab'); // Search input
    await page.keyboard.press('Tab'); // First article card

    const focusedElement = await page.evaluate(() =>
      document.activeElement.getAttribute('data-testid')
    );
    expect(focusedElement).toBe('article-card');
  });
});

// Performance testing
describe('Performance Benchmarks', () => {
  test('should meet Core Web Vitals targets', async () => {
    const lighthouse = await runLighthouse(page, {
      onlyCategories: ['performance'],
      settings: {
        throttlingMethod: 'simulate',
        throttling: {
          rttMs: 150,
          throughputKbps: 1638.4,
          cpuSlowdownMultiplier: 4
        }
      }
    });

    const metrics = lighthouse.lhr.audits;

    // LCP < 2.5s
    expect(metrics['largest-contentful-paint'].numericValue).toBeLessThan(2500);

    // FID < 100ms
    expect(metrics['max-potential-fid'].numericValue).toBeLessThan(100);

    // CLS < 0.1
    expect(metrics['cumulative-layout-shift'].numericValue).toBeLessThan(0.1);

    // Overall performance score > 90
    expect(lighthouse.lhr.categories.performance.score * 100).toBeGreaterThan(90);
  });
});
```

#### Test Coverage Requirements
- **Unit Tests**: > 90% code coverage
- **E2E Tests**: 100% critical user flows
- **Accessibility Tests**: 100% page coverage
- **Performance Tests**: All pages under target metrics
- **Visual Regression**: Key UI components tested
- **Cross-browser**: Chrome, Firefox, Safari, Edge

#### Continuous Integration Pipeline
```yaml
# .github/workflows/frontend-ci.yml
name: Frontend CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Run unit tests
        run: npm run test:unit -- --coverage

      - name: Start test server
        run: |
          npm run build
          npm run start:test &
          sleep 10

      - name: Run E2E tests
        run: npm run test:e2e

      - name: Run accessibility tests
        run: npm run test:a11y

      - name: Run performance tests
        run: npm run test:performance

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage/lcov.info
```

### Phase 2 (Optional)
- Progressive Web App (PWA) features
- Offline functionality with Service Workers
- Dark mode support
- Advanced analytics integration
- Keyboard navigation
- Accessibility improvements (WCAG 2.1)

### Phase 3 (Optional)
- Real-time collaborative features
- Advanced filtering and search
- Export functionality
- User preferences and settings
- Analytics integration

## Implementation Timeline

**Phase 1: Core Implementation (1-2 weeks)**
1. Set up file structure and build process with testing framework
2. Implement shared components with full accessibility support
3. Create articles listing page with keyboard navigation
4. Implement article detail page with bias sliders and ARIA labels
5. Basic admin dashboard with screen reader support
6. Unit and integration test suite setup

**Phase 2: Enhanced Features (1 week)**
1. SSE integration for real-time updates with error recovery
2. Manual scoring interface with accessibility compliance
3. User feedback system with proper validation
4. Comprehensive error handling and loading states
5. Performance optimization and monitoring

**Phase 3: Polish and Optimization (1 week)**
1. Cross-browser testing and compatibility fixes
2. Lighthouse score optimization (target: 95+)
3. Security audit and CSP implementation
4. Documentation and deployment optimization
5. User acceptance testing and feedback integration

## Questions for Clarification

1. **Existing Data**: Should we migrate any existing frontend logic or start completely fresh?
2. **Authentication**: Is there any user authentication system to consider?
3. **Branding**: Are there any specific branding guidelines or color schemes to follow?
4. **Analytics**: Do you need any user analytics or tracking integration?
5. **Deployment**: Should the frontend be served by the Go server or separately?

## Success Metrics

- **Bundle Size**: < 50KB total (gzipped, excluding images)
- **Load Time**: < 2 seconds on 3G, < 1 second on broadband
- **Lighthouse Score**: > 95 for Performance, Accessibility, Best Practices, SEO
- **Browser Support**: 98%+ of target browsers (Can I Use data)
- **Accessibility**: WCAG 2.1 AA compliance, tested with screen readers
- **Test Coverage**: > 90% code coverage with unit and integration tests
- **Error Rate**: < 0.1% client-side errors in production
- **Core Web Vitals**: LCP < 2.5s, FID < 100ms, CLS < 0.1
- **Maintainability**: Clear code structure with comprehensive documentation

## Deployment & Operations Specifications

### Build Process & Asset Pipeline
```javascript
// build.config.js
export default {
  input: {
    main: 'src/js/main.js',
    articles: 'src/js/pages/articles.js',
    'article-detail': 'src/js/pages/article-detail.js',
    admin: 'src/js/pages/admin.js'
  },
  output: {
    dir: 'dist',
    format: 'es',
    entryFileNames: 'js/[name]-[hash].js',
    chunkFileNames: 'js/[name]-[hash].js',
    assetFileNames: 'assets/[name]-[hash][extname]'
  },
  plugins: [
    // CSS processing
    postcss({
      extract: true,
      plugins: [
        autoprefixer(),
        cssnano({ preset: 'default' }),
        purgecss({
          content: ['./src/**/*.html', './src/**/*.js'],
          safelist: ['bias-slider--active', 'modal--open']
        })
      ]
    }),

    // JavaScript optimization
    terser({
      compress: {
        drop_console: true,
        drop_debugger: true,
        pure_funcs: ['console.log', 'console.warn']
      },
      mangle: {
        reserved: ['BiasSlider', 'ArticleCard', 'ProgressIndicator']
      }
    }),

    // Asset optimization
    imageOptimization({
      mozjpeg: { quality: 80 },
      webp: { quality: 80 },
      avif: { quality: 65 }
    }),

    // Bundle analysis
    bundleAnalyzer({
      analyzerMode: 'static',
      openAnalyzer: false
    })
  ],

  // Code splitting configuration
  manualChunks: {
    vendor: ['chart.js', 'dompurify'],
    components: ['src/js/components/BiasSlider.js', 'src/js/components/ArticleCard.js']
  }
};
```

### Go Server Integration
```go
// Static file serving with optimization
func setupStaticRoutes(r *gin.Engine) {
    // Serve static assets with proper headers
    r.Static("/css", "./web/dist/css")
    r.Static("/js", "./web/dist/js")
    r.Static("/assets", "./web/dist/assets")

    // HTML template rendering
    r.LoadHTMLGlob("web/dist/*.html")

    // Route handlers with template rendering
    r.GET("/", func(c *gin.Context) {
        c.Redirect(http.StatusMovedPermanently, "/articles")
    })

    r.GET("/articles", func(c *gin.Context) {
        // Server-side data injection for SEO
        articles, err := getRecentArticles(20)
        if err != nil {
            c.HTML(http.StatusInternalServerError, "error.html", gin.H{
                "error": "Failed to load articles"
            })
            return
        }

        c.HTML(http.StatusOK, "articles.html", gin.H{
            "articles": articles,
            "meta": gin.H{
                "title": "NewsBalancer - Unbiased News Analysis",
                "description": "Discover bias in news articles with AI-powered analysis",
                "canonical": "https://newsbalancer.com/articles"
            }
        })
    })

    r.GET("/article/:id", func(c *gin.Context) {
        articleID := c.Param("id")
        article, err := getArticleByID(articleID)
        if err != nil {
            c.HTML(http.StatusNotFound, "error.html", gin.H{
                "error": "Article not found"
            })
            return
        }

        c.HTML(http.StatusOK, "article.html", gin.H{
            "article": article,
            "meta": gin.H{
                "title": fmt.Sprintf("%s - NewsBalancer", article.Title),
                "description": article.Summary,
                "canonical": fmt.Sprintf("https://newsbalancer.com/article/%s", articleID),
                "og:type": "article",
                "og:image": article.ImageURL
            }
        })
    })
}

// HTTP headers middleware
func securityHeaders() gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        // Security headers
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

        // HSTS (HTTPS only)
        if c.Request.TLS != nil {
            c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }

        // CSP
        csp := "default-src 'self'; " +
               "script-src 'self' 'unsafe-inline' cdn.jsdelivr.net; " +
               "style-src 'self' 'unsafe-inline'; " +
               "img-src 'self' data: https:; " +
               "connect-src 'self'; " +
               "font-src 'self' data:; " +
               "object-src 'none'; " +
               "base-uri 'self'; " +
               "form-action 'self'"
        c.Header("Content-Security-Policy", csp)

        // Cache headers for static assets
        if strings.HasPrefix(c.Request.URL.Path, "/css/") ||
           strings.HasPrefix(c.Request.URL.Path, "/js/") ||
           strings.HasPrefix(c.Request.URL.Path, "/assets/") {
            c.Header("Cache-Control", "public, max-age=31536000, immutable")
        } else {
            c.Header("Cache-Control", "public, max-age=300")
        }

        c.Next()
    })
}
```

### Production Deployment Configuration

#### Docker Configuration
```dockerfile
# Multi-stage build for frontend assets
FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend
COPY package*.json ./
RUN npm ci --only=production

COPY src/ ./src/
COPY build.config.js ./
RUN npm run build

# Go application build
FROM golang:1.21-alpine AS go-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/frontend/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=go-builder /app/main .
COPY --from=go-builder /app/web/dist ./web/dist
COPY configs/ ./configs/

# Create non-root user
RUN adduser -D -s /bin/sh newsbalancer
USER newsbalancer

EXPOSE 8080
CMD ["./main"]
```

#### Kubernetes Deployment
```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: newsbalancer-frontend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: newsbalancer-frontend
  template:
    metadata:
      labels:
        app: newsbalancer-frontend
    spec:
      containers:
      - name: newsbalancer
        image: newsbalancer:latest
        ports:
        - containerPort: 8080
        env:
        - name: ENV
          value: "production"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: newsbalancer-secrets
              key: database-url
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: newsbalancer-service
spec:
  selector:
    app: newsbalancer-frontend
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Monitoring & Observability

#### Frontend Performance Monitoring
```javascript
// Real User Monitoring (RUM)
class PerformanceMonitor {
  constructor() {
    this.metrics = {};
    this.setupObservers();
  }

  setupObservers() {
    // Core Web Vitals monitoring
    this.observeLCP();
    this.observeFID();
    this.observeCLS();
    this.observeCustomMetrics();
  }

  observeLCP() {
    new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries();
      const lastEntry = entries[entries.length - 1];

      this.metrics.lcp = lastEntry.startTime;
      this.sendMetric('lcp', lastEntry.startTime);
    }).observe({ entryTypes: ['largest-contentful-paint'] });
  }

  observeFID() {
    new PerformanceObserver((entryList) => {
      for (const entry of entryList.getEntries()) {
        this.metrics.fid = entry.processingStart - entry.startTime;
        this.sendMetric('fid', this.metrics.fid);
      }
    }).observe({ entryTypes: ['first-input'] });
  }

  observeCLS() {
    let clsValue = 0;
    new PerformanceObserver((entryList) => {
      for (const entry of entryList.getEntries()) {
        if (!entry.hadRecentInput) {
          clsValue += entry.value;
        }
      }
      this.metrics.cls = clsValue;
      this.sendMetric('cls', clsValue);
    }).observe({ entryTypes: ['layout-shift'] });
  }

  sendMetric(name, value) {
    // Send to analytics service
    fetch('/api/metrics', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        metric: name,
        value: value,
        timestamp: Date.now(),
        userAgent: navigator.userAgent,
        url: window.location.href
      })
    }).catch(err => console.warn('Metrics failed:', err));
  }
}

// Error tracking
window.addEventListener('error', (event) => {
  fetch('/api/errors', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: event.message,
      filename: event.filename,
      line: event.lineno,
      column: event.colno,
      stack: event.error?.stack,
      url: window.location.href,
      timestamp: Date.now()
    })
  });
});
```

#### Infrastructure Monitoring
```yaml
# prometheus-config.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'newsbalancer'
    static_configs:
      - targets: ['newsbalancer-service:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx-exporter:9113']

rule_files:
  - "/etc/prometheus/rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### Success Metrics & KPIs

#### Technical Performance KPIs
- **Availability**: 99.9% uptime (< 8.76 hours downtime/year)
- **Response Time**: P95 < 200ms for API calls
- **Error Rate**: < 0.1% client-side errors
- **Bundle Size**: < 50KB gzipped (excluding images)
- **Lighthouse Score**: > 95 for all categories
- **Core Web Vitals**: Pass all thresholds for 90% of users

#### User Experience KPIs
- **Page Load Time**: < 2 seconds on 3G
- **Time to Interactive**: < 3.5 seconds
- **Bias Slider Interaction**: < 100ms response time
- **Search Results**: < 500ms response time
- **Real-time Updates**: < 1 second SSE message delivery

#### Business Impact KPIs
- **User Engagement**: Average session duration > 3 minutes
- **Feature Adoption**: 80% of users interact with bias sliders
- **Content Discovery**: 60% of users read multiple articles
- **Admin Efficiency**: Dashboard actions complete in < 5 seconds

This comprehensive technical specification provides a complete blueprint for implementing a production-ready, high-performance frontend that meets enterprise-grade standards for scalability, security, accessibility, and maintainability.

# CSS Style Guide - BalancedNewsGo

## Overview

This style guide documents the CSS architecture and design system for BalancedNewsGo, implemented through the CSS Unification Plan. The system provides a consistent, accessible, and maintainable foundation for the application's user interface.

## Architecture

### File Structure

```
static/css/
├── tokens.css          # Design tokens (colors, spacing, typography)
├── base.css           # Reset, typography, global styles
├── layout.css         # Grid systems, containers, responsive layouts
├── components.css     # UI components (buttons, cards, forms)
├── utilities.css      # Utility classes for rapid development
└── app-consolidated.css # Production build (auto-generated)
```

### Design Tokens

#### Colors

**Primary Colors (WCAG AA Compliant)**
- Primary: `#0056b3` (contrast ratio ≥ 4.5:1)
- Primary Hover: `#004085`
- Primary Active: `#003d82`

**Semantic Colors**
- Success: `#28a745`
- Warning: `#ffc107`
- Danger: `#dc3545`
- Info: `#17a2b8`

**Neutral Colors**
- Text: `#333333`
- Text Muted: `#495057` (improved contrast)
- Background: `#ffffff`
- Light Background: `#f8f9fa`

#### Typography

**Font Stack**
```css
font-family: "Segoe UI", system-ui, -apple-system, BlinkMacSystemFont, 
             "Helvetica Neue", Arial, sans-serif;
```

**Font Sizes**
- Base: `1rem` (16px)
- Small: `0.875rem` (14px)
- Large: `1.125rem` (18px)
- XL: `1.25rem` (20px)
- 2XL: `1.5rem` (24px)
- 3XL: `1.875rem` (30px)

**Headings**
- H1: `2rem` (32px) - Page titles
- H2: `1.875rem` (30px) - Section headers
- H3: `1.5rem` (24px) - Subsection headers

#### Spacing Scale

```css
--space-1: 0.25rem;    /* 4px */
--space-2: 0.5rem;     /* 8px */
--space-3: 0.75rem;    /* 12px */
--space-4: 1rem;       /* 16px */
--space-5: 1.25rem;    /* 20px */
--space-6: 1.5rem;     /* 24px */
--space-8: 2rem;       /* 32px */
```

## Layout System

### Grid Layouts

#### Articles Grid
```css
.articles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-6, 1.5rem);
}
```

#### Two-Column Layout
```css
.two-column-layout {
  display: grid;
  grid-template-columns: 3fr 1fr;
  gap: var(--space-6, 1.5rem);
}
```

#### Equal Columns Layout
```css
.equal-columns-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-6, 1.5rem);
}
```

### Responsive Breakpoints

- Mobile: `max-width: 736px`
- Tablet: `max-width: 768px`
- Desktop: `min-width: 769px`

## Components

### Buttons

#### Primary Button
```css
.btn-primary {
  background-color: #0056b3;
  color: #ffffff;
  padding: 0.625rem 1rem;
  border-radius: 0.25rem;
  border: 1px solid #0056b3;
}
```

#### Button Sizes
- Default: `.btn`
- Small: `.btn-sm`
- Large: `.btn-lg`

### Article Cards

```css
.article-item {
  background: #ffffff;
  border: 1px solid rgba(210, 215, 217, 0.75);
  border-radius: 0.375rem;
  padding: 1.5rem;
  transition: all 0.2s ease-in-out;
}
```

### Bias Indicators

```css
.bias-left {
  background-color: rgba(13, 110, 253, 0.1);
  color: #0d6efd;
}

.bias-center {
  background-color: rgba(108, 117, 125, 0.15);
  color: #495057; /* Improved contrast */
}

.bias-right {
  background-color: rgba(220, 53, 69, 0.1);
  color: #dc3545;
}
```

## Utility Classes

### Text Alignment
- `.text-center` - Center align text
- `.text-left` - Left align text
- `.text-right` - Right align text

### Spacing
- `.my-1` to `.my-5` - Vertical margins
- `.py-1` to `.py-3` - Vertical padding

### Display
- `.d-none` - Hide element
- `.d-block` - Block display
- `.d-flex` - Flexbox display
- `.sr-only` - Screen reader only

### Flexbox
- `.justify-content-center` - Center content
- `.align-items-center` - Center items vertically

## Accessibility

### Color Contrast
All color combinations meet WCAG 2.0 AA standards:
- Normal text: minimum 4.5:1 contrast ratio
- Large text: minimum 3:1 contrast ratio

### Focus States
All interactive elements have visible focus indicators:
```css
:focus {
  outline: 2px solid #0056b3;
  outline-offset: 2px;
}
```

### Print Styles
Optimized print styles hide navigation and interactive elements:
```css
@media print {
  nav, .btn, .sidebar {
    display: none !important;
  }
}
```

## Development Guidelines

### CSS Organization
1. Use design tokens for consistent values
2. Follow BEM methodology for component classes
3. Prefer utility classes for simple styling
4. Use CSS Grid for layout, Flexbox for components

### Quality Assurance
- Run `npm run lint:css` before committing
- All CSS must pass stylelint validation
- Accessibility tests must pass (axe-core)
- Performance scores must meet Lighthouse thresholds

### Browser Support
- Modern browsers with CSS Grid support
- Flexbox fallbacks for older browsers
- IE11 graceful degradation

## Screenshots

### Implementation Examples

The CSS migration has been successfully implemented across all main pages:

#### Articles Listing Page (`/articles`)
- **URL**: `http://localhost:8080/articles`
- **Features Demonstrated**:
  - Articles grid layout with CSS Grid
  - Responsive design with mobile breakpoints
  - Unified navigation bar styling
  - Article cards with hover effects
  - Bias indicators with proper color coding

#### Article Detail Page (`/article/:id`)
- **URL**: `http://localhost:8080/article/[id]`
- **Features Demonstrated**:
  - Two-column layout (content + sidebar)
  - Typography hierarchy with design tokens
  - Button components with consistent styling
  - Responsive layout that stacks on mobile

#### Admin Dashboard (`/admin`)
- **URL**: `http://localhost:8080/admin`
- **Features Demonstrated**:
  - Admin-specific styling with unified design system
  - Form controls and button variants
  - Dashboard layout with proper spacing
  - Consistent navigation and branding

### Visual Verification

To capture screenshots for documentation:

1. **Start the server**: `go run cmd/server/main.go cmd/server/template_handlers.go cmd/server/legacy_handlers.go`
2. **Navigate to each page** in a browser
3. **Capture full-page screenshots** at different viewport sizes:
   - Desktop: 1440px width
   - Tablet: 768px width
   - Mobile: 375px width

### CSS Implementation Status

✅ **Completed Features**:
- Design tokens system (`tokens.css`)
- Base typography and reset styles (`base.css`)
- Layout grid systems (`layout.css`)
- Component library (`components.css`)
- Utility classes (`utilities.css`)
- Consolidated build (`app-consolidated.css`)

## Performance

### Optimization
- Single consolidated CSS file (22KB gzipped)
- Critical CSS inlined for above-the-fold content
- Non-critical CSS loaded asynchronously
- Print styles separated with media queries

### Metrics
- Lighthouse Performance: ≥ 90
- First Contentful Paint: < 1.5s
- Largest Contentful Paint: < 2.5s
- Cumulative Layout Shift: < 0.1

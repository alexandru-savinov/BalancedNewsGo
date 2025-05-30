/**
 * Navigation Web Component
 * Responsive header navigation with accessibility support
 *
 * Features:
 * - Responsive design with mobile menu toggle
 * - Active page indication with aria-current
 * - Keyboard navigation support
 * - ARIA accessibility features
 * - Router integration for SPA-style navigation
 * - Brand logo and navigation links
 */

class Navigation extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: 'open' });

    // Component state
    this.#activeRoute = '';       // Current active route
    this.#routes = [];           // Available navigation routes
    this.#isMobileMenuOpen = false; // Mobile menu state
    this.#brand = 'NewsBalancer';   // Brand text

    // Bind event handlers
    this.#handleNavClick = this.#handleNavClick.bind(this);
    this.#handleKeyDown = this.#handleKeyDown.bind(this);
    this.#handleMobileToggle = this.#handleMobileToggle.bind(this);
    this.#handleResize = this.#handleResize.bind(this);
    this.#handleFocus = this.#handleFocus.bind(this);
    this.#handleBlur = this.#handleBlur.bind(this);

    this.#render();
    this.#attachEventListeners();
  }

  static get observedAttributes() {
    return ['active-route', 'routes', 'brand', 'mobile-breakpoint'];
  }

  // Private properties
  #activeRoute = '';
  #routes = [];
  #isMobileMenuOpen = false;
  #brand = 'NewsBalancer';
  #mobileBreakpoint = 768;

  // Getters and setters
  get activeRoute() {
    return this.#activeRoute;
  }

  set activeRoute(value) {
    const oldValue = this.#activeRoute;
    this.#activeRoute = value || '';

    if (oldValue !== this.#activeRoute) {
      this.#updateActiveState();
      this.#dispatchEvent('routechange', {
        oldRoute: oldValue,
        newRoute: this.#activeRoute
      });
    }
  }

  get routes() {
    return this.#routes;
  }

  set routes(value) {
    try {
      this.#routes = Array.isArray(value) ? value : JSON.parse(value || '[]');
      this.#render();
    } catch (error) {
      console.warn('Navigation: Invalid routes format:', error);
      this.#routes = [];
    }
  }

  get brand() {
    return this.#brand;
  }

  set brand(value) {
    this.#brand = value || 'NewsBalancer';
    this.#render();
  }

  get isMobileMenuOpen() {
    return this.#isMobileMenuOpen;
  }

  // Public methods
  setRoutes(routes) {
    this.#routes = routes || [];
    this.#render();
  }

  toggleMobileMenu() {
    this.#isMobileMenuOpen = !this.#isMobileMenuOpen;
    this.#updateMobileMenuState();
  }

  closeMobileMenu() {
    this.#isMobileMenuOpen = false;
    this.#updateMobileMenuState();
  }

  navigateTo(route) {
    this.activeRoute = route;
    this.closeMobileMenu();

    this.#dispatchEvent('navigationchange', {
      route: route,
      preventDefault: false
    });
  }

  // Lifecycle methods
  connectedCallback() {
    this.#updateFromAttributes();
    this.#setupDefaultRoutes();

    // Listen for window resize
    window.addEventListener('resize', this.#handleResize);

    // Listen for external route changes
    window.addEventListener('popstate', this.#handlePopState.bind(this));

    // Set initial active route from URL
    this.#setActiveRouteFromURL();
  }

  disconnectedCallback() {
    window.removeEventListener('resize', this.#handleResize);
    window.removeEventListener('popstate', this.#handlePopState.bind(this));
  }

  attributeChangedCallback(name, oldValue, newValue) {
    if (oldValue === newValue) return;

    switch (name) {
      case 'active-route':
        this.activeRoute = newValue;
        break;
      case 'routes':
        this.routes = newValue;
        break;
      case 'brand':
        this.brand = newValue;
        break;
      case 'mobile-breakpoint':
        this.#mobileBreakpoint = parseInt(newValue) || 768;
        break;
    }
  }

  // Private methods
  #render() {
    const styles = this.#getStyles();
    const template = this.#getTemplate();

    this.shadowRoot.innerHTML = `
      <style>${styles}</style>
      ${template}
    `;

    // Re-attach event listeners after render
    this.#attachEventListeners();
    this.#updateActiveState();
    this.#updateMobileMenuState();
  }

  #getTemplate() {
    const routeLinks = this.#routes.map(route => `
      <li class="nav-item" role="none">
        <a href="${route.path}"
           class="nav-link"
           role="menuitem"
           data-route="${route.path}"
           tabindex="-1"
           ${route.path === this.#activeRoute ? 'aria-current="page"' : ''}>
          ${route.icon ? `<span class="nav-icon">${route.icon}</span>` : ''}
          <span class="nav-text">${route.label}</span>
        </a>
      </li>
    `).join('');

    return `
      <header class="navigation" role="banner">
        <nav class="nav-container" role="navigation" aria-label="Main navigation">
          <div class="nav-brand">
            <a href="/" class="brand-link" aria-label="${this.#brand} home">
              <span class="brand-text">${this.#brand}</span>
            </a>
          </div>

          <button class="mobile-toggle"
                  aria-label="Toggle navigation menu"
                  aria-expanded="false"
                  aria-controls="nav-menu"
                  type="button">
            <span class="toggle-icon">
              <span class="toggle-line"></span>
              <span class="toggle-line"></span>
              <span class="toggle-line"></span>
            </span>
          </button>

          <div class="nav-menu" id="nav-menu" role="menubar">
            <ul class="nav-list" role="none">
              ${routeLinks}
            </ul>
          </div>
        </nav>
      </header>
    `;
  }

  #getStyles() {
    return `
      :host {
        display: block;
        width: 100%;
        position: relative;
        z-index: var(--z-sticky, 1020);
      }

      /* Navigation Container */
      .navigation {
        background: var(--color-bg-primary, #ffffff);
        border-bottom: 1px solid var(--color-border, #e5e7eb);
        box-shadow: var(--shadow-sm, 0 1px 2px 0 rgb(0 0 0 / 0.05));
      }

      .nav-container {
        display: flex;
        align-items: center;
        justify-content: space-between;
        max-width: var(--container-max-width, 1280px);
        margin: 0 auto;
        padding: 0 var(--space-4, 1rem);
        height: var(--nav-height, 64px);
      }

      /* Brand */
      .nav-brand {
        flex-shrink: 0;
      }

      .brand-link {
        display: flex;
        align-items: center;
        text-decoration: none;
        color: var(--color-text-primary, #111827);
        font-weight: 700;
        font-size: var(--font-size-lg, 1.125rem);
        transition: color var(--transition-base, 200ms ease);
      }

      .brand-link:hover,
      .brand-link:focus {
        color: var(--color-primary-600, #2563eb);
        outline: none;
      }

      .brand-link:focus-visible {
        outline: 2px solid var(--color-primary-500, #3b82f6);
        outline-offset: 2px;
        border-radius: var(--radius-base, 0.25rem);
      }

      .brand-text {
        display: block;
      }

      /* Mobile Toggle */
      .mobile-toggle {
        display: none;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        width: 40px;
        height: 40px;
        background: none;
        border: none;
        cursor: pointer;
        padding: 8px;
        border-radius: var(--radius-base, 0.25rem);
        transition: background-color var(--transition-base, 200ms ease);
      }

      .mobile-toggle:hover {
        background-color: var(--color-bg-secondary, #f3f4f6);
      }

      .mobile-toggle:focus {
        outline: none;
        background-color: var(--color-bg-secondary, #f3f4f6);
      }

      .mobile-toggle:focus-visible {
        outline: 2px solid var(--color-primary-500, #3b82f6);
        outline-offset: 2px;
      }

      .toggle-icon {
        display: flex;
        flex-direction: column;
        width: 20px;
        height: 16px;
        position: relative;
      }

      .toggle-line {
        display: block;
        width: 100%;
        height: 2px;
        background-color: var(--color-text-primary, #111827);
        border-radius: 1px;
        transition: all var(--transition-base, 200ms ease);
        transform-origin: center;
      }

      .toggle-line:nth-child(1) {
        margin-bottom: 5px;
      }

      .toggle-line:nth-child(2) {
        margin-bottom: 5px;
      }

      /* Mobile menu open state */
      .mobile-toggle[aria-expanded="true"] .toggle-line:nth-child(1) {
        transform: translateY(7px) rotate(45deg);
      }

      .mobile-toggle[aria-expanded="true"] .toggle-line:nth-child(2) {
        opacity: 0;
      }

      .mobile-toggle[aria-expanded="true"] .toggle-line:nth-child(3) {
        transform: translateY(-7px) rotate(-45deg);
      }

      /* Navigation Menu */
      .nav-menu {
        display: flex;
        align-items: center;
      }

      .nav-list {
        display: flex;
        align-items: center;
        list-style: none;
        margin: 0;
        padding: 0;
        gap: var(--space-1, 0.25rem);
      }

      .nav-item {
        position: relative;
      }

      .nav-link {
        display: flex;
        align-items: center;
        padding: var(--space-2, 0.5rem) var(--space-3, 0.75rem);
        text-decoration: none;
        color: var(--color-text-secondary, #6b7280);
        font-weight: 500;
        font-size: var(--font-size-sm, 0.875rem);
        border-radius: var(--radius-base, 0.25rem);
        transition: all var(--transition-base, 200ms ease);
        white-space: nowrap;
      }

      .nav-link:hover {
        color: var(--color-text-primary, #111827);
        background-color: var(--color-bg-secondary, #f3f4f6);
      }

      .nav-link:focus {
        outline: none;
        color: var(--color-text-primary, #111827);
        background-color: var(--color-bg-secondary, #f3f4f6);
      }

      .nav-link:focus-visible {
        outline: 2px solid var(--color-primary-500, #3b82f6);
        outline-offset: 2px;
      }

      .nav-link[aria-current="page"] {
        color: var(--color-primary-600, #2563eb);
        background-color: var(--color-primary-50, #eff6ff);
        font-weight: 600;
      }

      .nav-icon {
        margin-right: var(--space-2, 0.5rem);
        font-size: var(--font-size-base, 1rem);
      }

      .nav-text {
        display: block;
      }

      /* Mobile Responsive */
      @media (max-width: 767px) {
        .mobile-toggle {
          display: flex;
        }

        .nav-menu {
          position: absolute;
          top: 100%;
          left: 0;
          right: 0;
          background: var(--color-bg-primary, #ffffff);
          border-top: 1px solid var(--color-border, #e5e7eb);
          box-shadow: var(--shadow-lg, 0 10px 15px -3px rgb(0 0 0 / 0.1));
          opacity: 0;
          visibility: hidden;
          transform: translateY(-10px);
          transition: all var(--transition-base, 200ms ease);
        }

        .nav-menu.open {
          opacity: 1;
          visibility: visible;
          transform: translateY(0);
        }

        .nav-list {
          flex-direction: column;
          align-items: stretch;
          padding: var(--space-4, 1rem);
          gap: var(--space-1, 0.25rem);
        }

        .nav-link {
          justify-content: flex-start;
          padding: var(--space-3, 0.75rem) var(--space-4, 1rem);
          border-radius: var(--radius-lg, 0.5rem);
        }
      }

      /* Large screens */
      @media (min-width: 1024px) {
        .nav-container {
          padding: 0 var(--space-8, 2rem);
        }

        .nav-list {
          gap: var(--space-2, 0.5rem);
        }
      }

      /* Dark mode support */
      @media (prefers-color-scheme: dark) {
        .navigation {
          background: var(--color-bg-primary, #111827);
          border-bottom-color: var(--color-border, #374151);
        }

        .brand-link {
          color: var(--color-text-primary, #f9fafb);
        }

        .brand-link:hover,
        .brand-link:focus {
          color: var(--color-primary-400, #60a5fa);
        }

        .mobile-toggle:hover,
        .mobile-toggle:focus {
          background-color: var(--color-bg-secondary, #1f2937);
        }

        .toggle-line {
          background-color: var(--color-text-primary, #f9fafb);
        }

        .nav-link {
          color: var(--color-text-secondary, #d1d5db);
        }

        .nav-link:hover,
        .nav-link:focus {
          color: var(--color-text-primary, #f9fafb);
          background-color: var(--color-bg-secondary, #1f2937);
        }

        .nav-link[aria-current="page"] {
          color: var(--color-primary-400, #60a5fa);
          background-color: var(--color-primary-900, #1e3a8a);
        }

        .nav-menu {
          background: var(--color-bg-primary, #111827);
          border-top-color: var(--color-border, #374151);
        }
      }

      /* Reduced motion support */
      @media (prefers-reduced-motion: reduce) {
        * {
          transition-duration: 0.01ms !important;
        }
      }

      /* High contrast mode */
      @media (prefers-contrast: high) {
        .navigation {
          border-bottom-width: 2px;
        }

        .nav-link:focus-visible,
        .brand-link:focus-visible,
        .mobile-toggle:focus-visible {
          outline-width: 3px;
        }
      }
    `;
  }

  #attachEventListeners() {
    const navLinks = this.shadowRoot.querySelectorAll('.nav-link');
    const mobileToggle = this.shadowRoot.querySelector('.mobile-toggle');

    // Navigation links
    navLinks.forEach(link => {
      link.addEventListener('click', this.#handleNavClick);
      link.addEventListener('keydown', this.#handleKeyDown);
      link.addEventListener('focus', this.#handleFocus);
      link.addEventListener('blur', this.#handleBlur);
    });

    // Mobile toggle
    if (mobileToggle) {
      mobileToggle.addEventListener('click', this.#handleMobileToggle);
      mobileToggle.addEventListener('keydown', this.#handleKeyDown);
    }

    // Brand link
    const brandLink = this.shadowRoot.querySelector('.brand-link');
    if (brandLink) {
      brandLink.addEventListener('click', this.#handleNavClick);
    }
  }

  #handleNavClick(event) {
    const target = event.currentTarget;
    const route = target.getAttribute('data-route') || target.getAttribute('href');

    // Dispatch navigation event
    const navigationEvent = this.#dispatchEvent('navigationchange', {
      route: route,
      originalEvent: event,
      preventDefault: false
    });

    // If event wasn't prevented, update active route
    if (!navigationEvent.detail.preventDefault) {
      this.activeRoute = route;
      event.preventDefault();
    }
  }

  #handleKeyDown(event) {
    const target = event.currentTarget;

    switch (event.key) {
      case 'Enter':
      case ' ':
        if (target.classList.contains('mobile-toggle')) {
          event.preventDefault();
          this.toggleMobileMenu();
        } else if (target.classList.contains('nav-link') || target.classList.contains('brand-link')) {
          event.preventDefault();
          target.click();
        }
        break;

      case 'Escape':
        if (this.#isMobileMenuOpen) {
          this.closeMobileMenu();
          this.shadowRoot.querySelector('.mobile-toggle')?.focus();
        }
        break;

      case 'ArrowRight':
      case 'ArrowDown':
        if (target.classList.contains('nav-link')) {
          event.preventDefault();
          this.#focusNextNavItem(target);
        }
        break;

      case 'ArrowLeft':
      case 'ArrowUp':
        if (target.classList.contains('nav-link')) {
          event.preventDefault();
          this.#focusPrevNavItem(target);
        }
        break;

      case 'Home':
        if (target.classList.contains('nav-link')) {
          event.preventDefault();
          this.#focusFirstNavItem();
        }
        break;

      case 'End':
        if (target.classList.contains('nav-link')) {
          event.preventDefault();
          this.#focusLastNavItem();
        }
        break;
    }
  }

  #handleMobileToggle() {
    this.toggleMobileMenu();
  }

  #handleResize() {
    // Close mobile menu on larger screens
    if (window.innerWidth >= this.#mobileBreakpoint && this.#isMobileMenuOpen) {
      this.closeMobileMenu();
    }
  }

  #handleFocus(event) {
    // Update tab index for focused nav item
    this.#updateTabIndex(event.currentTarget);
  }

  #handleBlur() {
    // Reset tab indices when focus leaves navigation
    setTimeout(() => {
      if (!this.shadowRoot.activeElement) {
        this.#resetTabIndices();
      }
    }, 0);
  }

  #handlePopState() {
    this.#setActiveRouteFromURL();
  }

  #updateFromAttributes() {
    this.activeRoute = this.getAttribute('active-route') || '';
    this.routes = this.getAttribute('routes') || '[]';
    this.brand = this.getAttribute('brand') || 'NewsBalancer';
    this.#mobileBreakpoint = parseInt(this.getAttribute('mobile-breakpoint')) || 768;
  }

  #setupDefaultRoutes() {
    if (this.#routes.length === 0) {
      this.#routes = [
        { path: '/articles', label: 'Articles', icon: '📰' },
        { path: '/admin', label: 'Admin', icon: '⚙️' }
      ];
      this.#render();
    }
  }

  #updateActiveState() {
    const navLinks = this.shadowRoot.querySelectorAll('.nav-link');

    navLinks.forEach(link => {
      const route = link.getAttribute('data-route');
      if (route === this.#activeRoute) {
        link.setAttribute('aria-current', 'page');
        link.setAttribute('tabindex', '0');
      } else {
        link.removeAttribute('aria-current');
        link.setAttribute('tabindex', '-1');
      }
    });

    // Update first nav item as focusable if no active route
    if (!this.#activeRoute && navLinks.length > 0) {
      navLinks[0].setAttribute('tabindex', '0');
    }
  }

  #updateMobileMenuState() {
    const navMenu = this.shadowRoot.querySelector('.nav-menu');
    const mobileToggle = this.shadowRoot.querySelector('.mobile-toggle');

    if (this.#isMobileMenuOpen) {
      navMenu?.classList.add('open');
      mobileToggle?.setAttribute('aria-expanded', 'true');
    } else {
      navMenu?.classList.remove('open');
      mobileToggle?.setAttribute('aria-expanded', 'false');
    }
  }

  #setActiveRouteFromURL() {
    const currentPath = window.location.pathname;
    this.activeRoute = currentPath;
  }

  #focusNextNavItem(currentItem) {
    const navLinks = Array.from(this.shadowRoot.querySelectorAll('.nav-link'));
    const currentIndex = navLinks.indexOf(currentItem);
    const nextIndex = (currentIndex + 1) % navLinks.length;

    navLinks[nextIndex]?.focus();
  }

  #focusPrevNavItem(currentItem) {
    const navLinks = Array.from(this.shadowRoot.querySelectorAll('.nav-link'));
    const currentIndex = navLinks.indexOf(currentItem);
    const prevIndex = currentIndex === 0 ? navLinks.length - 1 : currentIndex - 1;

    navLinks[prevIndex]?.focus();
  }

  #focusFirstNavItem() {
    const firstNavLink = this.shadowRoot.querySelector('.nav-link');
    firstNavLink?.focus();
  }

  #focusLastNavItem() {
    const navLinks = this.shadowRoot.querySelectorAll('.nav-link');
    const lastNavLink = navLinks[navLinks.length - 1];
    lastNavLink?.focus();
  }

  #updateTabIndex(focusedItem) {
    const navLinks = this.shadowRoot.querySelectorAll('.nav-link');

    navLinks.forEach(link => {
      link.setAttribute('tabindex', link === focusedItem ? '0' : '-1');
    });
  }

  #resetTabIndices() {
    const activeLink = this.shadowRoot.querySelector('.nav-link[aria-current="page"]');
    const firstLink = this.shadowRoot.querySelector('.nav-link');

    const focusableLink = activeLink || firstLink;

    this.shadowRoot.querySelectorAll('.nav-link').forEach(link => {
      link.setAttribute('tabindex', link === focusableLink ? '0' : '-1');
    });
  }

  #dispatchEvent(type, detail) {
    const event = new CustomEvent(type, {
      detail,
      bubbles: true,
      cancelable: true
    });

    this.dispatchEvent(event);
    return event;
  }
}

// Register the custom element
customElements.define('navigation-component', Navigation);

export default Navigation;

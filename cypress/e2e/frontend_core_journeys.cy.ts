/// <reference types="cypress" />

/**
 * Cypress E2E tests for the core user journeys of the React frontend.
 * Covers:
 * - Dashboard Loading
 * - Article Detail View &amp; Debug Info
 * - Re-analyze Interaction
 * - Feedback Submission
 */
describe('Frontend Core User Journeys', () => {
  beforeEach(() => {
    // Intercept the initial articles API call to ensure data is loaded before tests run
    // Adjust the alias and URL pattern if needed
    // Add logging within the intercept handler
    // cy.intercept('GET', 'http://localhost:8080/api/articles', (req) => {
    //   console.log('Cypress: Intercepting GET /api/articles request');
    // }).as('getArticles');
    cy.visit('/'); // Visit the base URL defined in cypress.config.js
    // Wait and assertion moved to the relevant test ('Dashboard Loading')
  });

  context('Dashboard Loading', () => {
    it.only('should load the homepage and display a list of articles', () => {
      cy.visit('/'); // Keep visit here for .only test isolation
      cy.log('Checking for ArticleList container...');
      cy.get('[data-testid="article-list-container"]', { timeout: 20000 })
        .should('be.visible')
        .invoke('prop', 'outerHTML')
        .then(html => {
          cy.log('RAW HTML:', html);
          // eslint-disable-next-line no-console
          console.log('[CYPRESS DEBUG] ArticleList container outerHTML:', html);
        });
    });
  });

  context('Article Detail View', () => {
    beforeEach(() => {
      // Click on the first article card to expand it. Adjust selector.
      cy.get('[data-testid^="article-card-"]').should('have.length.greaterThan', 0); // Ensure cards are present before selecting
      cy.get('[data-testid^="article-card-"]').first().as('firstArticle');
      cy.get('@firstArticle').click();
      // Potentially wait for detail API call if details are fetched separately
      // cy.intercept('GET', 'http://localhost:8080/api/articles/*').as('getArticleDetail');
      // cy.wait('@getArticleDetail');
    });

    it('should display essential debug information when an article card is expanded', () => {
      // Verify essential debug info is displayed. Adjust selectors based on actual implementation.
      cy.get('@firstArticle').find('[data-testid="composite-score"]').should('be.visible');
      cy.get('@firstArticle').find('[data-testid="article-source"]').should('be.visible');
      cy.get('@firstArticle').find('[data-testid="fetch-timestamp"]').should('be.visible');
      cy.get('@firstArticle').find('[data-testid="score-timestamp"]').should('be.visible');
    });

    it('should render the BiasSlider component', () => {
      // Check if the BiasSlider component is rendered. Adjust selector.
      cy.get('@firstArticle').find('[data-testid="bias-slider"]').should('be.visible');
    });

    // Optional bonus test
    it.skip('should show tooltips on hover (basic check)', () => {
      // Basic check for tooltip presence. This might need refinement based on tooltip implementation.
      // cy.get('@firstArticle').find('[data-tooltip-target]').first().trigger('mouseover');
      // cy.get('[role="tooltip"]').should('be.visible');
    });
  });

  context('Re-analyze Interaction', () => {
    beforeEach(() => {
      // Expand the first article card
      cy.get('[data-testid^="article-card-"]').should('have.length.greaterThan', 0); // Ensure cards are present before selecting
      cy.get('[data-testid^="article-card-"]').first().as('firstArticle').click();
      // Intercept the re-analyze API call. Adjust URL pattern and alias.
      // Assuming the article ID is part of the data-testid or can be extracted.
      cy.get('@firstArticle').invoke('attr', 'data-testid').then((testId) => {
        const articleId = testId?.split('-').pop(); // Example: Extract ID
        if (articleId) {
          cy.intercept('POST', `http://localhost:8080/api/llm/reanalyze/${articleId}`).as('reanalyze');
        } else {
          // Fallback or fail if ID cannot be extracted
          cy.intercept('POST', 'http://localhost:8080/api/llm/reanalyze/*').as('reanalyze');
        }
      });
    });

    it('should trigger re-analysis and show progress', () => {
      // Click the re-analyze button. Adjust selector.
      cy.get('@firstArticle').find('[data-testid="reanalyze-button"]').click();

      // Verify the API call was made
      cy.wait('@reanalyze').its('response.statusCode').should('be.oneOf', [200, 202]); // Allow 200 or 202 Accepted

      // Verify progress indicator appears. Adjust selector/text content check.
      cy.get('@firstArticle').find('[data-testid="reanalyze-progress"]').should('be.visible').and('contain.text', 'Re-analyzing'); // Example text
    });
  });

  context('Feedback Submission', () => {
     beforeEach(() => {
      // Expand the first article card
      cy.get('[data-testid^="article-card-"]').should('have.length.greaterThan', 0); // Ensure cards are present before selecting
      cy.get('[data-testid^="article-card-"]').first().as('firstArticle').click();
      // Intercept the feedback API call. Adjust URL pattern and alias.
      cy.intercept('POST', 'http://localhost:8080/api/feedback').as('submitFeedback');
    });

    it('should allow submitting feedback and verify the API call', () => {
      // Fill out the feedback form. Adjust selectors.
      cy.get('@firstArticle').find('[data-testid="feedback-comments"]').type('This is a test comment.');
      cy.get('@firstArticle').find('[data-testid="feedback-rating"]').select('5'); // Assuming a select dropdown for rating
      cy.get('@firstArticle').find('[data-testid="feedback-tags"]').type('test, cypress'); // Assuming a text input for tags

      // Submit the form. Adjust selector.
      cy.get('@firstArticle').find('[data-testid="submit-feedback-button"]').click();

      // Verify the API call was made with correct data
      cy.wait('@submitFeedback').then((interception) => {
        expect(interception.request.body).to.have.property('comments', 'This is a test comment.');
        expect(interception.request.body).to.have.property('rating', 5); // Assuming rating is sent as a number
        expect(interception.request.body).to.have.property('tags').and.deep.equal(['test', 'cypress']); // Assuming tags are sent as an array
        // Add check for article ID if it's part of the payload
        // expect(interception.request.body).to.have.property('article_id');
      });

      // Verify success message or form reset. Adjust selector/assertion.
      cy.get('@firstArticle').find('[data-testid="feedback-success-message"]').should('be.visible');
      // OR check if form fields are cleared
      // cy.get('@firstArticle').find('[data-testid="feedback-comments"]').should('have.value', '');
    });
  });
});
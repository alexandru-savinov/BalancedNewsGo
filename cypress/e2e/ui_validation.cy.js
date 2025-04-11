// Cypress E2E tests for UI Validation: CompositeScore, tooltips, confidence indicators, and error handling

describe('UI Validation: CompositeScore and Indicators', () => {
  beforeEach(() => {
    cy.visit('/');
  });

  it('should display articles with CompositeScore, tooltips, and confidence indicators', () => {
    // Wait for articles to load (adjust selector as needed)
    cy.get('.article, .article-row, .article-list-item').should('exist');

    cy.get('.article, .article-row, .article-list-item').each(($article, idx) => {
      cy.wrap($article).within(() => {
        // CompositeScore
        cy.get('.composite-score')
          .should('exist')
          .invoke('text')
          .then((text) => {
            cy.log(`Article[${idx}] CompositeScore text: ${text}`);
            expect(text).to.match(/Score: ([-\d.]+|Not yet scored|No data|Error|Loading\.\.\.)/);
          });
        cy.get('.composite-score')
          .should('have.attr', 'data-tooltip')
          .then((tooltip) => {
            cy.log(`Article[${idx}] CompositeScore tooltip: ${tooltip}`);
            expect(tooltip).to.match(/CompositeScore:/);
          });

        // Confidence indicator
        cy.get('.confidence')
          .should('exist')
          .invoke('text')
          .then((text) => {
            cy.log(`Article[${idx}] Confidence text: ${text}`);
            expect(text).to.match(/Confidence: (\d+%|No data|N\/A|Loading\.\.\.)/);
          });
        cy.get('.confidence')
          .should('have.attr', 'data-tooltip')
          .then((tooltip) => {
            cy.log(`Article[${idx}] Confidence tooltip: ${tooltip}`);
            expect(tooltip).to.match(/Confidence:/);
          });

        // Composite indicator
        cy.get('.composite-indicator')
          .should('exist')
          .should('have.attr', 'data-tooltip')
          .then((tooltip) => {
            cy.log(`Article[${idx}] Composite indicator tooltip: ${tooltip}`);
            expect(tooltip).to.match(/CompositeScore: .*Confidence:/s);
          });
      });
    });
  });

  it('should show error message if bias scoring data fails to load', () => {
    // Simulate error by intercepting the bias data API and forcing a failure
    cy.intercept('GET', /\/api\/articles\/\d+\/bias/, { statusCode: 500 }).as('biasApiError');
    // Reload page to trigger API call
    cy.visit('/');
    // Wait for error message in explanation
    cy.get('.bias-explanation, #bias-explanation')
      .should('contain.text', 'Failed to load bias scoring data. Please try again later.');
    cy.get('.composite-score').should('contain.text', 'Score: Error');
    cy.get('.confidence').should('contain.text', 'Confidence: N/A');
  });
});
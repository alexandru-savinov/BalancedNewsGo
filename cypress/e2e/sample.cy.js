// Basic Cypress E2E test for the BalancedNewsGo UI

describe('BalancedNewsGo UI', () => {
  it('should load the main page', () => {
    cy.visit('/');
    cy.contains('BalancedNews'); // Adjust this selector to match actual UI text
  });

  it('should ensure background scoring job processes new articles', () => {
    cy.log('Scoring Automation: Start background scoring check');

    // Step 1: Trigger refresh to fetch new articles
    cy.request('POST', '/api/refresh').then((refreshResp) => {
      cy.log('Scoring Automation: Triggered /api/refresh');
      // Step 2: Poll /api/articles for a new article with a non-null CompositeScore
      const maxAttempts = 10;
      const pollInterval = 3000; // ms

      function pollForScoredArticle(attempt = 1) {
        cy.log(`Scoring Automation: Polling for scored article (attempt ${attempt})`);
        cy.request('/api/articles').then((resp) => {
          const articles = resp.body;
          // Find an article with a non-null, non-zero CompositeScore
          const scored = articles.find(
            (a) => a.CompositeScore !== null && a.CompositeScore !== undefined && a.CompositeScore !== 0
          );
          if (scored) {
            cy.log(
              `Scoring Automation: Found scored article (id=${scored.ID}, score=${scored.CompositeScore})`
            );
            cy.log('Scoring Automation: Background scoring job completed');
          } else if (attempt < maxAttempts) {
            cy.wait(pollInterval).then(() => pollForScoredArticle(attempt + 1));
          } else {
            throw new Error('Scoring Automation: No scored article found after polling');
          }
        });
      }

      pollForScoredArticle();
    });
  });
});
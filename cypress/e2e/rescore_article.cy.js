describe('Article Rescoring and UI Update via SSE', () => {
  const articleId = 1646; // Target article ID
  const articleSelector = `[data-article-id="${articleId}"]`;
  const scoreButtonSelector = '.score-article-btn'; // Corrected selector based on index.html
  const scoreProgressMessagesSelector = '.score-progress-messages'; // Correct selector for the UL containing messages
  const compositeScoreSelector = '.composite-score';
  const biasLabelSelector = '.bias-label';
  const confidenceSelector = '.confidence';
  const errorMessageSelector = '.score-error-message';
  const scoringTimeout = 60000; // 60 seconds timeout for scoring

  beforeEach(() => {
    // Optional: Intercept initial articles load if needed to ensure article 1646 exists
    // cy.intercept('GET', '/api/articles', { fixture: 'articles_with_1646.json' }).as('getArticles');
    cy.visit('/');
    // cy.wait('@getArticles'); // Wait if intercepting
  });

  it(`should trigger rescore for article ${articleId}, wait for SSE update, and verify UI changes`, () => {
    // 1. Find the specific article and the score button
    cy.get(articleSelector).should('be.visible');
    cy.get(articleSelector).find(scoreButtonSelector).should('be.visible').as('scoreButton');

    // 2. Clear session storage for this article to force fetch, then click
    cy.window().then((win) => {
      win.sessionStorage.removeItem(`bias_data_${articleId}`);
    });
    cy.get('@scoreButton').click();

    // 3. Assert progress UI appears (button might disable, text might change)
    cy.get('@scoreButton').should('be.disabled');
    // Optional: Check for a spinner or specific progress text if available
    // cy.get(articleSelector).find('.spinner').should('be.visible');
    cy.get(articleSelector).find(scoreProgressMessagesSelector).should('not.be.empty'); // Check the correct container

    // 4. Wait for scoring completion message via SSE update observation
    // Find the progress message container within the article and wait for the completion message
    cy.get(articleSelector).find(scoreProgressMessagesSelector, { timeout: scoringTimeout })
      .should('contain.text', 'Scoring complete!'); // Check within the correct container

    // 5. Wait for the button to be re-enabled (indicates process finished)
    cy.get('@scoreButton').should('not.be.disabled');
    // Optional: Check if button text changes to "Rescore article"
    // cy.get('@scoreButton').should('contain.text', 'Rescore article');

    // 6. Assert UI updates reflecting the new score
    cy.get(articleSelector).find(compositeScoreSelector)
      .should('not.contain.text', 'No data')
      .and('not.contain.text', 'Loading...')
      .and('match', /Score: [-+]?\d+\.\d{2}/); // Asserts format like Score: -0.50 or Score: 0.75

    cy.get(articleSelector).find(biasLabelSelector)
      .should('not.contain.text', 'Scoring unavailable')
      .and('not.contain.text', 'Loading...')
      .and('not.be.empty'); // Check it has some text like Center, Left, Right etc.

    cy.get(articleSelector).find(confidenceSelector)
      .should('not.contain.text', 'N/A')
      .and('match', /\d+% Confidence/); // Asserts format like 85% Confidence

    // Ensure our temporary debug message div is hidden or empty now
    cy.get(articleSelector).find(errorMessageSelector).should('not.be.visible');
  });
});
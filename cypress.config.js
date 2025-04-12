const { defineConfig } = require('cypress');

module.exports = defineConfig({
  e2e: {
    baseUrl: 'http://localhost:5173', // Updated for React frontend dev server
    specPattern: 'cypress/e2e/**/*.cy.ts', // Updated to look for .ts files
    supportFile: false
  }
});
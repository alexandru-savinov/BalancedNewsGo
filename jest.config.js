module.exports = {
  testEnvironment: 'jsdom',
  testMatch: [
    '**/web/js/components/ProgressIndicator.test.js'
  ],
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  transformIgnorePatterns: [
    'node_modules/(?!(.*\\.mjs$|@babel/runtime))'
  ]
};

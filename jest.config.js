module.exports = {
  testEnvironment: 'jsdom',
  testMatch: [
    '**/web/js/components/*.test.js',
    '**/web/js/pages/*.test.js',
    '**/web/js/utils/*.test.js',
    '**/tests/frontend/**/*.test.js'
  ],
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  transformIgnorePatterns: [
    'node_modules/(?!(.*\\.mjs$|@babel/runtime))'
  ],
  collectCoverageFrom: [
    'web/js/**/*.js',
    'static/assets/js/**/*.js',
    '!web/js/**/*.test.js',
    '!static/assets/js/vendor/**'
  ],
  testPathIgnorePatterns: [
    '/node_modules/',
    '/coverage/',
    '/test-results/'
  ]
};

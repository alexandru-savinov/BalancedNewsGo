module.exports = {
  testEnvironment: 'jsdom',
  testMatch: [
    '**/web/js/components/*.test.js',
    '**/web/js/pages/*.test.js',
    '**/web/js/utils/*.test.js',
    '**/static/js/components/*.test.js',
    '**/static/js/utils/*.test.js',
    '**/tests/frontend/**/*.test.js'
  ],
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  transformIgnorePatterns: [
    'node_modules/(?!(.*\\.mjs$|@babel/runtime))'
  ],
  collectCoverageFrom: [
    'web/js/**/*.js',
    'static/assets/js/**/*.js',
    'static/js/**/*.js',
    '!web/js/**/*.test.js',
    '!static/assets/js/vendor/**',
    '!static/js/**/*.test.js'
  ],
  testPathIgnorePatterns: [
    '/node_modules/',
    '/coverage/',
    '/test-results/'
  ]
};

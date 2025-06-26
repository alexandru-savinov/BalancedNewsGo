module.exports = {
  ci: {
    collect: {
      url: [
        'http://localhost:8080/articles'
      ],
      // Server is already running, so we don't need to start it
      // startServerCommand: './newsbalancer.exe',
      // startServerReadyPattern: 'Server running on',
      // startServerReadyTimeout: 30000,
      numberOfRuns: 3
    },
    assert: {
      assertions: {
        'categories:performance': ['error', { minScore: 0.9 }],
        'categories:accessibility': ['error', { minScore: 0.9 }],
        'categories:best-practices': ['error', { minScore: 0.9 }],
        'categories:seo': ['error', { minScore: 0.8 }],
        'categories:pwa': 'off'
      }
    },
    upload: {
      target: 'temporary-public-storage'
    }
  }
};

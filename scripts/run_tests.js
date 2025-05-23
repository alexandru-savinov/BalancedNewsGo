const newman = require('newman');

// Simple helper to execute the primary Newman collection used by the other test scripts
newman.run({
    collection: require('./postman/unified_backend_tests.json'),
    environment: require('./postman/local_environment.json'),
    reporters: ['cli']
}, function (err) {
    if (err) { throw err; }
    console.log('Collection run complete!');
});
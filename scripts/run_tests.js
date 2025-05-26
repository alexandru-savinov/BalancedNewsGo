const newman = require('newman');

newman.run({
    collection: require('./postman/backend_fixes_tests_updated.json'),
    environment: require('./postman/local_environment.json'),
    reporters: ['cli']
}, function (err) {
    if (err) { throw err; }
    console.log('Collection run complete!');
});

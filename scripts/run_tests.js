const newman = require('newman');
const path = require('path');

newman.run({
    collection: require(path.resolve(__dirname, '../postman/unified_backend_tests.json')),
    environment: require(path.resolve(__dirname, '../postman/local_environment.json')),
    reporters: ['cli']
}, function (err) {
    if (err) { throw err; }
    console.log('Collection run complete!');
});
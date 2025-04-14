#!/bin/bash

# Run all test cases
echo "Running all test cases"
npx newman run memory-bank/essential_rescoring_tests.json --reporters cli
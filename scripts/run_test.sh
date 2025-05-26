#!/bin/bash

# Run a single test case
echo "Running test for TC4 - Rescore with Upper Boundary Score (1.0)"
npx newman run memory-bank/essential_rescoring_tests.json --folder "TC4 - Rescore with Upper Boundary Score (1.0)" --reporters cli

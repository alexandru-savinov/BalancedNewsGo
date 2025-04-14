@echo off
REM Make sure the backend server is running before executing this script
REM You can start the server with: go run cmd/server/main.go

REM Run only the TC3 test case (Rescore with Invalid Score)
npx newman run memory-bank/postman_rescoring_collection.json --folder "TC3 - Rescore with Invalid Score (Negative)" --reporters cli,json --reporter-json-export test-results/tc3_rescore_invalid_score.json

REM Display the results
echo TC3 test completed. Results saved to test-results/tc3_rescore_invalid_score.json
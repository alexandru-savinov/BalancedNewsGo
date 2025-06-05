@echo off
REM Change directory to the repository root relative to this script so
REM it works from any checkout location.
cd /d %~dp0..
npx newman run memory-bank/postman_rescoring_collection.json --folder "TC3 - Rescore with Invalid Score (Out of Range)" --reporters cli,json --reporter-json-export test-results/tc3_rescore_invalid_score.json
echo TC3 test completed. Results saved to test-results/tc3_rescore_invalid_score.json
pause

@echo off
cd /d C:\Users\user\Documents\dev\news_filter\newbalancer_go
npx newman run memory-bank/postman_rescoring_collection.json --folder "TC3 - Rescore with Invalid Score (Negative)" --reporters cli,json --reporter-json-export test-results/tc3_rescore_invalid_score.json
echo TC3 test completed. Results saved to test-results/tc3_rescore_invalid_score.json
pause
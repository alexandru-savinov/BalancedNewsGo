# APIdog Setup for News Filter API Testing

This directory contains APIdog configuration files for testing the News Filter API. APIdog is a powerful API testing tool that provides an alternative to Postman.

## Files

- `RescoreArticleTest.apidog.json`: OpenAPI 3.0 specification for the News Filter API
- `local_environment.json`: Environment variables for local testing
- `RescoreArticleTestFlow.json`: Test flow for verifying article rescoring functionality

## Getting Started with APIdog

1. Download and install APIdog from [https://apidog.com/](https://apidog.com/)
2. Import the OpenAPI specification file (`RescoreArticleTest.apidog.json`)
3. Import the environment file (`local_environment.json`)
4. Import the test flow file (`RescoreArticleTestFlow.json`)

## Running Tests

1. Make sure your News Filter API server is running at `http://localhost:8080`
2. In APIdog, select the "Local Environment"
3. Run the "Rescore Article Test Flow" to execute the full test sequence

## Test Flow

The test flow performs the following steps:

1. List Articles: Retrieves a list of articles and stores the first article's ID
2. Get Article Before Rescoring: Retrieves the article details before rescoring
3. Trigger Rescoring: Initiates the rescoring process for the article
4. Get Article After Rescoring: Retrieves the article details after rescoring and compares scores

## API Endpoints

The following endpoints are included in the APIdog configuration:

- `GET /api/articles`: List all articles
- `GET /api/articles/{articleId}`: Get a specific article by ID
- `POST /api/articles/{articleId}/rescore`: Trigger rescoring for an article
- `POST /api/llm/reanalyze/{articleId}`: Reanalyze an article using LLM
- `GET /api/llm/score-progress/{articleId}`: Get the progress of a scoring operation

## Troubleshooting

If you encounter issues with the tests:

1. Verify that the API server is running at `http://localhost:8080`
2. Check that there are articles in the database
3. Ensure that the LLM service is available for rescoring operations
4. Review the APIdog console for detailed error messages

For more information on using APIdog, refer to the [official documentation](https://docs.apidog.com/).
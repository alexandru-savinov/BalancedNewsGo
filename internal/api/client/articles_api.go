package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// ArticlesApiService handles article-related API calls
type ArticlesApiService service

// GetArticles fetches a list of articles with optional filtering
func (a *ArticlesApiService) GetArticles(ctx context.Context, params ArticlesParams) ([]Article, error) {
	path := "/articles"

	// Build query parameters
	query := url.Values{}
	if params.Source != "" {
		query.Set("source", params.Source)
	}
	if params.Leaning != "" {
		query.Set("leaning", params.Leaning)
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	resp, err := a.client.makeRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't fail the operation since we're in defer
			// The response body has likely already been read
		}
	}()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	// Convert interface{} to []Article
	if response.Data == nil {
		return []Article{}, nil
	}

	articlesData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var articles []Article
	if err := json.Unmarshal(articlesData, &articles); err != nil {
		return nil, err
	}

	return articles, nil
}

// GetArticle fetches a single article by ID
func (a *ArticlesApiService) GetArticle(ctx context.Context, id int64) (*Article, error) {
	path := fmt.Sprintf("/articles/%d", id)

	resp, err := a.client.makeRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't fail the operation since we're in defer
		}
	}()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Data == nil {
		return nil, fmt.Errorf("no article data returned")
	}

	articleData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var article Article
	if err := json.Unmarshal(articleData, &article); err != nil {
		return nil, err
	}

	return &article, nil
}

// CreateArticle creates a new article
func (a *ArticlesApiService) CreateArticle(ctx context.Context, req CreateArticleRequest) (*CreateArticleResponse, error) {
	path := "/articles"

	resp, err := a.client.makeRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't fail the operation since we're in defer
		}
	}()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Data == nil {
		return nil, fmt.Errorf("no creation data returned")
	}

	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var createResp CreateArticleResponse
	if err := json.Unmarshal(responseData, &createResp); err != nil {
		return nil, err
	}

	return &createResp, nil
}

// GetArticleSummary gets the summary for an article
func (a *ArticlesApiService) GetArticleSummary(ctx context.Context, id int64) (string, error) {
	path := fmt.Sprintf("/articles/%d/summary", id)

	resp, err := a.client.makeRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if response.Data == nil {
		return "", fmt.Errorf("no summary data returned")
	}

	if summary, ok := response.Data.(string); ok {
		return summary, nil
	}

	return fmt.Sprintf("%v", response.Data), nil
}

// GetArticleBias gets the bias analysis for an article
func (a *ArticlesApiService) GetArticleBias(ctx context.Context, id int64) (*ScoreResponse, error) {
	path := fmt.Sprintf("/articles/%d/bias", id)

	resp, err := a.client.makeRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Data == nil {
		return nil, fmt.Errorf("no bias data returned")
	}

	biasData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var scoreResp ScoreResponse
	if err := json.Unmarshal(biasData, &scoreResp); err != nil {
		return nil, err
	}

	return &scoreResp, nil
}

// GetArticleEnsemble gets the ensemble details for an article
func (a *ArticlesApiService) GetArticleEnsemble(ctx context.Context, id int64) (interface{}, error) {
	path := fmt.Sprintf("/articles/%d/ensemble", id)

	resp, err := a.client.makeRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

import apiClient from './apiClient';
import { ArticlesApiResponse, ApiErrorResponse, Article, ArticleDetailApiResponse, ArticleDetailResponseData, LLMScore } from '../types/api.d'; // Import types

interface FetchArticlesParams {
  source?: string;
  leaning?: string;
  limit?: number;
  offset?: number;
}

/**
 * Fetches a list of articles from the API.
 * @param params - Optional query parameters for filtering and pagination.
 * @returns A promise that resolves with the list of articles.
 * @throws Throws an error if the API request fails or returns an error response.
 */
export const fetchArticles = async (params: FetchArticlesParams = {}): Promise<Article[]> => {
  try {
    // Construct query parameters, filtering out undefined values
    const queryParams = Object.entries(params)
      .filter(([, value]) => value !== undefined)
      .reduce((acc, [key, value]) => {
        acc[key] = String(value); // Ensure values are strings for URLSearchParams
        return acc;
      }, {} as Record<string, string>);

    const response = await apiClient.get<ArticlesApiResponse | ApiErrorResponse>('/articles', {
      params: queryParams,
    });

    // Check if the response indicates success and contains the expected data structure
    if (response.data.success && 'data' in response.data && Array.isArray(response.data.data)) {
      return response.data.data;
    } else if (!response.data.success && 'error' in response.data) {
      // Handle structured API error response
      console.error('API Error fetching articles:', response.data.error.message);
      throw new Error(`API Error: ${response.data.error.message} (Code: ${response.data.error.code})`);
    } else {
      // Handle unexpected response structure
      console.error('Unexpected API response structure:', response.data);
      throw new Error('Failed to fetch articles due to unexpected API response format.');
    }
  } catch (error: any) {
    // Handle network errors or errors thrown from the try block
    console.error('Error fetching articles:', error.message || error);
    // Re-throw a more generic error or the specific API error if available
    throw new Error(error.message || 'Failed to fetch articles. Please check the network connection or API status.');
  }
};



/**
 * Fetches detailed information for a single article by its ID.
 * @param id - The ID of the article to fetch.
 * @returns A promise that resolves with the detailed article data.
 * @throws Throws an error if the API request fails or returns an error response.
 */
export const fetchArticleById = async (id: number): Promise<ArticleDetailResponseData> => {
  try {
    const response = await apiClient.get<ArticleDetailApiResponse | ApiErrorResponse>(`/articles/${id}`);

    // Check if the response indicates success and contains the expected data structure
    if (response.data.success && 'data' in response.data && response.data.data.article) {
       // Basic validation of nested structure
       if (response.data.data.scores === undefined || response.data.data.composite_score === undefined || response.data.data.confidence === undefined) {
          console.error('Unexpected API response structure for article detail (missing fields):', response.data.data);
          throw new Error('Failed to fetch article details due to unexpected API response format (missing fields).');
       }
      return response.data.data;
    } else if (!response.data.success && 'error' in response.data) {
      // Handle structured API error response
      console.error(`API Error fetching article ${id}:`, response.data.error.message);
      throw new Error(`API Error: ${response.data.error.message} (Code: ${response.data.error.code})`);
    } else {
      // Handle unexpected response structure
      console.error('Unexpected API response structure for article detail:', response.data);
      throw new Error('Failed to fetch article details due to unexpected API response format.');
    }
  } catch (error: any) {
    // Handle network errors or errors thrown from the try block
    console.error(`Error fetching article ${id}:`, error.message || error);
    // Re-throw a more generic error or the specific API error if available
    throw new Error(error.message || `Failed to fetch article ${id}. Please check the network connection or API status.`);
  }
};



/**
 * Triggers the re-analysis (scoring) of an article by its ID.
 * @param id - The ID of the article to re-analyze.
 * @returns A promise that resolves on success.
 * @throws Throws an error if the API request fails or returns an error response.
 */
export const triggerArticleRescore = async (id: number): Promise<void> => {
  try {
    // Assuming a simple success/error response structure for this endpoint
    // The backend might return { success: true } or an ApiErrorResponse
    const response = await apiClient.post<{ success: boolean } | ApiErrorResponse>(`/llm/reanalyze/${id}`);

    // Check if 'success' property exists and is true
    if ('success' in response.data && response.data.success) {
      // Successfully triggered re-analysis
      return;
    } else if ('error' in response.data) {
      // Handle structured API error response
      console.error(`API Error triggering rescore for article ${id}:`, response.data.error.message);
      throw new Error(`API Error: ${response.data.error.message} (Code: ${response.data.error.code})`);
    } else {
      // Handle unexpected response structure (e.g., neither success nor error field)
      console.error('Unexpected API response structure for rescore trigger:', response.data);
      throw new Error('Failed to trigger article rescore due to unexpected API response format.');
    }
  } catch (error: any) {
    // Handle network errors or errors thrown from the try block (e.g., axios errors)
    console.error(`Error triggering rescore for article ${id}:`, error.message || error);
    // Check if it's an API error structure nested within the catch
     if (error.response && error.response.data && 'error' in error.response.data) {
         const apiError = error.response.data.error;
         throw new Error(`API Error: ${apiError.message} (Code: ${apiError.code})`);
     }
    // Re-throw a more generic error or the specific API error if available
    throw new Error(error.message || `Failed to trigger rescore for article ${id}. Please check the network connection or API status.`);
  }
};


// Add other article-related API functions here later, e.g.:
// export const fetchArticleById = async (id: number): Promise<ArticleDetail> => { ... };
// export const triggerArticleRescore = async (id: number): Promise<void> => { ... };
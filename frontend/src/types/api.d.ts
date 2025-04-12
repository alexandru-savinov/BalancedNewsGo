// Define interfaces for API data structures

export interface Article {
  ID: number;
  Title: string;
  URL: string;
  Source: string;
  Content: string; // Used as summary in the current frontend
  CreatedAt: string; // ISO 8601 date string
  UpdatedAt: string; // ISO 8601 date string
  image_url?: string; // Optional image URL
  image_caption?: string; // Optional image caption
  CompositeScore?: number | null; // Optional composite score (might be null)
  // Add other fields as needed based on API responses
}

// Interface for the successful API response structure from /api/articles
export interface ArticlesApiResponse {
  success: boolean;
  data: Article[];
  // Add pagination or other metadata fields if the API includes them
  // total?: number;
  // limit?: number;
  // offset?: number;
}

// Interface for error responses (adjust as needed based on actual API errors)
export interface ApiErrorResponse {
  success: boolean;
  error: {
    code: string;
    message: string;
  };
}

// Add other API response types here (e.g., for feedback, bias details)


// Interface for individual LLM scores (based on api.go)
export interface LLMScore {
  ID: number;
  ArticleID: number;
  Model: string;
  Score: number;
  Metadata: string; // Often JSON stringified data
  CreatedAt: string; // ISO 8601 date string
  // Add Confidence if it's part of the individual score, otherwise it's likely calculated/aggregated
}

// Interface for the detailed data structure within the /api/articles/:id response
export interface ArticleDetailResponseData {
  article: Article;
  scores: LLMScore[];
  composite_score: number | null;
  confidence: number | null;
}

// Interface for the successful API response structure from /api/articles/:id
export interface ArticleDetailApiResponse {
  success: boolean;
  data: ArticleDetailResponseData;
}

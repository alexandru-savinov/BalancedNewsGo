import apiClient from './apiClient';
import { ApiErrorResponse } from '../types/api.d'; // Import common error type

// Define the structure of the feedback data to be sent to the API
export interface FeedbackPayload {
  article_id: number;
  user_comment?: string;
  rating?: number; // e.g., 1-5 stars
  tags?: string[]; // e.g., ["Parse Error", "Model Disagreement"]
  // Add other relevant fields as needed based on backend API definition
}

// Define the expected success response structure (adjust if needed)
interface FeedbackSuccessResponse {
  success: true;
  message: string;
  feedback_id?: number; // Optional: if the backend returns the ID of the created feedback
}

/**
 * Submits user feedback for a specific article to the backend API.
 * @param payload - The feedback data to submit.
 * @returns A promise that resolves with the success response data.
 * @throws Throws an error if the API request fails or returns an error response.
 */
export const submitFeedback = async (payload: FeedbackPayload): Promise<FeedbackSuccessResponse> => {
  try {
    const response = await apiClient.post<FeedbackSuccessResponse | ApiErrorResponse>('/feedback', payload);

    // Check for success response
    if (response.data.success && 'message' in response.data) {
      return response.data;
    } else if (!response.data.success && 'error' in response.data) {
      // Handle structured API error response
      console.error(`API Error submitting feedback for article ${payload.article_id}:`, response.data.error.message);
      throw new Error(`API Error: ${response.data.error.message} (Code: ${response.data.error.code})`);
    } else {
      // Handle unexpected response structure
      console.error('Unexpected API response structure for feedback submission:', response.data);
      throw new Error('Failed to submit feedback due to unexpected API response format.');
    }
  } catch (error: any) {
    // Handle network errors or errors thrown from the try block
    console.error(`Error submitting feedback for article ${payload.article_id}:`, error.message || error);
     // Check if it's an API error structure nested within the catch
     if (error.response && error.response.data && 'error' in error.response.data) {
         const apiError = error.response.data.error;
         throw new Error(`API Error: ${apiError.message} (Code: ${apiError.code})`);
     }
    // Re-throw a more generic error
    throw new Error(error.message || `Failed to submit feedback for article ${payload.article_id}. Please check the network connection or API status.`);
  }
};
import { API_BASE_URL } from './apiClient'; // Assuming apiClient exports the base URL

interface SseCallbacks {
  onOpen?: () => void;
  onProgress?: (data: any) => void; // Define a more specific type later if needed
  onComplete?: (data: any) => void; // Define a more specific type later if needed
  onError?: (error: Event | string) => void;
  onClose?: () => void; // Callback when connection is explicitly closed or fails definitively
}

/**
 * Connects to the score progress SSE endpoint for a given article ID.
 * Manages the EventSource lifecycle and calls provided callbacks.
 *
 * @param articleId The ID of the article to get score progress for.
 * @param callbacks An object containing callback functions for different SSE events.
 * @returns A function to manually close the SSE connection.
 */
export const connectToScoreProgress = (articleId: number, callbacks: SseCallbacks): (() => void) => {
  const url = `${API_BASE_URL}/llm/score-progress/${articleId}`;
  console.log(`SSE: Connecting to ${url}`);
  const eventSource = new EventSource(url);
  let isClosed = false; // Flag to prevent multiple close calls

  const closeConnection = () => {
    if (!isClosed && eventSource && eventSource.readyState !== EventSource.CLOSED) {
      console.log(`SSE: Closing connection to ${url}`);
      eventSource.close();
      isClosed = true;
      callbacks.onClose?.(); // Notify that the connection is closed
    }
  };

  eventSource.onopen = () => {
    console.log(`SSE: Connection opened to ${url}`);
    callbacks.onOpen?.();
  };

  // Generic message handler (can be customized with specific event names if backend uses them)
  eventSource.onmessage = (event) => {
    console.log('SSE: Message received:', event.data);
    try {
      const data = JSON.parse(event.data);
      // TODO: Differentiate between progress and completion messages based on data structure
      // For now, assume all messages are progress unless a specific 'type' or 'status' field indicates completion
      if (data.status === 'complete') { // Example completion check
         callbacks.onComplete?.(data);
         closeConnection(); // Close connection on completion message
      } else {
         callbacks.onProgress?.(data);
      }
    } catch (error) {
      console.error('SSE: Error parsing message data:', error);
      callbacks.onError?.('Error parsing message data');
      // Decide if we should close connection on parse error
      // closeConnection();
    }
  };

  eventSource.onerror = (error) => {
    console.error(`SSE: Error with connection to ${url}:`, error);
    callbacks.onError?.(error);
    // Don't close immediately on error, EventSource might retry.
    // Close only if readyState becomes CLOSED.
    if (eventSource.readyState === EventSource.CLOSED) {
        console.log(`SSE: Connection definitively closed due to error for ${url}`);
        if (!isClosed) { // Ensure onClose is called only once
            isClosed = true;
            callbacks.onClose?.();
        }
    }
  };

  // Return the function to allow manual closing
  return closeConnection;
};
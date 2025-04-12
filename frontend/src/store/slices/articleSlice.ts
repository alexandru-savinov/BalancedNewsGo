import { StateCreator } from 'zustand';
import { Article, ArticleDetailResponseData } from '../../types/api.d'; // Import types
import { fetchArticles, fetchArticleById } from '../../services/articleService'; // Import service functions
import { connectToScoreProgress } from '../../services/sseService'; // Import SSE service

// Define the shape of the Article slice state
export interface ArticleSliceState {
  articles: Article[];
  isLoading: boolean; // Loading state for the list
  error: string | null; // Error state for the list
  selectedArticle: ArticleDetailResponseData | null; // State for the detailed view
  isLoadingDetail: boolean; // Loading state for the detail view
  errorDetail: string | null; // Error state for the detail view
  // SSE State for Score Progress
  sseConnectionStatus: 'idle' | 'connecting' | 'open' | 'closed' | 'error';
  sseProgressData: any | null; // Store latest progress message data
  sseError: string | null; // Store SSE specific errors
  sseCloseFunction: (() => void) | null; // Function to close the current SSE connection
  // Actions
  fetchArticles: (params?: { source?: string; leaning?: string; limit?: number; offset?: number }) => Promise<void>;
  fetchArticleDetail: (id: number) => Promise<void>; // Action for fetching details
  connectScoreProgressSse: (articleId: number) => void; // Action to connect SSE
  disconnectScoreProgressSse: () => void; // Action to disconnect SSE
}

// Create the slice using StateCreator for potential middleware compatibility
export const createArticleSlice: StateCreator<ArticleSliceState, [], [], ArticleSliceState> = (set, get) => ({
  articles: [],
  isLoading: false,
  error: null,
  selectedArticle: null,
  isLoadingDetail: false,
  errorDetail: null,
  // SSE State Init
  sseConnectionStatus: 'idle',
  sseProgressData: null,
  sseError: null,
  sseCloseFunction: null,

  // Action to fetch articles
  fetchArticles: async (params = {}) => {
    // Check if already loading to prevent concurrent fetches
    if (get().isLoading) {
      console.warn('Already fetching articles.');
      return;
    }

    console.log('[articleSlice] Proceeding with fetchArticles...');
    set({ isLoading: true, error: null }); // Set loading state, clear previous error
    try {
      console.log('[articleSlice] Calling fetchArticles service with params:', params);
      const fetchedArticles = await fetchArticles(params); // Call the API service
console.log('articleSlice: Updating state with fetched articles:', fetchedArticles); // DEBUG LOG
      console.log('[articleSlice] Received articles from service:', fetchedArticles);

      set({ articles: fetchedArticles, isLoading: false }); // Update state on success
      console.log('[articleSlice] Setting state: isLoading=false, articles=', fetchedArticles);
    } catch (err: any) {
      console.error('Error in fetchArticles action:', err);
      set({ error: err.message || 'Failed to fetch articles', isLoading: false }); // Update state on error
    }
  },

  // Action to fetch article details
  fetchArticleDetail: async (id: number) => {
    // Check if already loading details for this ID or another ID
    if (get().isLoadingDetail) {
      console.warn('Already fetching article details.');
      return;
    }
    // Clear previous selection and error when starting a new fetch
    set({ isLoadingDetail: true, errorDetail: null, selectedArticle: null });
    try {
      const detailedArticle = await fetchArticleById(id); // Call the API service
      set({ selectedArticle: detailedArticle, isLoadingDetail: false }); // Update state on success
    } catch (err: any) {
      console.error(`Error in fetchArticleDetail action for ID ${id}:`, err);
      set({ errorDetail: err.message || `Failed to fetch details for article ${id}`, isLoadingDetail: false }); // Update state on error
    }
  },

  // Action to connect to Score Progress SSE
  connectScoreProgressSse: (articleId: number) => {
    // Prevent multiple connections if one is already open or connecting
    const currentStatus = get().sseConnectionStatus;
    if (currentStatus === 'open' || currentStatus === 'connecting') {
      console.warn(`SSE: Connection status is already ${currentStatus}. Aborting new connection.`);
      // Optionally disconnect previous one first: get().disconnectScoreProgressSse();
      return;
    }

    // Disconnect any previous connection cleanly before starting a new one
    get().disconnectScoreProgressSse();

    set({
      sseConnectionStatus: 'connecting',
      sseProgressData: null, // Clear previous data
      sseError: null,        // Clear previous error
      sseCloseFunction: null // Clear previous close function
    });

    const closeSse = connectToScoreProgress(articleId, {
      onOpen: () => {
        console.log('SSE Store: Connection opened');
        set({ sseConnectionStatus: 'open', sseError: null });
      },
      onProgress: (data) => {
        console.log('SSE Store: Progress received', data);
        set({ sseProgressData: data, sseConnectionStatus: 'open' }); // Keep status open on progress
      },
      onComplete: (data) => {
        console.log('SSE Store: Completion message received', data);
        // Optionally update state with completion data if needed
        set({ sseProgressData: data, sseConnectionStatus: 'closed', sseCloseFunction: null }); // Status closed on completion
      },
      onError: (error) => {
        console.error('SSE Store: Error received', error);
        const errorMessage = typeof error === 'string' ? error : 'SSE connection error occurred';
        set({ sseError: errorMessage, sseConnectionStatus: 'error', sseCloseFunction: null }); // Set error status
      },
      onClose: () => {
        console.log('SSE Store: Connection closed');
        // Only update status if it wasn't already set to 'closed' or 'error' by onComplete/onError
        if (get().sseConnectionStatus !== 'closed' && get().sseConnectionStatus !== 'error') {
            set({ sseConnectionStatus: 'closed', sseCloseFunction: null });
        }
      }
    });

    // Store the function to close this specific connection
    set({ sseCloseFunction: closeSse });
  },

  // Action to manually disconnect the SSE
  disconnectScoreProgressSse: () => {
    const closeFunc = get().sseCloseFunction;
    if (closeFunc) {
      console.log('SSE Store: Manually disconnecting...');
      closeFunc(); // Call the close function returned by connectToScoreProgress
      set({ sseCloseFunction: null, sseConnectionStatus: 'closed' }); // Clear the function and set status
    } else {
       // If no close function, ensure status is idle or closed
       if (get().sseConnectionStatus !== 'idle' && get().sseConnectionStatus !== 'closed') {
           set({ sseConnectionStatus: 'closed' });
       }
    }
    // Optionally clear progress data on disconnect: set({ sseProgressData: null, sseError: null });
  },

});

// Note: This slice needs to be combined with other slices in the main store
// or used independently via its own hook if preferred.
// Example of combining (in store/index.ts):
// export const useStore = create<ArticleSliceState & OtherSliceState>()(devtools((...a) => ({
//   ...createArticleSlice(...a),
//   ...createOtherSlice(...a),
// })));
// Or create a dedicated hook:
// import { create } from 'zustand';
// export const useArticleStore = create(createArticleSlice);
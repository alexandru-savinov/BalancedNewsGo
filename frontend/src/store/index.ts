import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { ArticleSliceState, createArticleSlice } from './slices/articleSlice'; // Import slice

// Define a basic structure for the store state
// This will be expanded with slices later
// Combine slice states into the main AppState interface
interface AppState extends ArticleSliceState {
  // Add other slice states here using '&' if needed
  // e.g., & UISliceState
}

// Create the store
// We use devtools for easier debugging in browser extensions
export const useStore = create<AppState>()(
  devtools(
    // Combine slice creators
    (...a) => ({
      ...createArticleSlice(...a),
      // Add other slice creators here
      // ...createUISlice(...a),
    }),
    { name: 'NewsBalancerStore' } // Name for the devtools
  )
);

// Export the store hook for use in components
export default useStore;
import axios from 'axios';

// Determine the base URL for the API
// Default to localhost:8080 which is common for local Go development
// and matches the Cypress testing context.
// This could be made configurable via environment variables later.
const API_BASE_URL = import.meta.env.VITE_REACT_APP_API_URL || 'http://localhost:8080/api';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
    // Add other default headers if needed, e.g., for authentication later
  },
  // Add other Axios configurations if needed, e.g., timeout
  timeout: 10000, // 10 seconds
});

// Optional: Add interceptors for request/response handling
// apiClient.interceptors.request.use(config => {
//   // e.g., add auth token
//   return config;
// }, error => {
//   return Promise.reject(error);
// });

// apiClient.interceptors.response.use(response => {
//   // e.g., handle successful responses
//   return response;
// }, error => {
//   // e.g., handle global errors (like 401 Unauthorized)
//   console.error('API Error:', error.response || error.message);
//   return Promise.reject(error);
// });

export { API_BASE_URL }; // Export the base URL constant
export default apiClient;
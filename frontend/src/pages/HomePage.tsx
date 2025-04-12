import React, { useEffect } from 'react';
import ArticleList from '../features/articles/ArticleList'; // Import ArticleList
import useStore from '../store'; // Corrected import

const HomePage: React.FC = () => {
  const fetchArticles = useStore((state) => state.fetchArticles); // Moved hook usage

  useEffect(() => {
    console.log('[HomePage] useEffect triggered. Attempting to fetch articles...');
    fetchArticles();
  }, [fetchArticles]); // Moved useEffect

  return (
    <div data-testid="home-page-container">
      <h2>Home Page / Dashboard</h2>
      {/* Render the ArticleList component */}
      <ArticleList />
    </div>
  );
};

export default HomePage;
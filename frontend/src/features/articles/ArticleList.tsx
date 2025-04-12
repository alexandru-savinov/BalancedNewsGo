import React, { useEffect } from 'react';
import useStore from '../../store'; // Import the main store hook
import ArticleCard from './ArticleCard'; // Import the card component

const ArticleList: React.FC = () => {
  // Select state and actions from the Zustand store
  const articles = useStore((state) => state.articles);
  const isLoading = useStore((state) => state.isLoading);
  const error = useStore((state) => state.error);
  const fetchArticles = useStore((state) => state.fetchArticles);

  // Fetch articles when the component mounts
  useEffect(() => {
    fetchArticles(); // Call the action from the store slice
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchArticles]); // Dependency array includes fetchArticles

  console.log('ArticleList State:', { isLoading, error, articles }); // DEBUG LOG

  // Wrap all conditional rendering in a single root element
  console.log('[ArticleList] State JUST before return:', { isLoading, error, articlesLength: articles?.length });

  return (
    <div data-testid="article-list-container">
      {/* <div data-testid="static-debug-marker">DEBUG MARKER</div> */}

      {/* Render loading state */}
      {isLoading && (
        <>
          {console.log('[ArticleList] Rendering: Loading state')}
          <p>Loading articles...</p>
        </>
      )}

      {/* Render error state */}
      {error && !isLoading && (
        <>
          {console.log('[ArticleList] Rendering: Error state -', error)}
          <p style={{ color: 'red' }}>Error loading articles: {error}</p>
        </>
      )}

      {/* Render empty state */}
      {!isLoading && !error && (!articles || articles.length === 0) && (
        <>
          {console.log('[ArticleList] Rendering: No articles found state')}
          <p>No articles found.</p>
        </>
      )}

      {/* Render the list of articles */}
      {!isLoading && !error && articles && articles.length > 0 && (
        <>
          {console.log('ArticleList: Articles received from store:', articles)}
          {console.log('[ArticleList] Rendering: Mapping articles, count:', articles.length)}
          {console.log('[Debug] ArticleList: State before render. Count:', articles.length, 'isLoading:', isLoading)}
          {/* Removed data-testid from section */}
          <section aria-label="List of news articles">
            {articles.map((article) => (
              <React.Fragment key={article.ID}>
                {/* <div data-testid={`static-map-marker-${article.ID}`}>MAP MARKER {article.ID}</div> */}
                <ArticleCard article={article} />
              </React.Fragment>
            ))}
          </section>
        </>
      )}
    </div>
  );
};

export default ArticleList;
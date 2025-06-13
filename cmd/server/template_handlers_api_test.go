package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apiclient "github.com/alexandru-savinov/BalancedNewsGo/internal/api/wrapper"
)

// MockAPIClient mocks the API client for testing template handlers
type MockAPIClient struct {
	mock.Mock
}

func (m *MockAPIClient) GetArticles(ctx context.Context, params apiclient.ArticlesParams) ([]apiclient.Article, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]apiclient.Article), args.Error(1)
}

func (m *MockAPIClient) GetArticle(ctx context.Context, id int64) (*apiclient.Article, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apiclient.Article), args.Error(1)
}

func (m *MockAPIClient) GetArticleSummary(ctx context.Context, id int64) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockAPIClient) GetArticleBias(ctx context.Context, id int64) (*apiclient.ScoreResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apiclient.ScoreResponse), args.Error(1)
}

func (m *MockAPIClient) GetFeedHealth(ctx context.Context) (map[string]bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockAPIClient) GetCacheStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// Test helper to create handlers with mock API client
func newTestAPITemplateHandlers(mockClient *MockAPIClient) *APITemplateHandlers {
	return &APITemplateHandlers{
		client: mockClient,
	}
}

// Test helper to setup Gin with templates
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up template functions
	router.SetFuncMap(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"mul":   func(a, b float64) float64 { return a * b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
		"date":  func(t time.Time, layout string) string { return t.Format(layout) },
	})

	// For testing, we'll use a simple template pattern
	router.SetHTMLTemplate(template.Must(template.New("test").Parse(`
		{{define "articles.html"}}
		<html>
		<head><title>Articles</title></head>
		<body>
			{{if .Error}}
				<div class="error">{{.Error}}</div>
			{{else}}
				<div class="articles">
					{{range .Articles}}
						<article data-id="{{.ArticleID}}">
							<h2>{{.Title}}</h2>
							<p>{{.Content}}</p>
							<span class="source">{{.Source}}</span>
							<span class="score">{{.CompositeScore}}</span>
						</article>
					{{end}}
				</div>
				<nav class="pagination">
					{{range .Pages}}
						<a href="?page={{.}}" {{if eq . $.CurrentPage}}class="current"{{end}}>{{.}}</a>
					{{end}}
				</nav>
			{{end}}
		</body>
		</html>
		{{end}}

		{{define "article.html"}}
		<html>
		<head><title>{{.Article.Title}}</title></head>
		<body>
			{{if .Error}}
				<div class="error">{{.Error}}</div>
			{{else}}
				<article data-id="{{.Article.ArticleID}}">
					<h1>{{.Article.Title}}</h1>
					<div class="meta">
						<span class="source">{{.Article.Source}}</span>
						<span class="date">{{.Article.PubDate | date "2006-01-02"}}</span>
						<span class="score">{{.Article.CompositeScore}}</span>
					</div>
					<div class="content">{{.Article.Content}}</div>
					{{if .Summary}}
						<div class="summary">
							<h3>Summary</h3>
							<p>{{.Summary}}</p>
						</div>
					{{end}}
				</article>
			{{end}}
		</body>
		</html>
		{{end}}

		{{define "admin.html"}}
		<html>
		<head><title>Admin Dashboard</title></head>
		<body>
			{{if .Error}}
				<div class="error">{{.Error}}</div>
			{{else}}
				<div class="dashboard">
					<h1>Admin Dashboard</h1>
					<div class="stats">
						{{range $key, $value := .CacheStats}}
							<div class="stat">{{$key}}: {{$value}}</div>
						{{end}}
					</div>
					<div class="feed-health">
						{{range $feed, $healthy := .FeedHealth}}
							<div class="feed {{if $healthy}}healthy{{else}}unhealthy{{end}}">
								{{$feed}}: {{if $healthy}}OK{{else}}ERROR{{end}}
							</div>
						{{end}}
					</div>
				</div>
			{{end}}
		</body>
		</html>
		{{end}}
	`)))

	return router
}

// TestTemplateIndexHandler tests the articles listing handler
func TestTemplateIndexHandler(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*MockAPIClient)
		expectedStatus int
		checkResponse  func(*testing.T, string)
	}{
		{
			name:        "Success - Default parameters",
			queryParams: "",
			setupMock: func(m *MockAPIClient) {
				m.On("GetArticles", mock.Anything, mock.MatchedBy(func(params apiclient.ArticlesParams) bool {
					return params.Limit == 20 && params.Offset == 0
				})).Return([]apiclient.Article{
					{
						ArticleID:      1,
						Title:          "Test Article 1",
						Content:        "Test content 1",
						Source:         "test-source",
						CompositeScore: 0.75,
					},
					{
						ArticleID:      2,
						Title:          "Test Article 2",
						Content:        "Test content 2",
						Source:         "test-source",
						CompositeScore: -0.25,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Test Article 1")
				assert.Contains(t, body, "Test Article 2")
				assert.Contains(t, body, "test-source")
				assert.Contains(t, body, "0.75")
				assert.Contains(t, body, "-0.25")
				assert.Contains(t, body, "data-id=\"1\"")
				assert.Contains(t, body, "data-id=\"2\"")
			},
		},
		{
			name:        "Success - With filters",
			queryParams: "?source=cnn&bias=left&page=2",
			setupMock: func(m *MockAPIClient) {
				m.On("GetArticles", mock.Anything, mock.MatchedBy(func(params apiclient.ArticlesParams) bool {
					return params.Source == "cnn" && params.Leaning == "left" && params.Limit == 20 && params.Offset == 20
				})).Return([]apiclient.Article{
					{
						ArticleID: 3,
						Title:     "Filtered Article",
						Source:    "cnn",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Filtered Article")
				assert.Contains(t, body, "cnn")
			},
		},
		{
			name:        "API Error",
			queryParams: "",
			setupMock: func(m *MockAPIClient) {
				m.On("GetArticles", mock.Anything, mock.Anything).Return(nil, errors.New("API connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Error fetching articles")
				assert.Contains(t, body, "API connection failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockAPIClient)
			handlers := newTestAPITemplateHandlers(mockClient)
			router := setupTestRouter()

			tt.setupMock(mockClient)

			router.GET("/", handlers.TemplateIndexHandler())

			req := httptest.NewRequest("GET", "/"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w.Body.String())

			mockClient.AssertExpectations(t)
		})
	}
}

// TestTemplateArticleHandler tests the individual article handler
func TestTemplateArticleHandler(t *testing.T) {
	tests := []struct {
		name           string
		articleID      string
		setupMock      func(*MockAPIClient)
		expectedStatus int
		checkResponse  func(*testing.T, string)
	}{
		{
			name:      "Success - Article with summary",
			articleID: "123",
			setupMock: func(m *MockAPIClient) {
				article := &apiclient.Article{
					ArticleID:      123,
					Title:          "Test Article",
					Content:        "Test article content",
					Source:         "test-source",
					CompositeScore: 0.5,
					PubDate:        time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
				}
				m.On("GetArticle", mock.Anything, int64(123)).Return(article, nil)
				m.On("GetArticleSummary", mock.Anything, int64(123)).Return("This is a test summary", nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Test Article")
				assert.Contains(t, body, "Test article content")
				assert.Contains(t, body, "test-source")
				assert.Contains(t, body, "0.5")
				assert.Contains(t, body, "2023-12-25")
				assert.Contains(t, body, "This is a test summary")
				assert.Contains(t, body, "data-id=\"123\"")
			},
		},
		{
			name:      "Success - Article without summary",
			articleID: "456",
			setupMock: func(m *MockAPIClient) {
				article := &apiclient.Article{
					ArticleID: 456,
					Title:     "Article No Summary",
					Content:   "Content without summary",
				}
				m.On("GetArticle", mock.Anything, int64(456)).Return(article, nil)
				m.On("GetArticleSummary", mock.Anything, int64(456)).Return("", errors.New("summary not found"))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Article No Summary")
				assert.Contains(t, body, "Content without summary")
				assert.NotContains(t, body, "Summary") // Summary section should not appear
			},
		},
		{
			name:      "Invalid article ID",
			articleID: "invalid",
			setupMock: func(m *MockAPIClient) {
				// No mock setup needed as parsing should fail
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid article ID")
			},
		},
		{
			name:      "Article not found",
			articleID: "999",
			setupMock: func(m *MockAPIClient) {
				m.On("GetArticle", mock.Anything, int64(999)).Return(nil, errors.New("article not found"))
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Article not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockAPIClient)
			handlers := newTestAPITemplateHandlers(mockClient)
			router := setupTestRouter()

			tt.setupMock(mockClient)

			router.GET("/article/:id", handlers.TemplateArticleHandler())

			req := httptest.NewRequest("GET", "/article/"+tt.articleID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w.Body.String())

			mockClient.AssertExpectations(t)
		})
	}
}

// TestTemplateAdminHandler tests the admin dashboard handler
func TestTemplateAdminHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockAPIClient)
		expectedStatus int
		checkResponse  func(*testing.T, string)
	}{
		{
			name: "Success - Full dashboard data",
			setupMock: func(m *MockAPIClient) {
				cacheStats := map[string]interface{}{
					"hits":   150,
					"misses": 25,
					"size":   75,
				}
				feedHealth := map[string]bool{
					"cnn":      true,
					"bbc":      true,
					"reuters":  false,
					"guardian": true,
				}
				m.On("GetCacheStats").Return(cacheStats)
				m.On("GetFeedHealth", mock.Anything).Return(feedHealth, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Admin Dashboard")
				assert.Contains(t, body, "hits: 150")
				assert.Contains(t, body, "misses: 25")
				assert.Contains(t, body, "size: 75")
				assert.Contains(t, body, "cnn: OK")
				assert.Contains(t, body, "bbc: OK")
				assert.Contains(t, body, "reuters: ERROR")
				assert.Contains(t, body, "guardian: OK")
				assert.Contains(t, body, "class=\"feed healthy\"")
				assert.Contains(t, body, "class=\"feed unhealthy\"")
			},
		},
		{
			name: "Feed health error",
			setupMock: func(m *MockAPIClient) {
				cacheStats := map[string]interface{}{
					"hits": 100,
				}
				m.On("GetCacheStats").Return(cacheStats)
				m.On("GetFeedHealth", mock.Anything).Return(nil, errors.New("feed health check failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Error fetching dashboard data")
				assert.Contains(t, body, "feed health check failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockAPIClient)
			handlers := newTestAPITemplateHandlers(mockClient)
			router := setupTestRouter()

			tt.setupMock(mockClient)

			router.GET("/admin", handlers.TemplateAdminHandler())

			req := httptest.NewRequest("GET", "/admin", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w.Body.String())

			mockClient.AssertExpectations(t)
		})
	}
}

// TestTemplateHandlers_ContextTimeout tests context timeout handling
func TestTemplateHandlers_ContextTimeout(t *testing.T) {
	mockClient := new(MockAPIClient)
	handlers := newTestAPITemplateHandlers(mockClient)
	router := setupTestRouter()

	// Mock will simulate a slow response that times out
	mockClient.On("GetArticles", mock.Anything, mock.Anything).Return(nil, context.DeadlineExceeded)

	router.GET("/", handlers.TemplateIndexHandler())

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Error fetching articles")

	mockClient.AssertExpectations(t)
}

// TestTemplateHandlers_PaginationLogic tests pagination calculations
func TestTemplateHandlers_PaginationLogic(t *testing.T) {
	tests := []struct {
		name       string
		page       string
		articles   []apiclient.Article
		checkPages func(*testing.T, string)
	}{
		{
			name: "Page 1 with full results",
			page: "1",
			articles: func() []apiclient.Article {
				articles := make([]apiclient.Article, 20) // Full page
				for i := 0; i < 20; i++ {
					articles[i] = apiclient.Article{
						ArticleID: int64(i + 1),
						Title:     "Article " + string(rune(i+1)),
					}
				}
				return articles
			}(),
			checkPages: func(t *testing.T, body string) {
				assert.Contains(t, body, "href=\"?page=1\"")
				assert.Contains(t, body, "href=\"?page=2\"")
				assert.Contains(t, body, "class=\"current\"") // Current page marker
			},
		},
		{
			name: "Page 3 with partial results",
			page: "3",
			articles: []apiclient.Article{
				{ArticleID: 1, Title: "Last Article"},
			},
			checkPages: func(t *testing.T, body string) {
				assert.Contains(t, body, "href=\"?page=1\"")
				assert.Contains(t, body, "href=\"?page=2\"")
				assert.Contains(t, body, "href=\"?page=3\"")
				// Should show current page as active
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockAPIClient)
			handlers := newTestAPITemplateHandlers(mockClient)
			router := setupTestRouter()

			mockClient.On("GetArticles", mock.Anything, mock.Anything).Return(tt.articles, nil)

			router.GET("/", handlers.TemplateIndexHandler())

			req := httptest.NewRequest("GET", "/?page="+tt.page, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			tt.checkPages(t, w.Body.String())

			mockClient.AssertExpectations(t)
		})
	}
}

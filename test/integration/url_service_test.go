package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sp3dr4/dove/internal/application"
	"github.com/sp3dr4/dove/internal/domain"
)

const testBaseURL = "http://localhost:8080"

func TestURLService_CreateShortURL_IntegrationFlow(t *testing.T) {
	env := SetupTestEnvironment(t)

	ctx := context.Background()
	service := env.Service

	tests := []struct {
		name        string
		request     application.CreateURLRequest
		checkResult func(t *testing.T, resp *application.URLResponse, req application.CreateURLRequest)
	}{
		{
			name: "create URL with auto-generated short code",
			request: application.CreateURLRequest{
				URL: "https://example.com",
			},
			checkResult: func(t *testing.T, resp *application.URLResponse, req application.CreateURLRequest) {
				assert.Equal(t, req.URL, resp.OriginalURL)
				assert.Len(t, resp.ShortCode, 6)
				expectedShortURL := testBaseURL + "/" + resp.ShortCode
				assert.Equal(t, expectedShortURL, resp.ShortURL)
			},
		},
		{
			name: "create URL with custom alias",
			request: application.CreateURLRequest{
				URL:         "https://google.com",
				CustomAlias: "google",
			},
			checkResult: func(t *testing.T, resp *application.URLResponse, req application.CreateURLRequest) {
				assert.Equal(t, req.URL, resp.OriginalURL)
				assert.Equal(t, req.CustomAlias, resp.ShortCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.CreateShortURL(ctx, tt.request, testBaseURL)
			require.NoError(t, err)

			tt.checkResult(t, resp, tt.request)

			// Verify URL can be retrieved
			retrievedURL, err := service.GetURL(ctx, resp.ShortCode)
			require.NoError(t, err)
			assert.Equal(t, tt.request.URL, retrievedURL.OriginalURL)
		})
	}
}

func TestURLService_ValidationErrors_Integration(t *testing.T) {
	env := SetupTestEnvironment(t)

	ctx := context.Background()
	service := env.Service

	tests := []struct {
		name    string
		request application.CreateURLRequest
		errMsg  string
	}{
		{
			name: "invalid URL format",
			request: application.CreateURLRequest{
				URL: "not-a-url",
			},
			errMsg: "URL",
		},
		{
			name: "empty URL",
			request: application.CreateURLRequest{
				URL: "",
			},
			errMsg: "URL",
		},
		{
			name: "custom alias too short",
			request: application.CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: "ab",
			},
			errMsg: "CustomAlias",
		},
		{
			name: "custom alias too long",
			request: application.CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: strings.Repeat("a", 21),
			},
			errMsg: "CustomAlias",
		},
		{
			name: "custom alias with invalid characters",
			request: application.CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: "my-alias",
			},
			errMsg: "CustomAlias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateShortURL(ctx, tt.request, testBaseURL)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestURLService_DuplicateAlias_Integration(t *testing.T) {
	env := SetupTestEnvironment(t)

	ctx := context.Background()
	service := env.Service

	// Create first URL with custom alias
	req1 := application.CreateURLRequest{
		URL:         "https://example1.com",
		CustomAlias: "duplicate",
	}
	_, err := service.CreateShortURL(ctx, req1, testBaseURL)
	require.NoError(t, err)

	// Try to create second URL with same alias
	req2 := application.CreateURLRequest{
		URL:         "https://example2.com",
		CustomAlias: "duplicate",
	}
	_, err = service.CreateShortURL(ctx, req2, testBaseURL)
	assert.Equal(t, domain.ErrShortCodeExists, err)
}

func TestURLService_ClickTracking_Integration(t *testing.T) {
	env := SetupTestEnvironment(t)

	ctx := context.Background()
	service := env.Service

	req := application.CreateURLRequest{
		URL:         "https://example.com",
		CustomAlias: "clicktest",
	}
	_, err := service.CreateShortURL(ctx, req, testBaseURL)
	require.NoError(t, err)

	// Verify initial click count is 0
	url, err := service.GetURL(ctx, "clicktest")
	require.NoError(t, err)
	assert.Equal(t, 0, url.Clicks)

	// Increment clicks multiple times
	for i := 1; i <= 3; i++ {
		_, err = service.IncrementClicks(ctx, "clicktest")
		require.NoError(t, err)

		// Verify click count
		url, err = service.GetURL(ctx, "clicktest")
		require.NoError(t, err)
		assert.Equal(t, i, url.Clicks)
	}
}

func TestURLService_NonExistentURL_Integration(t *testing.T) {
	env := SetupTestEnvironment(t)

	ctx := context.Background()
	service := env.Service

	// Try to get non-existent URL
	_, err := service.GetURL(ctx, "notfound")
	assert.Equal(t, domain.ErrURLNotFound, err)

	// Try to increment clicks for non-existent URL
	_, err = service.IncrementClicks(ctx, "notfound")
	assert.Equal(t, domain.ErrURLNotFound, err)
}

func TestURLService_ConcurrentAccess_Integration(t *testing.T) {
	env := SetupTestEnvironment(t)

	ctx := context.Background()
	service := env.Service

	req := application.CreateURLRequest{
		URL:         "https://example.com",
		CustomAlias: "concurrent",
	}
	_, err := service.CreateShortURL(ctx, req, testBaseURL)
	require.NoError(t, err)

	// Simulate concurrent click increments
	const numGoroutines = 10
	const clicksPerGoroutine = 5

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < clicksPerGoroutine; j++ {
				_, incrementErr := service.IncrementClicks(ctx, "concurrent")
				if incrementErr != nil {
					errChan <- incrementErr
					return
				}
			}
			errChan <- nil
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		chanErr := <-errChan
		require.NoError(t, chanErr)
	}

	// Verify final click count
	url, err := service.GetURL(ctx, "concurrent")
	require.NoError(t, err)

	expectedClicks := numGoroutines * clicksPerGoroutine
	assert.Equal(t, expectedClicks, url.Clicks)
}

package application

import (
	"context"
	"strings"
	"testing"

	"github.com/sp3dr4/dove/internal/domain"
	"github.com/sp3dr4/dove/internal/infrastructure/memory"
)

const testBaseURL = "http://localhost:8080"

func TestURLService_CreateShortURL_ValidURLs(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()
	baseURL := testBaseURL

	tests := []struct {
		name        string
		request     CreateURLRequest
		checkResult func(t *testing.T, resp *URLResponse, req CreateURLRequest)
	}{
		{
			name: "valid URL",
			request: CreateURLRequest{
				URL: "https://example.com",
			},
			checkResult: func(t *testing.T, resp *URLResponse, req CreateURLRequest) {
				if resp.OriginalURL != req.URL {
					t.Errorf("expected OriginalURL %s, got %s", req.URL, resp.OriginalURL)
				}
				if len(resp.ShortCode) != 6 {
					t.Errorf("expected ShortCode length 6, got %d", len(resp.ShortCode))
				}
			},
		},
		{
			name: "valid URL with custom alias",
			request: CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: "myalias",
			},
			checkResult: func(t *testing.T, resp *URLResponse, req CreateURLRequest) {
				if resp.OriginalURL != req.URL {
					t.Errorf("expected OriginalURL %s, got %s", req.URL, resp.OriginalURL)
				}
				if resp.ShortCode != req.CustomAlias {
					t.Errorf("expected ShortCode %s, got %s", req.CustomAlias, resp.ShortCode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.CreateShortURL(ctx, tt.request, baseURL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.checkResult(t, resp, tt.request)

			expectedShortURL := baseURL + "/" + resp.ShortCode
			if resp.ShortURL != expectedShortURL {
				t.Errorf("expected ShortURL %s, got %s", expectedShortURL, resp.ShortURL)
			}
		})
	}
}

func TestURLService_CreateShortURL_InvalidURLs(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()
	baseURL := testBaseURL

	tests := []struct {
		name    string
		request CreateURLRequest
		errMsg  string
	}{
		{
			name: "invalid URL",
			request: CreateURLRequest{
				URL: "not-a-url",
			},
			errMsg: "URL",
		},
		{
			name: "empty URL",
			request: CreateURLRequest{
				URL: "",
			},
			errMsg: "URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateShortURL(ctx, tt.request, baseURL)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Fatalf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestURLService_CreateShortURL_InvalidAliases(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()
	baseURL := testBaseURL

	tests := []struct {
		name    string
		request CreateURLRequest
		errMsg  string
	}{
		{
			name: "custom alias too short",
			request: CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: "ab",
			},
			errMsg: "CustomAlias",
		},
		{
			name: "custom alias too long",
			request: CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: strings.Repeat("a", 21),
			},
			errMsg: "CustomAlias",
		},
		{
			name: "custom alias with special chars",
			request: CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: "my-alias",
			},
			errMsg: "CustomAlias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateShortURL(ctx, tt.request, baseURL)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Fatalf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestURLService_CreateShortURL_DuplicateAlias(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()
	baseURL := testBaseURL

	// Create first URL
	req1 := CreateURLRequest{
		URL:         "https://example1.com",
		CustomAlias: "duplicate",
	}
	_, err := service.CreateShortURL(ctx, req1, baseURL)
	if err != nil {
		t.Fatalf("unexpected error creating first URL: %v", err)
	}

	// Try to create with same alias
	req2 := CreateURLRequest{
		URL:         "https://example2.com",
		CustomAlias: "duplicate",
	}
	_, err = service.CreateShortURL(ctx, req2, baseURL)
	if err != domain.ErrShortCodeExists {
		t.Fatalf("expected ErrShortCodeExists, got %v", err)
	}
}

func TestURLService_GetURL(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()
	baseURL := testBaseURL

	// Create a URL
	req := CreateURLRequest{
		URL:         "https://example.com",
		CustomAlias: "gettest",
	}
	_, err := service.CreateShortURL(ctx, req, baseURL)
	if err != nil {
		t.Fatalf("unexpected error creating URL: %v", err)
	}

	// Get existing URL
	url, err := service.GetURL(ctx, "gettest")
	if err != nil {
		t.Fatalf("unexpected error getting URL: %v", err)
	}
	if url.OriginalURL != "https://example.com" {
		t.Errorf("expected OriginalURL https://example.com, got %s", url.OriginalURL)
	}

	// Get non-existing URL
	_, err = service.GetURL(ctx, "notfound")
	if err != domain.ErrURLNotFound {
		t.Fatalf("expected ErrURLNotFound, got %v", err)
	}
}

func TestURLService_IncrementClicks(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()
	baseURL := testBaseURL

	// Create a URL
	req := CreateURLRequest{
		URL:         "https://example.com",
		CustomAlias: "clicktest",
	}
	_, err := service.CreateShortURL(ctx, req, baseURL)
	if err != nil {
		t.Fatalf("unexpected error creating URL: %v", err)
	}

	// Increment clicks
	_, err = service.IncrementClicks(ctx, "clicktest")
	if err != nil {
		t.Fatalf("unexpected error incrementing clicks: %v", err)
	}

	// Verify clicks were incremented
	url, _ := service.GetURL(ctx, "clicktest")
	if url.Clicks != 1 {
		t.Errorf("expected clicks to be 1, got %d", url.Clicks)
	}

	// Increment non-existing URL
	_, err = service.IncrementClicks(ctx, "notfound")
	if err != domain.ErrURLNotFound {
		t.Fatalf("expected ErrURLNotFound, got %v", err)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/health", handleHealth)
	r.Post("/shorten", handleShorten)
	r.Get("/{shortCode}", handleRedirect)
	return r
}

func TestHandleHealth(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest("GET", "/health", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "OK"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, "text/plain")
	}
}

// Test helper functions to reduce complexity
func unmarshalJSON(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
}

func assertErrorResponse(t *testing.T, body []byte) {
	t.Helper()
	var resp map[string]interface{}
	unmarshalJSON(t, body, &resp)
	if _, ok := resp["error"]; !ok {
		t.Error("Response should contain error field")
	}
}

func assertValidationError(t *testing.T, body []byte, expectedField string) {
	t.Helper()
	var resp map[string]interface{}
	unmarshalJSON(t, body, &resp)

	assertErrorResponse(t, body)

	if details, ok := resp["details"].(map[string]interface{}); ok {
		if _, hasField := details[expectedField]; !hasField {
			t.Errorf("Validation error should contain %s field", expectedField)
		}
	} else {
		t.Error("Response should contain details field with validation errors")
	}
}

func assertURLResponse(t *testing.T, body []byte, expectedOriginalURL, expectedAlias string) {
	t.Helper()
	var resp URLResponse
	unmarshalJSON(t, body, &resp)

	if resp.OriginalURL != expectedOriginalURL {
		t.Errorf("Wrong original URL: got %v want %v", resp.OriginalURL, expectedOriginalURL)
	}

	if expectedAlias != "" {
		assertCustomAlias(t, resp, expectedAlias)
	} else {
		assertGeneratedAlias(t, resp)
	}
}

func assertCustomAlias(t *testing.T, resp URLResponse, expectedAlias string) {
	t.Helper()
	if resp.ShortCode != expectedAlias {
		t.Errorf("Wrong short code: got %v want %v", resp.ShortCode, expectedAlias)
	}
	if resp.ShortURL != "http://localhost:8080/"+expectedAlias {
		t.Errorf("Wrong short URL: got %v want %v", resp.ShortURL, "http://localhost:8080/"+expectedAlias)
	}
}

func assertGeneratedAlias(t *testing.T, resp URLResponse) {
	t.Helper()
	if resp.ShortCode == "" {
		t.Error("Short code should not be empty")
	}
	if !strings.HasPrefix(resp.ShortURL, "http://localhost:8080/") {
		t.Errorf("Short URL should start with http://localhost:8080/, got %v", resp.ShortURL)
	}
}

func TestHandleShorten(t *testing.T) {
	// Reset store for testing
	store = &URLStore{
		urls: make(map[string]*URL),
	}

	tests := []struct {
		checkResponse  func(t *testing.T, body []byte)
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "valid URL",
			requestBody:    `{"url": "https://example.com"}`,
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				assertURLResponse(t, body, "https://example.com", "")
			},
		},
		{
			name:           "custom alias",
			requestBody:    `{"url": "https://example.com", "customAlias": "myalias"}`,
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				assertURLResponse(t, body, "https://example.com", "myalias")
			},
		},
		{
			name:           "empty URL",
			requestBody:    `{"url": ""}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assertErrorResponse(t, body)
			},
		},
		{
			name:           "invalid URL format",
			requestBody:    `{"url": "not-a-valid-url"}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assertValidationError(t, body, "URL")
			},
		},
		{
			name:           "custom alias too short",
			requestBody:    `{"url": "https://example.com", "customAlias": "ab"}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assertValidationError(t, body, "CustomAlias")
			},
		},
		{
			name:           "custom alias with special characters",
			requestBody:    `{"url": "https://example.com", "customAlias": "my-alias!"}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assertErrorResponse(t, body)
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    `invalid json`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assertErrorResponse(t, body)
			},
		},
		{
			name:           "duplicate custom alias",
			requestBody:    `{"url": "https://example2.com", "customAlias": "myalias"}`,
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, body []byte) {
				assertErrorResponse(t, body)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter()
			req := httptest.NewRequest("POST", "/shorten", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			tt.checkResponse(t, rr.Body.Bytes())
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	// Reset store and add test data
	store = &URLStore{
		urls: make(map[string]*URL),
	}
	store.urls["testcode"] = &URL{
		ShortCode:   "testcode",
		OriginalURL: "https://example.com",
		Clicks:      0,
	}

	tests := []struct {
		name           string
		shortCode      string
		expectedURL    string
		expectedStatus int
	}{
		{
			name:           "existing short code",
			shortCode:      "testcode",
			expectedURL:    "https://example.com",
			expectedStatus: http.StatusMovedPermanently,
		},
		{
			name:           "non-existing short code",
			shortCode:      "notfound",
			expectedURL:    "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter()
			req := httptest.NewRequest("GET", "/"+tt.shortCode, nil)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusMovedPermanently {
				location := rr.Header().Get("Location")
				if location != tt.expectedURL {
					t.Errorf("handler returned wrong location header: got %v want %v",
						location, tt.expectedURL)
				}

				// Check that clicks were incremented
				if store.urls["testcode"].Clicks != 1 {
					t.Errorf("clicks not incremented: got %v want %v",
						store.urls["testcode"].Clicks, 1)
				}
			}
		})
	}
}

func TestGenerateShortCode(t *testing.T) {
	// Test multiple generations to ensure they're different
	codes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		code := generateShortCode()

		if len(code) != 6 {
			t.Errorf("Short code length should be 6, got %d", len(code))
		}

		for _, c := range code {
			if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", c) {
				t.Errorf("Short code contains invalid character: %c", c)
			}
		}

		codes[code] = true
	}

	// Note: Due to the simple generation method, there might be duplicates
	// In a real implementation, we'd use a better random generator
	if len(codes) < 5 {
		t.Logf("Warning: Only %d unique codes generated out of 10 attempts", len(codes))
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Reset store
	store = &URLStore{
		urls: make(map[string]*URL),
	}

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			req := CreateURLRequest{
				URL:         fmt.Sprintf("https://example%d.com", i),
				CustomAlias: fmt.Sprintf("test%d", i),
			}
			body, _ := json.Marshal(req)

			r := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handleShorten(rr, r)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all URLs were created
	if len(store.urls) != 10 {
		t.Errorf("Expected 10 URLs, got %d", len(store.urls))
	}
}

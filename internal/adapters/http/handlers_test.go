package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sp3dr4/dove/internal/application"
	"github.com/sp3dr4/dove/internal/infrastructure/memory"
)

func TestHandlers_HandleShorten_ValidationErrorCasing(t *testing.T) {
	repo := memory.NewURLRepository()
	service := application.NewURLService(repo)
	handlers := NewHandlers(service, "http://localhost:8080", repo)

	tests := []struct {
		name           string
		payload        string
		expectedFields []string
	}{
		{
			name:           "invalid customAlias should return customAlias in error",
			payload:        `{"url": "https://example.com", "customAlias": "ab"}`,
			expectedFields: []string{"customAlias"},
		},
		{
			name:           "missing url should return url in error",
			payload:        `{"customAlias": "validalias"}`,
			expectedFields: []string{"url"},
		},
		{
			name:           "invalid url should return url in error",
			payload:        `{"url": "not-a-url", "customAlias": "validalias"}`,
			expectedFields: []string{"url"},
		},
		{
			name:           "multiple validation errors should return correct field names",
			payload:        `{"url": "not-a-url", "customAlias": "ab"}`,
			expectedFields: []string{"url", "customAlias"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := performValidationTest(t, handlers, tt.payload)
			checkExpectedFields(t, details, tt.expectedFields)
			checkNoUnexpectedFields(t, details, tt.expectedFields)
		})
	}
}

func performValidationTest(t *testing.T, handlers *Handlers, payload string) map[string]interface{} {
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleShorten(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	details, ok := response["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected details field in response, got: %v", response)
	}

	return details
}

func checkExpectedFields(t *testing.T, details map[string]interface{}, expectedFields []string) {
	for _, expectedField := range expectedFields {
		if _, exists := details[expectedField]; !exists {
			t.Errorf("expected field %q in error details, but got fields: %v", expectedField, getKeys(details))
		}
	}
}

func checkNoUnexpectedFields(t *testing.T, details map[string]interface{}, expectedFields []string) {
	for field := range details {
		found := false
		for _, expectedField := range expectedFields {
			if field == expectedField {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected field %q in error details, expected only: %v", field, expectedFields)
		}
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

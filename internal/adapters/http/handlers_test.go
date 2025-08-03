package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok, "expected details field in response, got: %v", response)

	return details
}

func checkExpectedFields(t *testing.T, details map[string]interface{}, expectedFields []string) {
	for _, expectedField := range expectedFields {
		assert.Contains(t, details, expectedField, "expected field %q in error details, but got fields: %v", expectedField, getKeys(details))
	}
}

func checkNoUnexpectedFields(t *testing.T, details map[string]interface{}, expectedFields []string) {
	for field := range details {
		assert.Contains(t, expectedFields, field, "unexpected field %q in error details, expected only: %v", field, expectedFields)
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

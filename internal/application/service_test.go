package application

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sp3dr4/dove/internal/infrastructure/memory"
)

// TestURLService_ShortCodeGeneration tests the short code generation algorithm
func TestURLService_ShortCodeGeneration(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()

	// Test that generated short codes are unique and have correct length
	urls := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
	}

	shortCodes := make(map[string]bool)

	for _, url := range urls {
		req := CreateURLRequest{URL: url}
		resp, err := service.CreateShortURL(ctx, req, "http://localhost:8080")
		require.NoError(t, err)

		assert.Len(t, resp.ShortCode, 6)
		assert.NotContains(t, shortCodes, resp.ShortCode)
		shortCodes[resp.ShortCode] = true

		for _, char := range resp.ShortCode {
			assert.True(t, (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9'),
				"short code contains invalid character: %c", char)
		}
	}
}

// TestURLService_CustomAliasValidation tests custom alias validation logic
func TestURLService_CustomAliasValidation(t *testing.T) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	ctx := context.Background()

	tests := []struct {
		name        string
		customAlias string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid alias",
			customAlias: "validalias",
			expectError: false,
		},
		{
			name:        "minimum length alias",
			customAlias: "abc",
			expectError: false,
		},
		{
			name:        "maximum length alias",
			customAlias: strings.Repeat("a", 20),
			expectError: false,
		},
		{
			name:        "too short alias",
			customAlias: "ab",
			expectError: true,
			errorMsg:    "CustomAlias",
		},
		{
			name:        "too long alias",
			customAlias: strings.Repeat("a", 21),
			expectError: true,
			errorMsg:    "CustomAlias",
		},
		{
			name:        "alias with hyphen",
			customAlias: "my-alias",
			expectError: true,
			errorMsg:    "CustomAlias",
		},
		{
			name:        "alias with underscore",
			customAlias: "my_alias",
			expectError: true,
			errorMsg:    "CustomAlias",
		},
		{
			name:        "alias with space",
			customAlias: "my alias",
			expectError: true,
			errorMsg:    "CustomAlias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateURLRequest{
				URL:         "https://example.com",
				CustomAlias: tt.customAlias,
			}

			_, err := service.CreateShortURL(ctx, req, "http://localhost:8080")

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

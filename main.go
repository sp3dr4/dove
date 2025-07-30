// Package main implements a simple URL shortener service.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

const (
	defaultTimeout      = 60 * time.Second
	defaultReadTimeout  = 15 * time.Second
	defaultWriteTimeout = 15 * time.Second
	defaultIdleTimeout  = 60 * time.Second
	shortCodeLength     = 6
)

// URL represents a shortened URL with its metadata.
type URL struct {
	CreatedAt   time.Time `json:"createdAt"`
	ShortCode   string    `json:"shortCode"`
	OriginalURL string    `json:"originalUrl"`
	Clicks      int       `json:"clicks"`
}

// CreateURLRequest represents the request payload for creating a short URL.
type CreateURLRequest struct {
	URL         string `json:"url" validate:"required,url"`
	CustomAlias string `json:"customAlias,omitempty" validate:"omitempty,alphanum,min=3,max=20"`
}

// URLResponse represents the response when a URL is successfully shortened.
type URLResponse struct {
	CreatedAt   time.Time `json:"createdAt"`
	ShortURL    string    `json:"shortUrl"`
	ShortCode   string    `json:"shortCode"`
	OriginalURL string    `json:"originalUrl"`
}

// URLStore provides thread-safe storage for URLs.
type URLStore struct {
	urls map[string]*URL
	mu   sync.RWMutex
}

var (
	store = &URLStore{
		urls: make(map[string]*URL),
	}
	validate = validator.New()
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(defaultTimeout))

	r.Get("/health", handleHealth)
	r.Post("/shorten", handleShorten)
	r.Get("/{shortCode}", handleRedirect)

	port := ":8080"
	slog.Info("Starting server", "port", port)

	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "OK")
}

func handleShorten(w http.ResponseWriter, r *http.Request) {
	var req CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validate.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		if !errors.As(err, &validationErrors) {
			respondWithError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		errorMessages := make(map[string]string)

		for _, e := range validationErrors {
			field := e.Field()
			switch e.Tag() {
			case "required":
				errorMessages[field] = fmt.Sprintf("%s is required", field)
			case "url":
				errorMessages[field] = fmt.Sprintf("%s must be a valid URL", field)
			case "alphanum":
				errorMessages[field] = fmt.Sprintf("%s must contain only alphanumeric characters", field)
			case "min":
				errorMessages[field] = fmt.Sprintf("%s must be at least %s characters long", field, e.Param())
			case "max":
				errorMessages[field] = fmt.Sprintf("%s must be at most %s characters long", field, e.Param())
			default:
				errorMessages[field] = fmt.Sprintf("%s is invalid", field)
			}
		}

		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": errorMessages,
		})
		return
	}

	shortCode := req.CustomAlias
	if shortCode == "" {
		shortCode = generateShortCode()
	}

	store.mu.Lock()
	if _, exists := store.urls[shortCode]; exists {
		store.mu.Unlock()
		respondWithError(w, http.StatusConflict, "Short code already exists")
		return
	}

	url := &URL{
		ShortCode:   shortCode,
		OriginalURL: req.URL,
		CreatedAt:   time.Now(),
		Clicks:      0,
	}
	store.urls[shortCode] = url
	store.mu.Unlock()

	response := URLResponse{
		ShortURL:    fmt.Sprintf("http://localhost:8080/%s", shortCode),
		ShortCode:   shortCode,
		OriginalURL: req.URL,
		CreatedAt:   url.CreatedAt,
	}

	slog.Info("Created short URL", "short_code", shortCode, "original_url", req.URL)
	respondWithJSON(w, http.StatusCreated, response)
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")

	store.mu.RLock()
	url, exists := store.urls[shortCode]
	store.mu.RUnlock()

	if !exists {
		respondWithError(w, http.StatusNotFound, "Short URL not found")
		return
	}

	store.mu.Lock()
	url.Clicks++
	store.mu.Unlock()

	slog.Info("Redirecting", "short_code", shortCode, "original_url", url.OriginalURL, "clicks", url.Clicks)
	http.Redirect(w, r, url.OriginalURL, http.StatusMovedPermanently)
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, shortCodeLength)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]any{
		"error": map[string]string{
			"message": message,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

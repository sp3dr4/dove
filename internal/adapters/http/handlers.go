package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/sp3dr4/dove/internal/application"
	"github.com/sp3dr4/dove/internal/domain"
)

type Handlers struct {
	service *application.URLService
	baseURL string
	repo    domain.URLRepository
}

func NewHandlers(service *application.URLService, baseURL string, repo domain.URLRepository) *Handlers {
	return &Handlers{
		service: service,
		baseURL: baseURL,
		repo:    repo,
	}
}

// HandleHealth handles the health check endpoint.
//
//	@Summary		Health check endpoint
//	@Description	Check if the service is running
//	@Tags			health
//	@Produce		plain
//	@Success		200	{string}	string	"OK"
//	@Router			/health [get]
func (h *Handlers) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "OK")
}

// HandleReady handles the readiness check endpoint.
//
//	@Summary		Readiness check endpoint
//	@Description	Check if the service is ready to serve requests (includes database connectivity)
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	object{status=string,timestamp=string}	"Service is ready"
//	@Failure		503	{object}	ErrorResponse							"Service is not ready"
//	@Router			/ready [get]
func (h *Handlers) HandleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.repo.HealthCheck(ctx); err != nil {
		slog.Error("Readiness check failed", "error", err)
		respondWithError(w, http.StatusServiceUnavailable, "Service not ready: database unavailable")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// HandleShorten handles the URL shortening endpoint.
//
//	@Summary		Create a short URL
//	@Description	Create a shortened URL from a long URL
//	@Tags			urls
//	@Accept			json
//	@Produce		json
//	@Param			request	body		application.CreateURLRequest	true	"URL to shorten"
//	@Success		201		{object}	application.URLResponse			"Successfully created short URL"
//	@Failure		400		{object}	ValidationErrorResponse			"Invalid request or validation error"
//	@Failure		409		{object}	ErrorResponse					"Short code already exists"
//	@Router			/shorten [post]
func (h *Handlers) HandleShorten(w http.ResponseWriter, r *http.Request) {
	var req application.CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	response, err := h.service.CreateShortURL(r.Context(), req, h.baseURL)
	if err != nil {
		if errors.Is(err, domain.ErrShortCodeExists) {
			respondWithError(w, http.StatusConflict, "Short code already exists")
			return
		}

		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			handleValidationError(w, validationErrors)
			return
		}

		slog.Error("Failed to create short URL", "error", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create short URL")
		return
	}

	slog.Info("Created short URL", "short_code", response.ShortCode, "original_url", response.OriginalURL)
	respondWithJSON(w, http.StatusCreated, response)
}

// HandleRedirect handles the redirect endpoint.
//
//	@Summary		Redirect to original URL
//	@Description	Redirect to the original URL using the short code
//	@Tags			urls
//	@Param			shortCode	path	string	true	"Short code"
//	@Success		301			"Redirect to original URL"
//	@Failure		404			{object}	ErrorResponse	"Short URL not found"
//	@Router			/{shortCode} [get]
func (h *Handlers) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")

	url, err := h.service.GetURL(r.Context(), shortCode)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			respondWithError(w, http.StatusNotFound, "Short URL not found")
			return
		}
		slog.Error("Failed to get URL", "error", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get URL")
		return
	}

	updatedURL, err := h.service.IncrementClicks(r.Context(), shortCode)
	if err != nil {
		slog.Error("Failed to increment clicks", "error", err)
		// Continue with redirect even if click increment fails
		slog.Info("Redirecting", "short_code", shortCode, "original_url", url.OriginalURL, "clicks", url.Clicks)
	} else {
		slog.Info("Redirecting", "short_code", shortCode, "original_url", url.OriginalURL, "clicks", updatedURL.Clicks)
	}
	http.Redirect(w, r, url.OriginalURL, http.StatusMovedPermanently)
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error     map[string]string `json:"error"`
	Timestamp string            `json:"timestamp" example:"2024-01-31T12:00:00Z"`
}

// ValidationErrorResponse represents a validation error response.
type ValidationErrorResponse struct {
	Details map[string]string `json:"details"`
	Error   string            `json:"error" example:"Validation failed"`
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func handleValidationError(w http.ResponseWriter, validationErrors validator.ValidationErrors) {
	errorMessages := make(map[string]string)
	for _, e := range validationErrors {
		field := getJSONFieldName(e)
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
}

// getJSONFieldName extracts the JSON tag name from a validation error
func getJSONFieldName(e validator.FieldError) string {
	structType := getStructTypeFromError(e)
	if structType == nil {
		return e.Field()
	}

	field, found := structType.FieldByName(e.StructField())
	if !found {
		return e.Field()
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return e.Field()
	}

	if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
		jsonTag = jsonTag[:commaIndex]
	}

	return jsonTag
}

// getStructTypeFromError extracts the struct type from a validation error
func getStructTypeFromError(e validator.FieldError) reflect.Type {
	// Use the Type() method to get the field's type, then navigate to the parent struct
	// The StructNamespace gives us something like "CreateURLRequest.URL"
	namespace := e.StructNamespace()

	// Split the namespace to get the struct name
	parts := strings.Split(namespace, ".")
	if len(parts) < 2 {
		return nil
	}

	return getTypeFromStructName(parts[0])
}

// getTypeFromStructName returns the reflect.Type for a given struct name
// This acts as a registry for known request types
func getTypeFromStructName(structName string) reflect.Type {
	switch structName {
	case "CreateURLRequest":
		return reflect.TypeOf(application.CreateURLRequest{})
	// Add more request types here as needed
	// case "UpdateURLRequest":
	//     return reflect.TypeOf(application.UpdateURLRequest{})
	default:
		return nil
	}
}

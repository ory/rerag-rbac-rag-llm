// Package errors provides secure error handling utilities
package errors

import (
	"encoding/json"
	"log"
	"net/http"

	"llm-rag-poc/internal/config"
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	// RequestID is included in development/staging for debugging
	RequestID string `json:"request_id,omitempty"`
	// Details are only included in development mode
	Details string `json:"details,omitempty"`
}

// ErrorHandler provides secure error handling based on configuration
type ErrorHandler struct {
	config *config.Config
}

// NewErrorHandler creates a new error handler with the given configuration
func NewErrorHandler(cfg *config.Config) *ErrorHandler {
	return &ErrorHandler{
		config: cfg,
	}
}

// HandleAuthError handles authentication-related errors with consistent responses
func (h *ErrorHandler) HandleAuthError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	var response ErrorResponse

	if h.config.Security.ErrorMode == "secure" || h.config.IsProduction() {
		// In secure mode, provide minimal information to prevent user enumeration
		response = ErrorResponse{
			Code:      http.StatusUnauthorized,
			Status:    "Unauthorized",
			Message:   "Authentication required",
			RequestID: h.getRequestID(requestID),
		}
	} else {
		// In development mode, provide more details
		response = ErrorResponse{
			Code:      http.StatusUnauthorized,
			Status:    "Unauthorized",
			Message:   "Authentication failed",
			RequestID: requestID,
			Details:   err.Error(),
		}
	}

	h.logError("AUTH_ERROR", err, requestID, r)
	h.writeJSONError(w, response)
}

// HandleAuthorizationError handles authorization/permission errors
func (h *ErrorHandler) HandleAuthorizationError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	var response ErrorResponse

	if h.config.Security.ErrorMode == "secure" || h.config.IsProduction() {
		response = ErrorResponse{
			Code:      http.StatusForbidden,
			Status:    "Forbidden",
			Message:   "Access denied",
			RequestID: h.getRequestID(requestID),
		}
	} else {
		response = ErrorResponse{
			Code:      http.StatusForbidden,
			Status:    "Forbidden",
			Message:   "Permission denied",
			RequestID: requestID,
			Details:   err.Error(),
		}
	}

	h.logError("AUTHZ_ERROR", err, requestID, r)
	h.writeJSONError(w, response)
}

// HandleValidationError handles input validation errors
func (h *ErrorHandler) HandleValidationError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	var response ErrorResponse

	if h.config.Security.ErrorMode == "secure" || h.config.IsProduction() {
		response = ErrorResponse{
			Code:      http.StatusBadRequest,
			Status:    "Bad Request",
			Message:   "Invalid request",
			RequestID: h.getRequestID(requestID),
		}
	} else {
		response = ErrorResponse{
			Code:      http.StatusBadRequest,
			Status:    "Bad Request",
			Message:   "Invalid request parameters",
			RequestID: requestID,
			Details:   err.Error(),
		}
	}

	h.logError("VALIDATION_ERROR", err, requestID, r)
	h.writeJSONError(w, response)
}

// HandleInternalError handles internal server errors
func (h *ErrorHandler) HandleInternalError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	response := ErrorResponse{
		Code:      http.StatusInternalServerError,
		Status:    "Internal Server Error",
		Message:   "An internal error occurred",
		RequestID: h.getRequestID(requestID),
	}

	// Never expose internal error details in production
	if h.config.IsDevelopment() && h.config.Security.ErrorMode != "secure" {
		response.Details = err.Error()
	}

	h.logError("INTERNAL_ERROR", err, requestID, r)
	h.writeJSONError(w, response)
}

// HandleNotFoundError handles resource not found errors
func (h *ErrorHandler) HandleNotFoundError(w http.ResponseWriter, r *http.Request, resource string, requestID string) {
	var response ErrorResponse

	if h.config.Security.ErrorMode == "secure" || h.config.IsProduction() {
		response = ErrorResponse{
			Code:      http.StatusNotFound,
			Status:    "Not Found",
			Message:   "Resource not found",
			RequestID: h.getRequestID(requestID),
		}
	} else {
		response = ErrorResponse{
			Code:      http.StatusNotFound,
			Status:    "Not Found",
			Message:   "Resource not found: " + resource,
			RequestID: requestID,
		}
	}

	h.logError("NOT_FOUND", nil, requestID, r)
	h.writeJSONError(w, response)
}

// HandleRateLimitError handles rate limiting errors
func (h *ErrorHandler) HandleRateLimitError(w http.ResponseWriter, r *http.Request, requestID string) {
	response := ErrorResponse{
		Code:      http.StatusTooManyRequests,
		Status:    "Too Many Requests",
		Message:   "Rate limit exceeded",
		RequestID: h.getRequestID(requestID),
	}

	h.logError("RATE_LIMIT", nil, requestID, r)
	h.writeJSONError(w, response)
}

// HandleDatabaseError handles database-related errors
func (h *ErrorHandler) HandleDatabaseError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	response := ErrorResponse{
		Code:      http.StatusInternalServerError,
		Status:    "Internal Server Error",
		Message:   "Database operation failed",
		RequestID: h.getRequestID(requestID),
	}

	// Only show database errors in development
	if h.config.IsDevelopment() && h.config.Security.ErrorMode != "secure" {
		response.Details = err.Error()
	}

	h.logError("DATABASE_ERROR", err, requestID, r)
	h.writeJSONError(w, response)
}

// HandleServiceError handles external service errors (Ollama, Keto)
func (h *ErrorHandler) HandleServiceError(w http.ResponseWriter, r *http.Request, service string, err error, requestID string) {
	response := ErrorResponse{
		Code:      http.StatusBadGateway,
		Status:    "Bad Gateway",
		Message:   "External service unavailable",
		RequestID: h.getRequestID(requestID),
	}

	if h.config.IsDevelopment() && h.config.Security.ErrorMode != "secure" {
		response.Message = "Service unavailable: " + service
		response.Details = err.Error()
	}

	h.logError("SERVICE_ERROR", err, requestID, r)
	h.writeJSONError(w, response)
}

// writeJSONError writes an error response as JSON
func (h *ErrorHandler) writeJSONError(w http.ResponseWriter, response ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Code)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding error response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// logError logs errors with context
func (h *ErrorHandler) logError(errorType string, err error, requestID string, r *http.Request) {
	logData := map[string]interface{}{
		"type":       errorType,
		"request_id": requestID,
		"method":     r.Method,
		"path":       r.URL.Path,
		"user_agent": r.Header.Get("User-Agent"),
		"remote_ip":  getClientIP(r),
	}

	if err != nil {
		logData["error"] = err.Error()
	}

	if h.config.App.LogFormat == "json" {
		if jsonLog, jsonErr := json.Marshal(logData); jsonErr == nil {
			log.Printf("ERROR: %s", string(jsonLog))
		} else {
			log.Printf("ERROR: %s - %v", errorType, err)
		}
	} else {
		log.Printf("ERROR [%s] %s %s: %v (request_id: %s)",
			errorType, r.Method, r.URL.Path, err, requestID)
	}
}

// getRequestID returns request ID for logging, only in development
func (h *ErrorHandler) getRequestID(requestID string) string {
	if h.config.IsProduction() && h.config.Security.ErrorMode == "secure" {
		return ""
	}
	return requestID
}

// getClientIP extracts the real client IP from request headers
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Predefined error types for common scenarios

// ErrInvalidAuthHeader indicates malformed authorization header
var ErrInvalidAuthHeader = &StandardError{
	Type:    "INVALID_AUTH_HEADER",
	Message: "Invalid authorization header format",
}

// ErrMissingAuthHeader indicates missing authorization header
var ErrMissingAuthHeader = &StandardError{
	Type:    "MISSING_AUTH_HEADER",
	Message: "Missing authorization header",
}

// ErrUserNotFound indicates user not found
var ErrUserNotFound = &StandardError{
	Type:    "USER_NOT_FOUND",
	Message: "User not found",
}

// ErrInvalidToken indicates invalid token
var ErrInvalidToken = &StandardError{
	Type:    "INVALID_TOKEN",
	Message: "Invalid token",
}

// StandardError represents a standard application error
type StandardError struct {
	Type    string
	Message string
	Cause   error
}

// Error implements the error interface
func (e *StandardError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause
func (e *StandardError) Unwrap() error {
	return e.Cause
}

// WithCause adds a cause to the error
func (e *StandardError) WithCause(cause error) *StandardError {
	return &StandardError{
		Type:    e.Type,
		Message: e.Message,
		Cause:   cause,
	}
}

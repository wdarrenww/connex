package middleware

import (
	"encoding/json"
	"net/http"

	"connex/pkg/logger"

	"go.uber.org/zap"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Request string `json:"request_id,omitempty"`
}

// WriteError writes a standardized error response
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteStructuredError(w, status, msg, "", "")
}

// WriteStructuredError writes a structured error response with additional context
func WriteStructuredError(w http.ResponseWriter, status int, msg, code, requestID string) {
	// Log the error for debugging (but don't expose sensitive details)
	logger.GetGlobal().Error("API error",
		zap.Int("status", status),
		zap.String("message", msg),
		zap.String("code", code),
		zap.String("request_id", requestID),
	)

	// Create standardized error response
	response := ErrorResponse{
		Error:   getSafeErrorMessage(status, msg),
		Code:    code,
		Request: requestID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// getSafeErrorMessage returns a safe error message that doesn't leak sensitive information
func getSafeErrorMessage(status int, msg string) string {
	// For client errors (4xx), we can be more specific
	if status >= 400 && status < 500 {
		return msg
	}

	// For server errors (5xx), use generic messages to avoid information leakage
	switch status {
	case http.StatusInternalServerError:
		return "Internal server error"
	case http.StatusServiceUnavailable:
		return "Service temporarily unavailable"
	case http.StatusGatewayTimeout:
		return "Request timeout"
	default:
		return "An error occurred"
	}
}

// WriteValidationError writes a validation error response
func WriteValidationError(w http.ResponseWriter, field, message string) {
	response := ErrorResponse{
		Error: "Validation failed",
		Code:  "VALIDATION_ERROR",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}

// WriteAuthenticationError writes an authentication error response
func WriteAuthenticationError(w http.ResponseWriter, message string) {
	response := ErrorResponse{
		Error: "Authentication failed",
		Code:  "AUTH_ERROR",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(response)
}

// WriteAuthorizationError writes an authorization error response
func WriteAuthorizationError(w http.ResponseWriter, message string) {
	response := ErrorResponse{
		Error: "Access denied",
		Code:  "FORBIDDEN",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(response)
}

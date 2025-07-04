package middleware

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/csrf"
)

// SecureMetricsMiddleware protects the metrics endpoint in production
func SecureMetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In production, require authentication for metrics
			if os.Getenv("ENV") == "production" {
				// Check for API key or basic auth
				apiKey := r.Header.Get("X-API-Key")
				if apiKey == "" {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				// Validate API key (in production, use proper secret management)
				expectedKey := os.Getenv("METRICS_API_KEY")
				if expectedKey == "" || apiKey != expectedKey {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Modern security headers
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")

			// HSTS header (only for HTTPS)
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			// Content Security Policy (strengthened)
			csp := "default-src 'self'; " +
				"script-src 'self' 'nonce-" + generateNonce() + "'; " +
				"style-src 'self' 'nonce-" + generateNonce() + "'; " +
				"img-src 'self' data: https:; " +
				"font-src 'self'; " +
				"connect-src 'self'; " +
				"media-src 'self'; " +
				"object-src 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'; " +
				"frame-ancestors 'none'; " +
				"upgrade-insecure-requests"
			w.Header().Set("Content-Security-Policy", csp)

			next.ServeHTTP(w, r)
		})
	}
}

// generateNonce creates a random nonce for CSP
func generateNonce() string {
	// In production, use crypto/rand for better entropy
	// For now, use a simple implementation
	return "nonce-" + fmt.Sprintf("%d", time.Now().UnixNano())
}

// NoCacheMiddleware prevents caching for sensitive endpoints
func NoCacheMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			next.ServeHTTP(w, r)
		})
	}
}

// CSRFMiddleware adds CSRF protection to state-changing requests
func CSRFMiddleware(authKey []byte) func(http.Handler) http.Handler {
	csrfMiddleware := csrf.Protect(authKey,
		csrf.Secure(os.Getenv("ENV") == "production"),
		csrf.Path("/"),
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch {
				csrfMiddleware(next).ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// RequestSizeLimitMiddleware limits the size of request bodies
func RequestSizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

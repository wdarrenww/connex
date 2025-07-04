package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connex/internal/api/auth"
	"connex/internal/api/user"
	"connex/internal/config"
	"connex/internal/middleware"
	"connex/pkg/logger"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecurityIntegration tests security features with full middleware stack
func TestSecurityIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	cfg, err := config.Load()
	require.NoError(t, err)

	// Initialize logger
	logger.InitGlobal("debug", "test")

	// Create router with all middleware
	r := chi.NewRouter()

	// Add security middleware
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.RequestSizeLimitMiddleware(1024 * 1024)) // 1MB limit
	r.Use(middleware.IPRateLimit(10, time.Minute))            // 10 requests per minute
	r.Use(middleware.CSRFMiddleware([]byte("test-csrf-key")))
	r.Use(middleware.SecurityMonitoringMiddleware())

	// Create services and handlers
	userService := user.NewService()
	authHandler := auth.NewHandler(userService, cfg.JWT.Secret)

	// Setup routes
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)

	// Test 1: Security Headers
	t.Run("SecurityHeaders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/login", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		headers := w.Header()
		requiredHeaders := []string{
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
			"Referrer-Policy",
			"Permissions-Policy",
			"Cross-Origin-Embedder-Policy",
			"Cross-Origin-Opener-Policy",
			"Cross-Origin-Resource-Policy",
			"Content-Security-Policy",
		}

		for _, header := range requiredHeaders {
			assert.NotEmpty(t, headers.Get(header), "Missing security header: %s", header)
		}

		// Check CSP
		csp := headers.Get("Content-Security-Policy")
		assert.Contains(t, csp, "nonce-", "CSP should contain nonce")
		assert.NotContains(t, csp, "unsafe-inline", "CSP should not contain unsafe-inline")
	})

	// Test 2: Request Size Limiting
	t.Run("RequestSizeLimit", func(t *testing.T) {
		// Create large payload (2MB)
		largeBody := strings.Repeat("a", 2*1024*1024)
		reqBody := map[string]interface{}{
			"name":     largeBody,
			"email":    "test@example.com",
			"password": "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "Large request should be rejected")
	})

	// Test 3: Rate Limiting
	t.Run("RateLimiting", func(t *testing.T) {
		// Make multiple requests to trigger rate limiting
		for i := 0; i < 15; i++ {
			reqBody := map[string]interface{}{
				"email":    fmt.Sprintf("test%d@example.com", i),
				"password": "wrongpassword",
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if i >= 10 {
				// After 10 requests, should be rate limited
				assert.Equal(t, http.StatusTooManyRequests, w.Code, "Should be rate limited after 10 requests")
			}
		}
	})

	// Test 4: CSRF Protection
	t.Run("CSRFProtection", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "Test User",
			"email":    "csrf-test@example.com",
			"password": "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		// No CSRF token - should be rejected
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code, "Request without CSRF token should be rejected")
	})

	// Test 5: Input Validation
	t.Run("InputValidation", func(t *testing.T) {
		testCases := []struct {
			name     string
			email    string
			password string
			expected int
		}{
			{"invalid email", "invalid-email", "SecurePass123!", http.StatusBadRequest},
			{"weak password", "test@example.com", "weak", http.StatusBadRequest},
			{"xss in name", "test@example.com", "SecurePass123!", http.StatusBadRequest},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reqBody := map[string]interface{}{
					"name":     "<script>alert('xss')</script>",
					"email":    tc.email,
					"password": tc.password,
				}
				body, _ := json.Marshal(reqBody)

				req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				assert.Equal(t, tc.expected, w.Code, "Input validation should reject invalid input")
			})
		}
	})

	// Test 6: Error Handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test invalid JSON
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check that error response has expected structure
		assert.Contains(t, response, "error")
		assert.NotContains(t, response, "stack_trace")
		assert.NotContains(t, response, "internal_error")
	})

	// Test 7: JWT Validation
	t.Run("JWTValidation", func(t *testing.T) {
		// Test with invalid token
		req := httptest.NewRequest("GET", "/api/users/1", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid JWT should be rejected")

		// Test without token
		req = httptest.NewRequest("GET", "/api/users/1", nil)
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Missing JWT should be rejected")
	})

	// Test 8: Security Monitoring
	t.Run("SecurityMonitoring", func(t *testing.T) {
		// Test failed login attempts
		for i := 0; i < 5; i++ {
			reqBody := map[string]interface{}{
				"email":    "monitoring-test@example.com",
				"password": "wrongpassword",
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// Should return 401 for failed login
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		}

		// Test suspicious user agent
		req := httptest.NewRequest("GET", "/api/auth/login", nil)
		req.Header.Set("User-Agent", "sqlmap/1.0")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Should still work but be logged
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestSecurityHeadersComprehensive tests all security headers in detail
func TestSecurityHeadersComprehensive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(middleware.SecurityHeadersMiddleware())

	userService := user.NewService()
	authHandler := auth.NewHandler(userService, cfg.JWT.Secret)
	r.Post("/api/auth/register", authHandler.Register)

	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	headers := w.Header()

	// Test specific header values
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))

	// Test CSP
	csp := headers.Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' 'nonce-")
	assert.Contains(t, csp, "style-src 'self' 'nonce-")
	assert.Contains(t, csp, "img-src 'self' data: https:")
	assert.Contains(t, csp, "font-src 'self'")
	assert.Contains(t, csp, "connect-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.NotContains(t, csp, "unsafe-inline")
	assert.NotContains(t, csp, "unsafe-eval")
}

// TestRateLimitingComprehensive tests rate limiting in detail
func TestRateLimitingComprehensive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(middleware.IPRateLimit(5, time.Minute)) // 5 requests per minute

	userService := user.NewService()
	authHandler := auth.NewHandler(userService, cfg.JWT.Secret)
	r.Post("/api/auth/login", authHandler.Login)

	// Test rate limiting with different IPs
	testCases := []struct {
		name     string
		ip       string
		expected int
	}{
		{"same IP multiple requests", "192.168.1.1", http.StatusTooManyRequests},
		{"different IP", "192.168.1.2", http.StatusBadRequest}, // Should not be rate limited
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Make multiple requests
			for i := 0; i < 10; i++ {
				reqBody := map[string]interface{}{
					"email":    fmt.Sprintf("test%d@example.com", i),
					"password": "wrongpassword",
				}
				body, _ := json.Marshal(reqBody)

				req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Forwarded-For", tc.ip)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				if i >= 5 && tc.ip == "192.168.1.1" {
					assert.Equal(t, http.StatusTooManyRequests, w.Code, "Should be rate limited after 5 requests")
				}
			}
		})
	}
}

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Initialize logger for tests
	logger.InitGlobal("debug", "test")

	// Run tests
	m.Run()
}

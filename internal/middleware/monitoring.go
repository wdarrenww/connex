package middleware

import (
	"net/http"
	"sync"
	"time"

	"connex/pkg/logger"
	"connex/pkg/telemetry"

	"go.uber.org/zap"
)

// SecurityMonitoringMiddleware monitors for security events
func SecurityMonitoringMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Capture response status for monitoring
			responseWriter := &securityResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(responseWriter, r)

			// Monitor for security events
			monitorSecurityEvents(r, responseWriter.statusCode)
		})
	}
}

// monitorSecurityEvents checks for various security events
func monitorSecurityEvents(r *http.Request, statusCode int) {
	// Monitor failed authentication attempts
	if r.URL.Path == "/api/auth/login" && statusCode == http.StatusUnauthorized {
		logger.GetGlobal().Warn("failed login attempt",
			zap.String("ip", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.String("path", r.URL.Path),
			zap.Int("status", statusCode),
		)
		telemetry.RecordSecurityEvent("failed_login", r.RemoteAddr)
	}

	// Monitor rate limit violations
	if statusCode == http.StatusTooManyRequests {
		logger.GetGlobal().Warn("rate limit exceeded",
			zap.String("ip", r.RemoteAddr),
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
		)
		telemetry.RecordSecurityEvent("rate_limit_violation", r.RemoteAddr)
	}

	// Monitor suspicious patterns
	if isSuspiciousRequest(r) {
		logger.GetGlobal().Warn("suspicious request detected",
			zap.String("ip", r.RemoteAddr),
			zap.String("path", r.URL.Path),
			zap.String("user_agent", r.UserAgent()),
			zap.String("referer", r.Referer()),
		)
		telemetry.RecordSecurityEvent("suspicious_request", r.RemoteAddr)
	}
}

// isSuspiciousRequest checks for suspicious request patterns
func isSuspiciousRequest(r *http.Request) bool {
	// Check for common attack patterns in User-Agent
	suspiciousUserAgents := []string{
		"sqlmap", "nikto", "nmap", "wget", "curl", "python", "perl",
		"masscan", "dirb", "gobuster", "wfuzz", "burp", "zap",
	}

	userAgent := r.UserAgent()
	for _, suspicious := range suspiciousUserAgents {
		if containsIgnoreCase(userAgent, suspicious) {
			return true
		}
	}

	// Check for suspicious paths
	suspiciousPaths := []string{
		"/admin", "/wp-admin", "/phpmyadmin", "/config", "/.env",
		"/.git", "/.svn", "/backup", "/test", "/debug",
	}

	path := r.URL.Path
	for _, suspicious := range suspiciousPaths {
		if containsIgnoreCase(path, suspicious) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if a string contains another string (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(s) == len(substr) && s == substr ||
			len(s) > len(substr) && (contains(s, substr) || contains(s, substr)))
}

// contains checks if a string contains another string
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(s) == len(substr) && s == substr ||
			len(s) > len(substr) && (s[:len(substr)] == substr ||
				contains(s[1:], substr)))
}

// securityResponseWriter captures the status code for monitoring
type securityResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *securityResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *securityResponseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// FailedLoginTracker tracks failed login attempts per IP
type FailedLoginTracker struct {
	attempts    map[string]int
	lastAttempt map[string]time.Time
	mutex       sync.RWMutex
}

var globalFailedLoginTracker = &FailedLoginTracker{
	attempts:    make(map[string]int),
	lastAttempt: make(map[string]time.Time),
}

// RecordFailedLogin records a failed login attempt
func (t *FailedLoginTracker) RecordFailedLogin(ip string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	now := time.Now()
	t.attempts[ip]++
	t.lastAttempt[ip] = now

	// Log warning if multiple failed attempts
	if t.attempts[ip] >= 3 {
		logger.GetGlobal().Warn("multiple failed login attempts",
			zap.String("ip", ip),
			zap.Int("attempts", t.attempts[ip]),
			zap.Time("last_attempt", t.lastAttempt[ip]),
		)
	}
}

// GetFailedAttempts returns the number of failed attempts for an IP
func (t *FailedLoginTracker) GetFailedAttempts(ip string) int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.attempts[ip]
}

// ResetFailedAttempts resets failed attempts for an IP (after successful login)
func (t *FailedLoginTracker) ResetFailedAttempts(ip string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.attempts, ip)
	delete(t.lastAttempt, ip)
}

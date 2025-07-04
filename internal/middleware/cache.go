package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"connex/internal/cache"
)

type CacheConfig struct {
	TTL           time.Duration
	KeyFunc       func(r *http.Request) string
	SkipCacheFunc func(r *http.Request) bool
}

// Cache creates a caching middleware
func Cache(config CacheConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Check if we should skip caching
			if config.SkipCacheFunc != nil && config.SkipCacheFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Generate cache key
			key := config.KeyFunc(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Hash the key for consistent length
			hash := md5.Sum([]byte(key))
			cacheKey := "cache:" + hex.EncodeToString(hash[:])

			// Try to get from cache
			var cachedResponse CachedResponse
			if err := cache.GetValue(cacheKey, &cachedResponse); err == nil {
				// Serve from cache
				for key, values := range cachedResponse.Headers {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(cachedResponse.StatusCode)
				w.Write(cachedResponse.Body)
				return
			}

			// Cache miss, capture response
			captureWriter := &responseCapture{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				headers:        make(http.Header),
				body:           []byte{},
			}

			next.ServeHTTP(captureWriter, r)

			// Cache successful responses
			if captureWriter.statusCode >= 200 && captureWriter.statusCode < 300 {
				cachedResponse := CachedResponse{
					StatusCode: captureWriter.statusCode,
					Headers:    captureWriter.headers,
					Body:       captureWriter.body,
				}
				cache.Set(cacheKey, cachedResponse, config.TTL)
			}
		})
	}
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
}

// responseCapture captures the response for caching
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	headers    http.Header
	body       []byte
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.statusCode = statusCode
	rc.ResponseWriter.WriteHeader(statusCode)
}

func (rc *responseCapture) Write(data []byte) (int, error) {
	rc.body = append(rc.body, data...)
	return rc.ResponseWriter.Write(data)
}

func (rc *responseCapture) Header() http.Header {
	return rc.headers
}

// URLPathCache creates caching based on URL path
func URLPathCache(ttl time.Duration) func(http.Handler) http.Handler {
	return Cache(CacheConfig{
		TTL: ttl,
		KeyFunc: func(r *http.Request) string {
			return r.URL.Path
		},
		SkipCacheFunc: func(r *http.Request) bool {
			// Skip caching for auth endpoints
			return strings.HasPrefix(r.URL.Path, "/api/auth")
		},
	})
}

// InvalidateCache invalidates cache entries matching a pattern
func InvalidateCache(pattern string) error {
	// This is a simplified implementation
	// In production, you might want to use Redis SCAN or maintain a cache index
	return cache.Delete(pattern)
}

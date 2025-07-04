package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connex/internal/cache"
	"net"
)

type RateLimitConfig struct {
	Requests int                          // Number of requests allowed
	Window   time.Duration                // Time window
	KeyFunc  func(r *http.Request) string // Function to generate rate limit key
}

// RateLimit creates a rate limiting middleware
func RateLimit(config RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := config.KeyFunc(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			limitKey := fmt.Sprintf("rate_limit:%s", key)
			windowKey := fmt.Sprintf("rate_limit_window:%s", key)

			redis := cache.Get()
			ctx := context.Background()

			// Get current window
			window, err := redis.Get(ctx, windowKey).Int64()
			if err != nil && err.Error() != "redis: nil" {
				// Redis error, allow request
				next.ServeHTTP(w, r)
				return
			}

			now := time.Now().Unix()
			if err != nil && err.Error() == "redis: nil" || now-window > int64(config.Window.Seconds()) {
				// New window or expired window
				window = now
				redis.Set(ctx, windowKey, window, config.Window)
				redis.Set(ctx, limitKey, 1, config.Window)
			} else {
				// Increment counter
				count, err := redis.Incr(ctx, limitKey).Result()
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}

				if count > int64(config.Requests) {
					w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Requests))
					w.Header().Set("X-RateLimit-Remaining", "0")
					w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(window+int64(config.Window.Seconds()), 10))
					http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
					return
				}

				remaining := config.Requests - int(count)
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Requests))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(window+int64(config.Window.Seconds()), 10))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getRealIP returns the real client IP, validating format and only trusting headers if behind a proxy
func getRealIP(r *http.Request) string {
	// TODO: Optionally, check if behind trusted proxy
	if ip := validateIP(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if ip := validateIP(r.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}
	return validateIP(r.RemoteAddr)
}

// validateIP checks if a string is a valid IP address
func validateIP(ip string) string {
	if ip == "" {
		return ""
	}
	parsed, _, err := net.SplitHostPort(ip)
	if err == nil {
		ip = parsed
	}
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

// IPRateLimit creates rate limiting based on client IP
func IPRateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimit(RateLimitConfig{
		Requests: requests,
		Window:   window,
		KeyFunc: func(r *http.Request) string {
			return getRealIP(r)
		},
	})
}

// AuthRateLimit creates stricter rate limiting for auth endpoints
func AuthRateLimit() func(http.Handler) http.Handler {
	return RateLimit(RateLimitConfig{
		Requests: 5,                // 5 attempts
		Window:   15 * time.Minute, // 15 minutes
		KeyFunc: func(r *http.Request) string {
			return "auth:" + getRealIP(r)
		},
	})
}

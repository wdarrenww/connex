package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connex/internal/cache"
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

// IPRateLimit creates rate limiting based on client IP
func IPRateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimit(RateLimitConfig{
		Requests: requests,
		Window:   window,
		KeyFunc: func(r *http.Request) string {
			// Get real IP considering proxies
			ip := r.Header.Get("X-Real-IP")
			if ip == "" {
				ip = r.Header.Get("X-Forwarded-For")
			}
			if ip == "" {
				ip = r.RemoteAddr
			}
			return ip
		},
	})
}

// AuthRateLimit creates stricter rate limiting for auth endpoints
func AuthRateLimit() func(http.Handler) http.Handler {
	return RateLimit(RateLimitConfig{
		Requests: 5,                // 5 attempts
		Window:   15 * time.Minute, // 15 minutes
		KeyFunc: func(r *http.Request) string {
			ip := r.Header.Get("X-Real-IP")
			if ip == "" {
				ip = r.Header.Get("X-Forwarded-For")
			}
			if ip == "" {
				ip = r.RemoteAddr
			}
			return fmt.Sprintf("auth:%s", ip)
		},
	})
}

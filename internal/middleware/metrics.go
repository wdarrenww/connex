package middleware

import (
	"net/http"
	"time"

	"connex/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MetricsMiddleware creates middleware that records HTTP metrics
func MetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer that captures the status code
			responseWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(responseWriter, r)

			// Record metrics
			duration := time.Since(start)
			endpoint := r.URL.Path
			if endpoint == "" {
				endpoint = "/"
			}

			telemetry.RecordHTTPRequest(r.Method, endpoint, responseWriter.statusCode, duration)
		})
	}
}

// TracingMiddleware creates middleware that adds OpenTelemetry tracing
func TracingMiddleware() func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware("connex-http",
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
		otelhttp.WithSpanOptions(
			trace.WithAttributes(
				attribute.String("http.method", ""),
				attribute.String("http.url", ""),
				attribute.String("http.user_agent", ""),
			),
		),
	)
}

// responseWriter captures the status code for metrics
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

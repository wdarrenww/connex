package telemetry

import (
	"context"
	"fmt"
	"time"

	"connex/internal/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	tracer trace.Tracer
	logger *zap.Logger

	// Prometheus metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	dbOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "table"},
	)

	dbOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	redisOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation"},
	)

	redisOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Redis operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	jobProcessingTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "job_processing_total",
			Help: "Total number of background jobs processed",
		},
		[]string{"job_type", "status"},
	)

	jobProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "job_processing_duration_seconds",
			Help:    "Background job processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"job_type"},
	)

	userRegistrationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "user_registrations_total",
			Help: "Total number of user registrations",
		},
	)

	activeUsersGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users_total",
			Help: "Total number of active users",
		},
	)

	securityEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "security_events_total",
			Help: "Total number of security events",
		},
		[]string{"event_type", "source"},
	)
)

// Init initializes OpenTelemetry tracing and metrics
func Init(cfg config.OTelConfig, log *zap.Logger) error {
	logger = log

	if !cfg.Enabled {
		logger.Info("OpenTelemetry disabled")
		return nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerURL)))
	if err != nil {
		return fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Create tracer
	tracer = tp.Tracer(cfg.ServiceName)

	logger.Info("OpenTelemetry initialized",
		zap.String("service", cfg.ServiceName),
		zap.String("environment", cfg.Environment),
		zap.String("jaeger_url", cfg.JaegerURL),
	)

	return nil
}

// GetTracer returns the global tracer
func GetTracer() trace.Tracer {
	return tracer
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name, opts...)
}

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	httpRequestsTotal.WithLabelValues(method, endpoint, fmt.Sprintf("%d", statusCode)).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordDBOperation records database operation metrics
func RecordDBOperation(operation, table string, duration time.Duration) {
	dbOperationsTotal.WithLabelValues(operation, table).Inc()
	dbOperationDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordRedisOperation records Redis operation metrics
func RecordRedisOperation(operation string, duration time.Duration) {
	redisOperationsTotal.WithLabelValues(operation).Inc()
	redisOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordJobProcessing records background job processing metrics
func RecordJobProcessing(jobType, status string, duration time.Duration) {
	jobProcessingTotal.WithLabelValues(jobType, status).Inc()
	jobProcessingDuration.WithLabelValues(jobType).Observe(duration.Seconds())
}

// RecordUserRegistration records user registration metrics
func RecordUserRegistration() {
	userRegistrationsTotal.Inc()
}

// SetActiveUsers sets the active users gauge
func SetActiveUsers(count int) {
	activeUsersGauge.Set(float64(count))
}

// LogWithTrace logs a message with trace context
func LogWithTrace(ctx context.Context, level string, msg string, fields ...zap.Field) {
	if span := trace.SpanFromContext(ctx); span != nil {
		fields = append(fields, zap.String("trace_id", span.SpanContext().TraceID().String()))
		fields = append(fields, zap.String("span_id", span.SpanContext().SpanID().String()))
	}

	switch level {
	case "debug":
		logger.Debug(msg, fields...)
	case "info":
		logger.Info(msg, fields...)
	case "warn":
		logger.Warn(msg, fields...)
	case "error":
		logger.Error(msg, fields...)
	default:
		logger.Info(msg, fields...)
	}
}

// RecordSecurityEvent records a security event
func RecordSecurityEvent(eventType string, source string) {
	// Record in metrics
	securityEventsTotal.WithLabelValues(eventType, source).Inc()

	// Log the event
	logger.Info("security event recorded",
		zap.String("event_type", eventType),
		zap.String("source", source),
		zap.Time("timestamp", time.Now().UTC()),
	)
}

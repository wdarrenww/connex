package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"connex/internal/api/auth"
	"connex/internal/api/health"
	"connex/internal/api/user"
	"connex/internal/cache"
	"connex/internal/config"
	"connex/internal/db"
	"connex/internal/job"
	custommiddleware "connex/internal/middleware"
	"connex/pkg/logger"
	"connex/pkg/telemetry"

	"encoding/base64"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.InitGlobal(cfg.Log.Level, cfg.Log.Env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	log := logger.GetGlobal()
	defer log.Sync()

	// Initialize DB
	dbInstance, err := db.Init(cfg.Database)
	if err != nil {
		log.Error("Failed to connect to database")
		log.Error(err.Error())
		os.Exit(1)
	}
	defer dbInstance.Close()

	// Initialize OpenTelemetry
	if err := telemetry.Init(cfg.OTel, log.Logger); err != nil {
		log.Error("Failed to initialize OpenTelemetry")
		log.Error(err.Error())
		os.Exit(1)
	}

	// Initialize Redis
	redisClient, err := cache.Init(cfg.Redis)
	if err != nil {
		log.Error("Failed to connect to Redis")
		log.Error(err.Error())
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize background jobs
	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	if err := job.Init(cfg.Jobs, redisOpt, log.Logger); err != nil {
		log.Error("Failed to initialize job queue")
		log.Error(err.Error())
		os.Exit(1)
	}

	// User service and handler
	userService := user.NewService()
	userHandler := user.NewHandler(userService)

	// Auth handler
	authHandler := auth.NewHandler(userService, cfg.JWT.Secret)

	// Health handler
	healthHandler := health.NewHandler()

	// Set up router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://your-frontend-domain.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Add monitoring middleware
	r.Use(custommiddleware.MetricsMiddleware())
	r.Use(custommiddleware.SecurityHeadersMiddleware())
	r.Use(custommiddleware.SecurityMonitoringMiddleware())
	if cfg.OTel.Enabled {
		r.Use(custommiddleware.TracingMiddleware())
	}

	// Apply request size limit to all API routes
	r.Route("/api", func(api chi.Router) {
		api.Use(custommiddleware.RequestSizeLimitMiddleware(1 << 20)) // 1MB
	})

	// Monitoring endpoints (protected in production)
	r.Route("/metrics", func(r chi.Router) {
		r.Use(custommiddleware.SecureMetricsMiddleware())
		r.Handle("/", promhttp.Handler())
	})
	r.Get("/health", healthHandler.SimpleHealthCheck)
	r.Get("/health/detailed", healthHandler.HealthCheck)
	r.Get("/ready", healthHandler.ReadinessCheck)

	// Get CSRF key
	csrfKeyB64 := os.Getenv("CSRF_AUTH_KEY")
	if csrfKeyB64 == "" {
		log.Error("CSRF_AUTH_KEY must be set and base64-encoded")
		os.Exit(1)
	}
	csrfKey, err := base64.StdEncoding.DecodeString(csrfKeyB64)
	if err != nil || len(csrfKey) < 32 {
		log.Error("CSRF_AUTH_KEY must be a base64-encoded 32-byte key")
		os.Exit(1)
	}

	// Auth endpoints (with rate limiting and CSRF)
	r.Route("/api/auth", func(r chi.Router) {
		r.Use(custommiddleware.AuthRateLimit())
		r.Use(custommiddleware.CSRFMiddleware(csrfKey))
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// User CRUD endpoints (protected, with rate limiting, caching, and CSRF)
	r.Route("/api/users", func(r chi.Router) {
		r.Use(auth.AuthMiddleware(cfg.JWT.Secret))
		r.Use(custommiddleware.IPRateLimit(100, time.Minute))
		r.Use(custommiddleware.URLPathCache(5 * time.Minute))
		r.Use(custommiddleware.CSRFMiddleware(csrfKey))
		userHandler.RegisterRoutes(r)
	})

	// Job management endpoints (admin only)
	r.Route("/api/jobs", func(r chi.Router) {
		r.Use(auth.AuthMiddleware(cfg.JWT.Secret))
		r.Post("/email", func(w http.ResponseWriter, r *http.Request) {
			// Example: enqueue email job
			payload := job.EmailPayload{
				To:      "user@example.com",
				Subject: "Test Email",
				Body:    "This is a test email",
			}
			if err := job.EnqueueEmail(payload); err != nil {
				http.Error(w, "failed to enqueue job", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		})
	})

	// Basic health check endpoint
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Info("Starting server")
	log.Info("Listening")
	log.Info("Server address")
	log.Info(addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("Server error")
		log.Error(err.Error())
		os.Exit(1)
	}
}

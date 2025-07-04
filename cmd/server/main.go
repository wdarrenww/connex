package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"connex/internal/api/auth"
	"connex/internal/api/user"
	"connex/internal/config"
	"connex/internal/db"
	"connex/pkg/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

	// User service and handler
	userService := user.NewService()
	userHandler := user.NewHandler(userService)

	// Auth handler
	authHandler := auth.NewHandler(userService, cfg.JWT.Secret)

	// Set up router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://your-frontend-domain.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Auth endpoints
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// User CRUD endpoints (protected)
	r.Route("/api/users", func(r chi.Router) {
		r.Use(auth.AuthMiddleware(cfg.JWT.Secret))
		userHandler.RegisterRoutes(r)
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

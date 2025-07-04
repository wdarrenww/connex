package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"connex/internal/cache"
	"connex/internal/db"
)

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Services  map[string]ServiceInfo `json:"services"`
	System    SystemInfo             `json:"system"`
}

type ServiceInfo struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

type SystemInfo struct {
	MemoryUsage string `json:"memory_usage"`
	Goroutines  int    `json:"goroutines"`
	NumCPU      int    `json:"num_cpu"`
	GoVersion   string `json:"go_version"`
}

var startTime = time.Now()

// Handler handles health check requests
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// HealthCheck provides a comprehensive health check
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime).String(),
		Services:  make(map[string]ServiceInfo),
		System:    h.getSystemInfo(),
	}

	// Check database
	dbStart := time.Now()
	dbErr := h.checkDatabase()
	dbLatency := time.Since(dbStart)
	if dbErr != nil {
		response.Status = "degraded"
		response.Services["database"] = ServiceInfo{
			Status:  "unhealthy",
			Message: dbErr.Error(),
			Latency: dbLatency.String(),
		}
	} else {
		response.Services["database"] = ServiceInfo{
			Status:  "healthy",
			Latency: dbLatency.String(),
		}
	}

	// Check Redis
	redisStart := time.Now()
	redisErr := h.checkRedis()
	redisLatency := time.Since(redisStart)
	if redisErr != nil {
		response.Status = "degraded"
		response.Services["redis"] = ServiceInfo{
			Status:  "unhealthy",
			Message: redisErr.Error(),
			Latency: redisLatency.String(),
		}
	} else {
		response.Services["redis"] = ServiceInfo{
			Status:  "healthy",
			Latency: redisLatency.String(),
		}
	}

	// Set appropriate status code
	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// SimpleHealthCheck provides a basic health check for load balancers
func (h *Handler) SimpleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// ReadinessCheck checks if the service is ready to accept traffic
func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// Check critical dependencies
	if err := h.checkDatabase(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "not ready",
			"message": "database unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}

func (h *Handler) checkDatabase() error {
	db := db.Get()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.Ping()
}

func (h *Handler) checkRedis() error {
	return cache.HealthCheck()
}

func (h *Handler) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		MemoryUsage: fmt.Sprintf("%d MB", m.Alloc/1024/1024),
		Goroutines:  runtime.NumGoroutine(),
		NumCPU:      runtime.NumCPU(),
		GoVersion:   runtime.Version(),
	}
}

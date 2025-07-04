package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler handles admin API requests
type Handler struct {
	logger *zap.Logger
}

// NewHandler creates a new admin handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// RegisterRoutes registers admin routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/dashboard", h.getDashboardData)
		r.Get("/users", h.getUsers)
		r.Get("/analytics", h.getAnalytics)
		r.Get("/system", h.getSystemStatus)
		r.Get("/logs", h.getLogs)
		r.Get("/metrics", h.getMetrics)
	})
}

// DashboardData represents the dashboard overview data
type DashboardData struct {
	Stats     DashboardStats  `json:"stats"`
	Charts    DashboardCharts `json:"charts"`
	Recent    DashboardRecent `json:"recent"`
	Activity  []ActivityItem  `json:"activity"`
	Timestamp time.Time       `json:"timestamp"`
}

// DashboardStats represents key metrics
type DashboardStats struct {
	TotalUsers    int     `json:"total_users"`
	ActiveUsers   int     `json:"active_users"`
	TotalRevenue  float64 `json:"total_revenue"`
	TotalOrders   int     `json:"total_orders"`
	UserGrowth    float64 `json:"user_growth"`
	RevenueGrowth float64 `json:"revenue_growth"`
	OrderGrowth   float64 `json:"order_growth"`
}

// DashboardCharts represents chart data
type DashboardCharts struct {
	UserActivity []ChartPoint `json:"user_activity"`
	SystemHealth SystemHealth `json:"system_health"`
}

// ChartPoint represents a data point for charts
type ChartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// SystemHealth represents system health metrics
type SystemHealth struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Disk    float64 `json:"disk"`
	Network float64 `json:"network"`
}

// DashboardRecent represents recent data
type DashboardRecent struct {
	Users    []UserSummary  `json:"users"`
	Orders   []OrderSummary `json:"orders"`
	Activity []ActivityItem `json:"activity"`
}

// UserSummary represents a user summary
type UserSummary struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	LastLogin time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderSummary represents an order summary
type OrderSummary struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ActivityItem represents an activity log item
type ActivityItem struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// getDashboardData returns comprehensive dashboard data
func (h *Handler) getDashboardData(w http.ResponseWriter, r *http.Request) {
	// In a real application, this would fetch data from the database
	// For now, we'll return mock data
	data := DashboardData{
		Stats: DashboardStats{
			TotalUsers:    1247,
			ActiveUsers:   892,
			TotalRevenue:  45231.50,
			TotalOrders:   3456,
			UserGrowth:    12.5,
			RevenueGrowth: 15.2,
			OrderGrowth:   -2.1,
		},
		Charts: DashboardCharts{
			UserActivity: []ChartPoint{
				{Label: "Mon", Value: 65},
				{Label: "Tue", Value: 59},
				{Label: "Wed", Value: 80},
				{Label: "Thu", Value: 81},
				{Label: "Fri", Value: 56},
				{Label: "Sat", Value: 55},
				{Label: "Sun", Value: 40},
			},
			SystemHealth: SystemHealth{
				CPU:     65.0,
				Memory:  45.0,
				Disk:    30.0,
				Network: 80.0,
			},
		},
		Recent: DashboardRecent{
			Users: []UserSummary{
				{
					ID:        1,
					Name:      "John Doe",
					Email:     "john@example.com",
					Status:    "active",
					LastLogin: time.Now().Add(-2 * time.Hour),
					CreatedAt: time.Now().AddDate(0, -1, 0),
				},
				{
					ID:        2,
					Name:      "Jane Smith",
					Email:     "jane@example.com",
					Status:    "active",
					LastLogin: time.Now().Add(-1 * time.Hour),
					CreatedAt: time.Now().AddDate(0, -2, 0),
				},
				{
					ID:        3,
					Name:      "Bob Johnson",
					Email:     "bob@example.com",
					Status:    "inactive",
					LastLogin: time.Now().AddDate(0, 0, -3),
					CreatedAt: time.Now().AddDate(0, -3, 0),
				},
			},
			Orders: []OrderSummary{
				{
					ID:        1,
					UserID:    1,
					Amount:    299.99,
					Status:    "completed",
					CreatedAt: time.Now().Add(-1 * time.Hour),
				},
				{
					ID:        2,
					UserID:    2,
					Amount:    149.50,
					Status:    "pending",
					CreatedAt: time.Now().Add(-2 * time.Hour),
				},
			},
		},
		Activity: []ActivityItem{
			{
				ID:        1,
				Type:      "login",
				User:      "John Doe",
				Action:    "logged in",
				Details:   "User logged in from 192.168.1.100",
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
			{
				ID:        2,
				Type:      "create",
				User:      "Jane Smith",
				Action:    "created a new account",
				Details:   "New user registration",
				Timestamp: time.Now().Add(-5 * time.Minute),
			},
			{
				ID:        3,
				Type:      "update",
				User:      "Bob Johnson",
				Action:    "updated their profile",
				Details:   "Profile information updated",
				Timestamp: time.Now().Add(-10 * time.Minute),
			},
		},
		Timestamp: time.Now(),
	}

	h.respondJSON(w, http.StatusOK, data)
}

// getUsers returns user management data
func (h *Handler) getUsers(w http.ResponseWriter, r *http.Request) {
	// Mock user data
	users := []UserSummary{
		{
			ID:        1,
			Name:      "John Doe",
			Email:     "john@example.com",
			Status:    "active",
			LastLogin: time.Now().Add(-2 * time.Hour),
			CreatedAt: time.Now().AddDate(0, -1, 0),
		},
		{
			ID:        2,
			Name:      "Jane Smith",
			Email:     "jane@example.com",
			Status:    "active",
			LastLogin: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now().AddDate(0, -2, 0),
		},
		{
			ID:        3,
			Name:      "Bob Johnson",
			Email:     "bob@example.com",
			Status:    "inactive",
			LastLogin: time.Now().AddDate(0, 0, -3),
			CreatedAt: time.Now().AddDate(0, -3, 0),
		},
		{
			ID:        4,
			Name:      "Alice Brown",
			Email:     "alice@example.com",
			Status:    "pending",
			LastLogin: time.Time{},
			CreatedAt: time.Now().AddDate(0, 0, -1),
		},
		{
			ID:        5,
			Name:      "Charlie Wilson",
			Email:     "charlie@example.com",
			Status:    "active",
			LastLogin: time.Now().Add(-30 * time.Minute),
			CreatedAt: time.Now().AddDate(0, -1, -15),
		},
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": len(users),
	})
}

// getAnalytics returns analytics data
func (h *Handler) getAnalytics(w http.ResponseWriter, r *http.Request) {
	// Mock analytics data
	analytics := map[string]interface{}{
		"user_growth": []ChartPoint{
			{Label: "Jan", Value: 100},
			{Label: "Feb", Value: 150},
			{Label: "Mar", Value: 200},
			{Label: "Apr", Value: 250},
			{Label: "May", Value: 300},
			{Label: "Jun", Value: 350},
		},
		"revenue_trend": []ChartPoint{
			{Label: "Jan", Value: 5000},
			{Label: "Feb", Value: 7500},
			{Label: "Mar", Value: 10000},
			{Label: "Apr", Value: 12500},
			{Label: "May", Value: 15000},
			{Label: "Jun", Value: 17500},
		},
		"top_products": []map[string]interface{}{
			{"name": "Product A", "sales": 150, "revenue": 7500},
			{"name": "Product B", "sales": 120, "revenue": 6000},
			{"name": "Product C", "sales": 100, "revenue": 5000},
		},
		"user_demographics": map[string]interface{}{
			"age_groups": map[string]int{
				"18-25": 30,
				"26-35": 45,
				"36-45": 15,
				"46+":   10,
			},
			"locations": map[string]int{
				"US":    40,
				"EU":    30,
				"Asia":  20,
				"Other": 10,
			},
		},
	}

	h.respondJSON(w, http.StatusOK, analytics)
}

// getSystemStatus returns system status information
func (h *Handler) getSystemStatus(w http.ResponseWriter, r *http.Request) {
	// Mock system status data
	status := map[string]interface{}{
		"system": map[string]interface{}{
			"cpu_usage":     65.0,
			"memory_usage":  45.0,
			"disk_usage":    30.0,
			"network_usage": 80.0,
			"uptime":        "15 days, 3 hours, 27 minutes",
			"load_average":  []float64{1.2, 1.1, 0.9},
		},
		"services": []map[string]interface{}{
			{"name": "Web Server", "status": "healthy", "uptime": "99.9%"},
			{"name": "Database", "status": "healthy", "uptime": "99.8%"},
			{"name": "Redis Cache", "status": "healthy", "uptime": "99.9%"},
			{"name": "Job Queue", "status": "warning", "uptime": "98.5%"},
		},
		"security": map[string]interface{}{
			"last_scan":       time.Now().Add(-6 * time.Hour),
			"vulnerabilities": 0,
			"failed_logins":   12,
			"blocked_ips":     3,
		},
	}

	h.respondJSON(w, http.StatusOK, status)
}

// getLogs returns system logs
func (h *Handler) getLogs(w http.ResponseWriter, r *http.Request) {
	// Mock log data
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-1 * time.Minute),
			"level":     "INFO",
			"message":   "User login successful",
			"user_id":   1,
			"ip":        "192.168.1.100",
		},
		{
			"timestamp": time.Now().Add(-2 * time.Minute),
			"level":     "WARN",
			"message":   "High memory usage detected",
			"details":   "Memory usage: 85%",
		},
		{
			"timestamp": time.Now().Add(-3 * time.Minute),
			"level":     "ERROR",
			"message":   "Database connection failed",
			"details":   "Connection timeout after 30 seconds",
		},
		{
			"timestamp": time.Now().Add(-5 * time.Minute),
			"level":     "INFO",
			"message":   "New user registration",
			"user_id":   5,
			"email":     "charlie@example.com",
		},
		{
			"timestamp": time.Now().Add(-10 * time.Minute),
			"level":     "INFO",
			"message":   "Backup completed successfully",
			"details":   "Database backup: 2.3GB",
		},
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	})
}

// getMetrics returns system metrics
func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	// Mock metrics data
	metrics := map[string]interface{}{
		"http_requests": map[string]interface{}{
			"total":             15420,
			"success":           15200,
			"errors":            220,
			"avg_response_time": 245,
		},
		"database": map[string]interface{}{
			"connections":     25,
			"queries_per_sec": 150,
			"slow_queries":    3,
		},
		"cache": map[string]interface{}{
			"hit_rate":     85.5,
			"miss_rate":    14.5,
			"memory_usage": "512MB",
		},
		"websocket": map[string]interface{}{
			"active_connections": 45,
			"messages_per_min":   120,
			"rooms":              8,
		},
	}

	h.respondJSON(w, http.StatusOK, metrics)
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

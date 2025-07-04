package ssr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"connex/pkg/logger"

	"go.uber.org/zap"
)

// SSRData represents data that can be injected into SSR templates
type SSRData struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	User        map[string]interface{} `json:"user,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	State       map[string]interface{} `json:"state,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// Handler handles server-side rendering
type Handler struct {
	templates map[string]*template.Template
	logger    *logger.Logger
	basePath  string
}

// NewHandler creates a new SSR handler
func NewHandler(templatePath string) *Handler {
	return &Handler{
		templates: make(map[string]*template.Template),
		logger:    logger.GetGlobal(),
		basePath:  templatePath,
	}
}

// LoadTemplate loads and caches a template
func (h *Handler) LoadTemplate(name string) error {
	templatePath := filepath.Join(h.basePath, name+".html")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	h.templates[name] = tmpl
	return nil
}

// RenderTemplate renders a template with data
func (h *Handler) RenderTemplate(w http.ResponseWriter, name string, data SSRData) error {
	tmpl, exists := h.templates[name]
	if !exists {
		// Try to load template if not cached
		if err := h.LoadTemplate(name); err != nil {
			return err
		}
		tmpl = h.templates[name]
	}

	// Convert data to JSON for client-side hydration
	stateJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Add state to template data
	templateData := map[string]interface{}{
		"Title":       data.Title,
		"Description": data.Description,
		"User":        data.User,
		"Meta":        data.Meta,
		"State":       data.State,
		"Config":      data.Config,
		"StateJSON":   template.JS(string(stateJSON)),
	}

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Set headers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write response
	_, err = w.Write(buf.Bytes())
	return err
}

// RenderSPA renders a single-page application with SSR data injection
func (h *Handler) RenderSPA(w http.ResponseWriter, r *http.Request, data SSRData) error {
	// Read the base HTML template
	indexPath := filepath.Join(h.basePath, "index.html")
	tmpl, err := template.ParseFiles(indexPath)
	if err != nil {
		return fmt.Errorf("failed to parse index template: %w", err)
	}

	// Convert data to JSON for client-side hydration
	stateJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Prepare template data
	templateData := map[string]interface{}{
		"Title":       data.Title,
		"Description": data.Description,
		"User":        data.User,
		"Meta":        data.Meta,
		"State":       data.State,
		"Config":      data.Config,
		"StateJSON":   template.JS(string(stateJSON)),
		"Path":        r.URL.Path,
	}

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Set headers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write response
	_, err = w.Write(buf.Bytes())
	return err
}

// Middleware creates SSR middleware for route-specific rendering
func (h *Handler) Middleware(routeData map[string]SSRData) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if route has SSR data
			if data, exists := routeData[r.URL.Path]; exists {
				if err := h.RenderSPA(w, r, data); err != nil {
					h.logger.Error("SSR rendering failed", zap.String("error", err.Error()), zap.String("path", r.URL.Path))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				return
			}

			// Continue to next handler if no SSR data
			next.ServeHTTP(w, r)
		})
	}
}

// InjectState injects state into the response for client-side hydration
func (h *Handler) InjectState(w http.ResponseWriter, data SSRData) error {
	stateJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	w.Header().Set("X-SSR-State", string(stateJSON))
	return nil
}

// GetStateFromRequest extracts SSR state from request headers
func (h *Handler) GetStateFromRequest(r *http.Request) (SSRData, error) {
	stateHeader := r.Header.Get("X-SSR-State")
	if stateHeader == "" {
		return SSRData{}, nil
	}

	var data SSRData
	if err := json.Unmarshal([]byte(stateHeader), &data); err != nil {
		return SSRData{}, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return data, nil
}

// CreateDefaultData creates default SSR data
func CreateDefaultData() SSRData {
	return SSRData{
		Title:       "Connex - Full Stack Web Application",
		Description: "A modern full-stack web application with Go backend and WebSocket support",
		Meta: map[string]interface{}{
			"viewport":    "width=device-width, initial-scale=1.0",
			"theme-color": "#667eea",
		},
		Config: map[string]interface{}{
			"apiBase": "/api",
			"wsUrl":   "/ws",
		},
	}
}

// CreateUserData creates SSR data with user information
func CreateUserData(user map[string]interface{}) SSRData {
	data := CreateDefaultData()
	data.User = user
	return data
}

// CreateRouteData creates SSR data for specific routes
func CreateRouteData(route string, additionalData map[string]interface{}) SSRData {
	data := CreateDefaultData()

	// Route-specific data
	switch {
	case strings.HasPrefix(route, "/dashboard"):
		data.Title = "Dashboard - Connex"
		data.Description = "User dashboard and analytics"
	case strings.HasPrefix(route, "/chat"):
		data.Title = "Chat - Connex"
		data.Description = "Real-time chat application"
	case strings.HasPrefix(route, "/profile"):
		data.Title = "Profile - Connex"
		data.Description = "User profile and settings"
	}

	// Merge additional data
	for key, value := range additionalData {
		data.State[key] = value
	}

	return data
}

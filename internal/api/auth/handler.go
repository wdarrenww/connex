package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"connex/internal/api/middleware"
	"connex/internal/api/user"
	"connex/pkg/jwt"
	"connex/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	UserService user.Service
	JWTSecret   string
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

func NewHandler(userService user.Service, jwtSecret string) *Handler {
	return &Handler{UserService: userService, JWTSecret: jwtSecret}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		middleware.WriteError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if err := validateEmail(req.Email); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateName(req.Name); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePassword(req.Password); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	newUser := &user.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	created, err := h.UserService.Create(r.Context(), newUser)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	logger.GetGlobal().Info("user registered",
		zap.Int64("user_id", created.ID),
		zap.String("email", created.Email),
		zap.String("ip", r.RemoteAddr),
		zap.String("action", "register"),
		zap.Time("timestamp", time.Now().UTC()),
	)
	token, err := jwt.GenerateJWT(created.ID, h.JWTSecret)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	resp := AuthResponse{Token: token, User: created}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		middleware.WriteError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if err := validatePassword(req.Password); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	found, err := h.UserService.GetByEmail(r.Context(), req.Email)
	if err != nil {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(found.PasswordHash), []byte(req.Password)); err != nil {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	logger.GetGlobal().Info("user login",
		zap.Int64("user_id", found.ID),
		zap.String("email", found.Email),
		zap.String("ip", r.RemoteAddr),
		zap.String("action", "login"),
		zap.Time("timestamp", time.Now().UTC()),
	)
	token, err := jwt.GenerateJWT(found.ID, h.JWTSecret)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	resp := AuthResponse{Token: token, User: found}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// validatePassword enforces strong password policy
func validatePassword(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return fmt.Errorf("password must contain upper, lower, digit, and special character")
	}
	if isCommonPassword(password) {
		return fmt.Errorf("password is too common")
	}
	return nil
}

// isCommonPassword checks against a small list of common passwords (expand in production)
func isCommonPassword(password string) bool {
	common := []string{"password", "123456", "qwerty", "letmein", "admin", "welcome", "iloveyou"}
	for _, p := range common {
		if strings.EqualFold(password, p) {
			return true
		}
	}
	return false
}

// validateEmail checks for a valid email format (basic RFC 5322)
func validateEmail(email string) error {
	if len(email) > 254 {
		return fmt.Errorf("email too long")
	}
	// Basic RFC 5322 regex
	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// validateName checks for a safe, reasonable name (no scripts, no dangerous chars, reasonable length)
func validateName(name string) error {
	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}
	// Disallow script tags and dangerous characters
	if strings.Contains(strings.ToLower(name), "<script") {
		return fmt.Errorf("name contains forbidden content")
	}
	var dangerous = regexp.MustCompile(`[<>"'\\/;]`)
	if dangerous.MatchString(name) {
		return fmt.Errorf("name contains forbidden characters")
	}
	return nil
}

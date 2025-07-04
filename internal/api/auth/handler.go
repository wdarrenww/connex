package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"connex/internal/api/middleware"
	"connex/internal/api/user"
	"connex/pkg/jwt"

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
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" || len(req.Password) < 8 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid input")
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
	if strings.TrimSpace(req.Email) == "" || len(req.Password) < 8 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid input")
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
	token, err := jwt.GenerateJWT(found.ID, h.JWTSecret)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	resp := AuthResponse{Token: token, User: found}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

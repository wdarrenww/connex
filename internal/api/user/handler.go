package user

import (
	"encoding/json"
	"net/http"
	"strconv"

	"connex/internal/api/middleware"

	"github.com/go-chi/chi/v5"
)

// Handler struct for dependency injection
// (add service here if needed)
type Handler struct {
	Service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{Service: service}
}

// RegisterRoutes registers user routes to the router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.CreateUser)
	r.Get("/", h.ListUsers)
	r.Get("/{id}", h.GetUser)
	r.Put("/{id}", h.UpdateUser)
	r.Delete("/{id}", h.DeleteUser)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := u.Validate(); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := h.Service.Create(r.Context(), &u)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Service.List(r.Context())
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if userID, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = userID
	}
	user, err := h.Service.Get(r.Context(), id)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u.ID = id
	if err := u.Validate(); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := h.Service.Update(r.Context(), &u)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not update user")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Service.Delete(r.Context(), id); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

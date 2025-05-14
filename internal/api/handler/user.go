package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/middleware"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/pizza-nz/restaurant-service/internal/service"
)

// UserHandler handles user-related requests
type UserHandler struct {
	authService *service.AuthService
}

// NewUserHandler creates a new user handler
func NewUserHandler(authService *service.AuthService) *UserHandler {
	return &UserHandler{
		authService: authService,
	}
}

// HandleUsers handles requests for users
func (h *UserHandler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from path
	path := strings.TrimPrefix(r.URL.Path, "/users")
	path = strings.TrimPrefix(path, "/")

	// Check for special endpoints
	if strings.Contains(path, "/password") {
		h.changePassword(w, r)
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.listUsers(w, r)
		} else {
			id, err := uuid.Parse(path)
			if err != nil {
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}
			h.getUser(w, r, id)
		}

	case http.MethodPost:
		if path != "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		h.createUser(w, r)

	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		h.updateUser(w, r, id)

	case http.MethodDelete:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		h.deleteUser(w, r, id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listUsers lists all users
func (h *UserHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, users)
}

// getUser gets a user by ID
func (h *UserHandler) getUser(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	user, err := h.authService.GetUser(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, user)
}

// createUser creates a new user
func (h *UserHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var req models.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.authService.RegisterUser(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, user)
}

// updateUser updates a user
func (h *UserHandler) updateUser(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.authService.UpdateUser(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, user)
}

// deleteUser deletes a user
func (h *UserHandler) deleteUser(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.authService.DeleteUser(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// changePassword changes the current user's password
func (h *UserHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from context
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusInternalServerError)
		return
	}

	err = h.authService.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: "Password changed successfully",
	})
}

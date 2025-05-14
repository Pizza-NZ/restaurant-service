package handler

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/api"
	"github.com/pizza-nz/restaurant-service/internal/service"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

// MenuHandler handles menu-related requests
type MenuHandler struct {
	menuService *service.MenuService
	hub         *websockets.Hub
}

// NewMenuHandler creates a new menu handler
func NewMenuHandler(menuService *service.MenuService, hub *websockets.Hub) *MenuHandler {
	return &MenuHandler{
		menuService: menuService,
		hub:         hub,
	}
}

// HandleMenuCategories handles requests from menu categories
func (h *MenuHandler) HandleMenuCategories(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/menu/categories")
	path = strings.TrimPrefix(path, "/")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		// if path empty then listCategory(w, r)
		if path == "" {
			h.listCategories(w, r)
		}
		// else parse path as id
		// check if err then BadRequest if true
		// finally getCategory(w, r, id)
	case http.MethodPost:
		// if path not empty then BadRequest
		// else createCategory(w, r)
	case http.MethodDelete:
		// parse path as id
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid category ID")
		}
		// finally deleteCategory(w, r, id)
	default:
		// Return MethodNotAllowed
	}
}

// listCategories lsit all menu categories
func (h *MenuHandler) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.menuService.GetCategories(r.Context())
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	respondJSON(w, categories)
}

// getCategory gets a menu category
func (h *MenuHandler) getCategory(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	category, err := h.menuService.GetCategory(r.Context(), id)
	if err != nil {
		api.NotFound(w, err)
		return
	}

	respondJSON(w, category)
}

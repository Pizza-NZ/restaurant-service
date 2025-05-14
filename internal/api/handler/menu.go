package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/api"
	"github.com/pizza-nz/restaurant-service/internal/models"
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

// broadcastMenuUpdate broadcasts a menu update to all connected clients
func (h *MenuHandler) broadcastMenuUpdate(updateType, id string) {
	// Create a message
	message := struct {
		Type string `json:"type"`
		Data struct {
			UpdateType string `json:"update_type"`
			ID         string `json:"id"`
		} `json:"data"`
	}{
		Type: "menu.update",
		Data: struct {
			UpdateType string `json:"update_type"`
			ID         string `json:"id"`
		}{
			UpdateType: updateType,
			ID:         id,
		},
	}

	// Marshal the message to JSON
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return
	}

	// Broadcast to all clients
	h.hub.BroadcastMessage(jsonMessage)
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
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid category ID")
			return
		}
		// finally getCategory(w, r, id)
		h.getCategory(w, r, id)
	case http.MethodPut:
		// parse path as id
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid category ID")
			return
		}
		h.updateCategory(w, r, id)
	case http.MethodPost:
		// if path not empty then BadRequest
		if path != "" {
			api.BadRequest(w, "Invalid request path")
			return
		}
		// else createCategory(w, r)
		h.createCategory(w, r)
	case http.MethodDelete:
		// parse path as id
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid category ID")
			return
		}
		// finally deleteCategory(w, r, id)
		h.deleteCategory(w, r, id)
	default:
		// Return MethodNotAllowed
		api.MethodNotAllowed(w)
		return
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

// createCategory creates a new menu category
func (h *MenuHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	var req models.MenuCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	category, err := h.menuService.CreateCategory(r.Context(), req)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("category.create", category.ID.String())

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, category)
}

// updateCategory updates a menu category
func (h *MenuHandler) updateCategory(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.MenuCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	category, err := h.menuService.UpdateCategory(r.Context(), id, req)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("category.update", category.ID.String())

	respondJSON(w, category)
}

// deleteCategory deletes a menu category
func (h *MenuHandler) deleteCategory(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.menuService.DeleteCategory(r.Context(), id)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("category.delete", id.String())

	w.WriteHeader(http.StatusNoContent)
	respondJSON(w, nil)
}

// HandleMenuItems handles requests from menu items
func (h *MenuHandler) HandleMenuItems(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/menu/items")
	path = strings.TrimPrefix(path, "/")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		// if path empty then listItems(w, r)
		if path == "" {
			h.listItems(w, r)
		}
		// else parse path as id
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid item ID")
			return
		}
		// finally getItem(w, r, id)
		h.getItem(w, r, id)
	case http.MethodPut:
		// parse path as id
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid item ID")
			return
		}
		h.updateItem(w, r, id)
	case http.MethodPost:
		// if path not empty then BadRequest
		if path != "" {
			api.BadRequest(w, "Invalid request path")
			return
		}
		h.createItem(w, r)
	case http.MethodDelete:
		id, err := uuid.Parse(path)
		if err != nil {
			api.BadRequest(w, "Invalid item ID")
			return
		}
		h.deleteItem(w, r, id)
	default:
		api.MethodNotAllowed(w)
		return
	}
}

// listItems lists menu items, optionally filtered by category
func (h *MenuHandler) listItems(w http.ResponseWriter, r *http.Request) {
	var category *uuid.UUID

	categoryIDStr := r.URL.Query().Get("category_id")
	if categoryIDStr != "" {
		id, err := uuid.Parse(categoryIDStr)
		if err != nil {
			api.BadRequest(w, "Invalid category ID")
			return
		}
		category = &id
	}

	items, err := h.menuService.GetItems(r.Context(), category)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	respondJSON(w, items)
}

// getItem gets a menu item by ID
func (h *MenuHandler) getItem(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	item, err := h.menuService.GetItem(r.Context(), id)
	if err != nil {
		api.NotFound(w, err)
		return
	}

	respondJSON(w, item)
}

// createItem creates a new menu item
func (h *MenuHandler) createItem(w http.ResponseWriter, r *http.Request) {
	var req models.MenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body "+err.Error())
		return
	}

	item, err := h.menuService.CreateItem(r.Context(), req)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("item.create", item.ID.String())

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, item)
}

// updateItem updates a menu item
func (h *MenuHandler) updateItem(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.MenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	item, err := h.menuService.UpdateItem(r.Context(), id, req)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("item.update", item.ID.String())

	respondJSON(w, item)
}

// deleteItem deletes a menu item
func (h *MenuHandler) deleteItem(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.menuService.DeleteItem(r.Context(), id)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("item.delete", id.String())

	w.WriteHeader(http.StatusNoContent)
	respondJSON(w, nil)
}

// HandleModifiers handles requests for modifiers
func (h *MenuHandler) HandleModifiers(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/modifiers")
	path = strings.TrimPrefix(path, "/")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		// if path empty then listModifiers(w, r)
		if path == "" {
			h.listModifiers(w, r)
		}
		// else parse path as id
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid modifier ID")
			return
		}
		// finally getModifier(w, r, id)
		h.getModifier(w, r, id)
	case http.MethodPut:
		// parse path as id
		id, err := uuid.Parse(path)
		// check if err then BadRequest if true
		if err != nil {
			api.BadRequest(w, "Invalid modifier ID")
			return
		}
		h.updateModifier(w, r, id)
	case http.MethodPost:
		// if path not empty then BadRequest
		if path != "" {
			api.BadRequest(w, "Invalid request path")
			return
		}
		h.createModifier(w, r)
	case http.MethodDelete:
		id, err := uuid.Parse(path)
		if err != nil {
			api.BadRequest(w, "Invalid modifier ID")
			return
		}
		h.deleteModifier(w, r, id)
	default:
		api.MethodNotAllowed(w)
		return
	}
}

// listModifiers lists all modifiers
func (h *MenuHandler) listModifiers(w http.ResponseWriter, r *http.Request) {
	modifiers, err := h.menuService.GetModifiers(r.Context())
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	respondJSON(w, modifiers)
}

// getModifier gets a modifier by ID
func (h *MenuHandler) getModifier(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	modifier, err := h.menuService.GetModifier(r.Context(), id)
	if err != nil {
		api.NotFound(w, err)
		return
	}

	respondJSON(w, modifier)
}

// createModifier creates a new modifier
func (h *MenuHandler) createModifier(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string                         `json:"name"`
		IsMultiple bool                           `json:"is_multiple"`
		Options    []models.ModifierOptionRequest `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	modifier, err := h.menuService.CreateModifier(r.Context(), req.Name, req.IsMultiple, req.Options)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("modifier.create", modifier.ID.String())

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, modifier)
}

// updateModifier updates a modifier
func (h *MenuHandler) updateModifier(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Name       string                         `json:"name"`
		IsMultiple bool                           `json:"is_multiple"`
		Options    []models.ModifierOptionRequest `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "Invalid request body")
		return
	}

	modifier, err := h.menuService.UpdateModifier(r.Context(), id, req.Name, req.IsMultiple, req.Options)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("modifier.update", modifier.ID.String())

	respondJSON(w, modifier)
}

// deleteModifier deletes a modifier
func (h *MenuHandler) deleteModifier(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.menuService.DeleteModifier(r.Context(), id)
	if err != nil {
		api.InternalServerError(w, err)
		return
	}

	// Broadcast menu update to connected clients
	h.broadcastMenuUpdate("modifier.delete", id.String())

	w.WriteHeader(http.StatusNoContent)
	respondJSON(w, nil)
}

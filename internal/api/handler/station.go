package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/pizza-nz/restaurant-service/internal/service"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

// StationHandler handles station-related requests
type StationHandler struct {
	stationService *service.StationService
	hub            *websockets.Hub
}

// NewStationHandler creates a new station handler
func NewStationHandler(stationService *service.StationService, hub *websockets.Hub) *StationHandler {
	return &StationHandler{
		stationService: stationService,
		hub:            hub,
	}
}

// HandleStations handles requests for stations
func (h *StationHandler) HandleStations(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from path
	path := strings.TrimPrefix(r.URL.Path, "/stations")
	path = strings.TrimPrefix(path, "/")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.listStations(w, r)
		} else {
			id, err := uuid.Parse(path)
			if err != nil {
				http.Error(w, "Invalid station ID", http.StatusBadRequest)
				return
			}
			h.getStation(w, r, id)
		}

	case http.MethodPost:
		if path != "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		h.createStation(w, r)

	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid station ID", http.StatusBadRequest)
			return
		}
		h.updateStation(w, r, id)

	case http.MethodDelete:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid station ID", http.StatusBadRequest)
			return
		}
		h.deleteStation(w, r, id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listStations lists all stations
func (h *StationHandler) listStations(w http.ResponseWriter, r *http.Request) {
	stations, err := h.stationService.GetStations(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, stations)
}

// getStation gets a station by ID
func (h *StationHandler) getStation(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	station, err := h.stationService.GetStation(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, station)
}

// createStation creates a new station
func (h *StationHandler) createStation(w http.ResponseWriter, r *http.Request) {
	var req models.StationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	station, err := h.stationService.CreateStation(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast station update to connected clients
	h.broadcastStationUpdate("station.created", station.ID.String())

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, station)
}

// updateStation updates a station
func (h *StationHandler) updateStation(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.StationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	station, err := h.stationService.UpdateStation(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast station update to connected clients
	h.broadcastStationUpdate("station.updated", id.String())

	respondJSON(w, station)
}

// deleteStation deletes a station
func (h *StationHandler) deleteStation(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.stationService.DeleteStation(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast station update to connected clients
	h.broadcastStationUpdate("station.deleted", id.String())

	w.WriteHeader(http.StatusNoContent)
}

// HandleRoutingRules handles requests for routing rules
func (h *StationHandler) HandleRoutingRules(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from path
	path := strings.TrimPrefix(r.URL.Path, "/routing")
	path = strings.TrimPrefix(path, "/")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		if path == "" {
			http.Error(w, "Menu item ID required", http.StatusBadRequest)
			return
		}

		menuItemID, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid menu item ID", http.StatusBadRequest)
			return
		}

		h.getRoutingRules(w, r, menuItemID)

	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid routing rule ID", http.StatusBadRequest)
			return
		}

		h.updateRoutingRule(w, r, id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getRoutingRules gets routing rules for a menu item
func (h *StationHandler) getRoutingRules(w http.ResponseWriter, r *http.Request, menuItemID uuid.UUID) {
	rules, err := h.stationService.GetRoutingRules(r.Context(), menuItemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, rules)
}

// updateRoutingRule updates a routing rule
func (h *StationHandler) updateRoutingRule(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.RoutingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rule, err := h.stationService.UpdateRoutingRule(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast routing update to connected clients
	h.broadcastStationUpdate("routing.updated", id.String())

	respondJSON(w, rule)
}

// broadcastStationUpdate broadcasts a station update to all connected clients
func (h *StationHandler) broadcastStationUpdate(updateType, id string) {
	// Create a message
	message := struct {
		Type string `json:"type"`
		Data struct {
			UpdateType string `json:"update_type"`
			ID         string `json:"id"`
		} `json:"data"`
	}{
		Type: "station.update",
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

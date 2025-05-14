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

// PrinterHandler handles printer-related requests
type PrinterHandler struct {
	printerService *service.PrinterService
	hub            *websockets.Hub
}

// NewPrinterHandler creates a new printer handler
func NewPrinterHandler(printerService *service.PrinterService, hub *websockets.Hub) *PrinterHandler {
	return &PrinterHandler{
		printerService: printerService,
		hub:            hub,
	}
}

// HandlePrinters handles requests for printers
func (h *PrinterHandler) HandlePrinters(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from path
	path := strings.TrimPrefix(r.URL.Path, "/printers")
	path = strings.TrimPrefix(path, "/")

	// Check for test endpoint
	if strings.Contains(path, "/test") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			id, err := uuid.Parse(parts[0])
			if err != nil {
				http.Error(w, "Invalid printer ID", http.StatusBadRequest)
				return
			}
			h.testPrinter(w, r, id)
			return
		}
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.listPrinters(w, r)
		} else {
			id, err := uuid.Parse(path)
			if err != nil {
				http.Error(w, "Invalid printer ID", http.StatusBadRequest)
				return
			}
			h.getPrinter(w, r, id)
		}

	case http.MethodPost:
		if path != "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		h.createPrinter(w, r)

	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid printer ID", http.StatusBadRequest)
			return
		}
		h.updatePrinter(w, r, id)

	case http.MethodDelete:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid printer ID", http.StatusBadRequest)
			return
		}
		h.deletePrinter(w, r, id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listPrinters lists all printers
func (h *PrinterHandler) listPrinters(w http.ResponseWriter, r *http.Request) {
	printers, err := h.printerService.GetPrinters(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, printers)
}

// getPrinter gets a printer by ID
func (h *PrinterHandler) getPrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	printer, err := h.printerService.GetPrinter(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, printer)
}

// createPrinter creates a new printer
func (h *PrinterHandler) createPrinter(w http.ResponseWriter, r *http.Request) {
	var req models.PrinterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	printer, err := h.printerService.CreatePrinter(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast printer update to connected clients
	h.broadcastPrinterUpdate("printer.created", printer.ID.String())

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, printer)
}

// updatePrinter updates a printer
func (h *PrinterHandler) updatePrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.PrinterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	printer, err := h.printerService.UpdatePrinter(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast printer update to connected clients
	h.broadcastPrinterUpdate("printer.updated", id.String())

	respondJSON(w, printer)
}

// deletePrinter deletes a printer
func (h *PrinterHandler) deletePrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.printerService.DeletePrinter(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast printer update to connected clients
	h.broadcastPrinterUpdate("printer.deleted", id.String())

	w.WriteHeader(http.StatusNoContent)
}

// testPrinter tests a printer
func (h *PrinterHandler) testPrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.printerService.TestPrinter(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: "Test page sent successfully",
	})
}

// HandleDisplays handles requests for displays
func (h *PrinterHandler) HandleDisplays(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from path
	path := strings.TrimPrefix(r.URL.Path, "/displays")
	path = strings.TrimPrefix(path, "/")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.listDisplays(w, r)
		} else {
			id, err := uuid.Parse(path)
			if err != nil {
				http.Error(w, "Invalid display ID", http.StatusBadRequest)
				return
			}
			h.getDisplay(w, r, id)
		}

	case http.MethodPost:
		if path != "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		h.createDisplay(w, r)

	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid display ID", http.StatusBadRequest)
			return
		}
		h.updateDisplay(w, r, id)

	case http.MethodDelete:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid display ID", http.StatusBadRequest)
			return
		}
		h.deleteDisplay(w, r, id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listDisplays lists all displays
func (h *PrinterHandler) listDisplays(w http.ResponseWriter, r *http.Request) {
	displays, err := h.printerService.GetDisplays(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, displays)
}

// getDisplay gets a display by ID
func (h *PrinterHandler) getDisplay(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	display, err := h.printerService.GetDisplay(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, display)
}

// createDisplay creates a new display
func (h *PrinterHandler) createDisplay(w http.ResponseWriter, r *http.Request) {
	var req models.DisplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	display, err := h.printerService.CreateDisplay(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast display update to connected clients
	h.broadcastPrinterUpdate("display.created", display.ID.String())

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, display)
}

// updateDisplay updates a display
func (h *PrinterHandler) updateDisplay(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.DisplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	display, err := h.printerService.UpdateDisplay(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast display update to connected clients
	h.broadcastPrinterUpdate("display.updated", id.String())

	respondJSON(w, display)
}

// deleteDisplay deletes a display
func (h *PrinterHandler) deleteDisplay(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.printerService.DeleteDisplay(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast display update to connected clients
	h.broadcastPrinterUpdate("display.deleted", id.String())

	w.WriteHeader(http.StatusNoContent)
}

// broadcastPrinterUpdate broadcasts a printer/display update to all connected clients
func (h *PrinterHandler) broadcastPrinterUpdate(updateType, id string) {
	// Create a message
	message := struct {
		Type string `json:"type"`
		Data struct {
			UpdateType string `json:"update_type"`
			ID         string `json:"id"`
		} `json:"data"`
	}{
		Type: "printer.update",
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

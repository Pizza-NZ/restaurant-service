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

// HandlePrinters is the main router for /api/printers endpoints.
func (h *PrinterHandler) HandlePrinters(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/printers")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	idStr := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	// Route for listing all printers or creating a new one
	if idStr == "" {
		switch r.Method {
		case http.MethodGet:
			h.listPrinters(w, r)
		case http.MethodPost:
			h.createPrinter(w, r)
		default:
			api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	// Routes that require a printer ID
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid printer ID")
		return
	}

	if action == "test" && r.Method == http.MethodPost {
		h.testPrinter(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getPrinter(w, r, id)
	case http.MethodPut:
		h.updatePrinter(w, r, id)
	case http.MethodDelete:
		h.deletePrinter(w, r, id)
	default:
		api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleOrderReceipt handles GETting receipt text and POSTing for reprints.
func (h *PrinterHandler) HandleOrderReceipt(w http.ResponseWriter, r *http.Request) {
	orderIDStr := r.PathValue("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getOrderReceiptText(w, r, orderID)
	case http.MethodPost:
		h.reprintOrderReceipt(w, r, orderID)
	default:
		api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// listPrinters lists all printers
func (h *PrinterHandler) listPrinters(w http.ResponseWriter, r *http.Request) {
	printers, err := h.printerService.GetPrinters(r.Context())
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to list printers")
		return
	}
	api.JSON(w, http.StatusOK, printers)
}

// getPrinter gets a printer by ID
func (h *PrinterHandler) getPrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	printer, err := h.printerService.GetPrinter(r.Context(), id)
	if err != nil {
		api.Error(w, http.StatusNotFound, "Printer not found")
		return
	}
	api.JSON(w, http.StatusOK, printer)
}

// createPrinter creates a new printer
func (h *PrinterHandler) createPrinter(w http.ResponseWriter, r *http.Request) {
	var req models.PrinterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	printer, err := h.printerService.CreatePrinter(r.Context(), req)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to create printer")
		return
	}

	h.broadcastDeviceUpdate("printer.created", printer.ID.String())
	api.JSON(w, http.StatusCreated, printer)
}

// updatePrinter updates a printer
func (h *PrinterHandler) updatePrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.PrinterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	printer, err := h.printerService.UpdatePrinter(r.Context(), id, req)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to update printer")
		return
	}
	h.broadcastDeviceUpdate("printer.updated", id.String())
	api.JSON(w, http.StatusOK, printer)
}

// deletePrinter deletes a printer
func (h *PrinterHandler) deletePrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.printerService.DeletePrinter(r.Context(), id)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to delete printer")
		return
	}
	h.broadcastDeviceUpdate("printer.deleted", id.String())
	w.WriteHeader(http.StatusNoContent)
}

// testPrinter tests a printer
func (h *PrinterHandler) testPrinter(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.printerService.TestPrinter(r.Context(), id)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to send test page to printer")
		return
	}
	api.JSON(w, http.StatusOK, map[string]string{"message": "Test page sent successfully"})
}

// getOrderReceiptText gets a receipt for an order
func (h *PrinterHandler) getOrderReceiptText(w http.ResponseWriter, r *http.Request, orderID uuid.UUID) {
	receipt, err := h.printerService.GenerateReceiptText(r.Context(), orderID)
	if err != nil {
		// Check if the error indicates the order was not found
		if strings.Contains(err.Error(), "failed to get order") {
			api.Error(w, http.StatusNotFound, "Order not found")
			return
		}
		api.Error(w, http.StatusInternalServerError, "Could not generate receipt text")
		return
	}
	api.JSON(w, http.StatusOK, map[string]string{"receipt": receipt})
}

// reprintOrderReceipt reprints a receipt for an order
func (h *PrinterHandler) reprintOrderReceipt(w http.ResponseWriter, r *http.Request, orderID uuid.UUID) {
	var req struct {
		PrinterID string `json:"printer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body: requires printer_id")
		return
	}

	printerID, err := uuid.Parse(req.PrinterID)
	if err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid printer ID")
		return
	}

	err = h.printerService.ReprintReceipt(r.Context(), orderID, printerID)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to reprint receipt")
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Reprint request sent successfully"})
}

// HandleDisplays handles requests for displays
func (h *PrinterHandler) HandleDisplays(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/displays")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.listDisplays(w, r)
		case http.MethodPost:
			h.createDisplay(w, r)
		default:
			api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	id, err := uuid.Parse(path)
	if err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid display ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getDisplay(w, r, id)
	case http.MethodPut:
		h.updateDisplay(w, r, id)
	case http.MethodDelete:
		h.deleteDisplay(w, r, id)
	default:
		api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// listDisplays lists all displays
func (h *PrinterHandler) listDisplays(w http.ResponseWriter, r *http.Request) {
	displays, err := h.printerService.GetDisplays(r.Context())
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to list displays")
		return
	}
	api.JSON(w, http.StatusOK, displays)
}

// getDisplay gets a display by ID
func (h *PrinterHandler) getDisplay(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	display, err := h.printerService.GetDisplay(r.Context(), id)
	if err != nil {
		api.Error(w, http.StatusNotFound, "Display not found")
		return
	}
	api.JSON(w, http.StatusOK, display)
}

// createDisplay creates a new display
func (h *PrinterHandler) createDisplay(w http.ResponseWriter, r *http.Request) {
	var req models.DisplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	display, err := h.printerService.CreateDisplay(r.Context(), req)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to create display")
		return
	}

	h.broadcastDeviceUpdate("display.created", display.ID.String())
	api.JSON(w, http.StatusCreated, display)
}

// updateDisplay updates a display
func (h *PrinterHandler) updateDisplay(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req models.DisplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	display, err := h.printerService.UpdateDisplay(r.Context(), id, req)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to update display")
		return
	}

	h.broadcastDeviceUpdate("display.updated", id.String())
	api.JSON(w, http.StatusOK, display)
}

// deleteDisplay deletes a display
func (h *PrinterHandler) deleteDisplay(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.printerService.DeleteDisplay(r.Context(), id)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to delete display")
		return
	}
	h.broadcastDeviceUpdate("display.deleted", id.String())
	w.WriteHeader(http.StatusNoContent)
}

// broadcastDeviceUpdate broadcasts a printer/display update to all connected clients
func (h *PrinterHandler) broadcastDeviceUpdate(updateType, id string) {
	message := map[string]interface{}{
		"type": "device.update",
		"data": map[string]string{
			"update_type": updateType,
			"id":          id,
		},
	}
	jsonMessage, _ := json.Marshal(message)
	h.hub.BroadcastMessage(jsonMessage)
}

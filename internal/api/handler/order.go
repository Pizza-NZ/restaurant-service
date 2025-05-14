package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/middleware"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/pizza-nz/restaurant-service/internal/service"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

// OrderHandler handles order-related requests
type OrderHandler struct {
	orderService *service.OrderService
	hub          *websockets.Hub
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService *service.OrderService, hub *websockets.Hub) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		hub:          hub,
	}
}

// HandleOrders handles requests for orders
func (h *OrderHandler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from path
	path := strings.TrimPrefix(r.URL.Path, "/orders")
	path = strings.TrimPrefix(path, "/")

	// Check for special endpoints
	if strings.HasPrefix(path, "history") {
		h.handleOrderHistory(w, r)
		return
	} else if strings.Contains(path, "/receipt") {
		// Extract order ID from path
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		id, err := uuid.Parse(parts[0])
		if err != nil {
			http.Error(w, "Invalid order ID", http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodGet {
			h.getOrderReceipt(w, r, id)
		} else if r.Method == http.MethodPost && parts[1] == "reprint" {
			h.reprintOrderReceipt(w, r, id)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.listOrders(w, r)
		} else {
			id, err := uuid.Parse(path)
			if err != nil {
				http.Error(w, "Invalid order ID", http.StatusBadRequest)
				return
			}
			h.getOrder(w, r, id)
		}

	case http.MethodPost:
		if path != "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		h.createOrder(w, r)

	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid order ID", http.StatusBadRequest)
			return
		}
		h.updateOrderStatus(w, r, id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleOrderItems handles requests for order items
func (h *OrderHandler) HandleOrderItems(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from path
	path := strings.TrimPrefix(r.URL.Path, "/order-items")
	path = strings.TrimPrefix(path, "/")

	// Check for special endpoints
	if strings.Contains(path, "/void") {
		// Extract item ID from path
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		id, err := uuid.Parse(parts[0])
		if err != nil {
			http.Error(w, "Invalid item ID", http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodPut {
			h.voidOrderItem(w, r, id)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			http.Error(w, "Invalid item ID", http.StatusBadRequest)
			return
		}
		h.updateOrderItemStatus(w, r, id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listOrders lists orders, optionally filtered by status
func (h *OrderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	var status *models.OrderStatus

	statusStr := r.URL.Query().Get("status")
	if statusStr != "" {
		s := models.OrderStatus(statusStr)
		status = &s
	}

	orders, err := h.orderService.ListOrders(r.Context(), status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, orders)
}

// getOrder gets an order by ID
func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	order, err := h.orderService.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, order)
}

// createOrder creates a new order
func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req models.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the user ID from context
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

	order, err := h.orderService.CreateOrder(r.Context(), req, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast order creation to connected clients
	h.broadcastOrderUpdate("order.new", order.ID.String())

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, order)
}

// updateOrderStatus updates an order's status
func (h *OrderHandler) updateOrderStatus(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	status := models.OrderStatus(req.Status)

	err := h.orderService.UpdateOrderStatus(r.Context(), id, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast order update to connected clients
	h.broadcastOrderUpdate("order.update", id.String())

	w.WriteHeader(http.StatusOK)
}

// updateOrderItemStatus updates an order item's status
func (h *OrderHandler) updateOrderItemStatus(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	status := models.OrderItemStatus(req.Status)

	err := h.orderService.UpdateOrderItemStatus(r.Context(), id, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast item update to connected clients
	h.broadcastOrderUpdate("item.update", id.String())

	w.WriteHeader(http.StatusOK)
}

// voidOrderItem voids an order item
func (h *OrderHandler) voidOrderItem(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.orderService.VoidOrderItem(r.Context(), id, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast item update to connected clients
	h.broadcastOrderUpdate("item.void", id.String())

	w.WriteHeader(http.StatusOK)
}

// handleOrderHistory handles order history requests
func (h *OrderHandler) handleOrderHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse date range from query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr == "" {
		// Default to the beginning of today
		startDate = time.Now().Truncate(24 * time.Hour)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			http.Error(w, "Invalid start_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	}

	if endDateStr == "" {
		// Default to the end of today
		endDate = startDate.Add(24 * time.Hour).Add(-time.Second)
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		// Set to end of the day
		endDate = endDate.Add(24 * time.Hour).Add(-time.Second)
	}

	orders, err := h.orderService.GetOrderHistory(r.Context(), startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, orders)
}

// getOrderReceipt gets a receipt for an order
func (h *OrderHandler) getOrderReceipt(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	receipt, err := h.orderService.GetOrderReceipt(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Receipt string `json:"receipt"`
	}{
		Receipt: receipt,
	}

	respondJSON(w, response)
}

// reprintOrderReceipt reprints a receipt for an order
func (h *OrderHandler) reprintOrderReceipt(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		PrinterID string `json:"printer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	printerID, err := uuid.Parse(req.PrinterID)
	if err != nil {
		http.Error(w, "Invalid printer ID", http.StatusBadRequest)
		return
	}

	err = h.orderService.ReprintOrderReceipt(r.Context(), id, printerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	respondJSON(w, struct {
		Success bool `json:"success"`
	}{
		Success: true,
	})
}

// HandleStationItems handles requests for station items
func (h *OrderHandler) HandleStationItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the station ID from the path
	path := strings.TrimPrefix(r.URL.Path, "/stations/")
	path = strings.TrimPrefix(path, "/items")

	stationID, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "Invalid station ID", http.StatusBadRequest)
		return
	}

	items, err := h.orderService.GetStationItems(r.Context(), stationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, items)
}

// broadcastOrderUpdate broadcasts an order update to all connected clients
func (h *OrderHandler) broadcastOrderUpdate(updateType, id string) {
	// Create a message
	message := struct {
		Type string `json:"type"`
		Data struct {
			UpdateType string `json:"update_type"`
			ID         string `json:"id"`
		} `json:"data"`
	}{
		Type: "order.update",
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

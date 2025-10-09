package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/api"
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
	path := strings.TrimPrefix(r.URL.Path, "/api/orders")
	path = strings.TrimPrefix(path, "/")

	if strings.HasPrefix(path, "history") {
		h.handleOrderHistory(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.listOrders(w, r)
		} else {
			id, err := uuid.Parse(path)
			if err != nil {
				api.Error(w, http.StatusBadRequest, "Invalid order ID")
				return
			}
			h.getOrder(w, r, id)
		}
	case http.MethodPost:
		if path != "" {
			api.Error(w, http.StatusBadRequest, "Invalid path")
			return
		}
		h.createOrder(w, r)
	case http.MethodPut:
		id, err := uuid.Parse(path)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid order ID")
			return
		}
		h.updateOrderStatus(w, r, id)
	default:
		api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleOrderItems handles requests for order items
func (h *OrderHandler) HandleOrderItems(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/order-items")
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")
	itemIDStr := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	if r.Method == http.MethodPut && action == "void" {
		h.voidOrderItem(w, r, itemID)
		return
	} else if r.Method == http.MethodPut {
		h.updateOrderItemStatus(w, r, itemID)
		return
	}

	api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
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
		api.Error(w, http.StatusInternalServerError, "Failed to list orders")
		return
	}

	api.JSON(w, http.StatusOK, orders)
}

// getOrder gets an order by ID
func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	order, err := h.orderService.GetOrder(r.Context(), id)
	if err != nil {
		api.Error(w, http.StatusNotFound, "Order not found")
		return
	}

	api.JSON(w, http.StatusOK, order)
}

// createOrder creates a new order
func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req models.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Error(w, http.StatusInternalServerError, "User ID not found in context")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Invalid user ID in context")
		return
	}

	order, err := h.orderService.CreateOrder(r.Context(), req, userID)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to create order")
		return
	}

	h.broadcastOrderUpdate("order.new", order.ID.String())
	api.JSON(w, http.StatusCreated, order)
}

// updateOrderStatus updates an order's status
func (h *OrderHandler) updateOrderStatus(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.orderService.UpdateOrderStatus(r.Context(), id, models.OrderStatus(req.Status))
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to update order status")
		return
	}

	h.broadcastOrderUpdate("order.update", id.String())
	w.WriteHeader(http.StatusOK)
}

// updateOrderItemStatus updates an order item's status
func (h *OrderHandler) updateOrderItemStatus(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.orderService.UpdateOrderItemStatus(r.Context(), id, models.OrderItemStatus(req.Status))
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to update order item status")
		return
	}

	h.broadcastOrderUpdate("item.update", id.String())
	w.WriteHeader(http.StatusOK)
}

// voidOrderItem voids an order item
func (h *OrderHandler) voidOrderItem(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.orderService.VoidOrderItem(r.Context(), id, req.Reason)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to void order item")
		return
	}

	h.broadcastOrderUpdate("item.void", id.String())
	w.WriteHeader(http.StatusOK)
}

// handleOrderHistory handles order history requests
func (h *OrderHandler) handleOrderHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr == "" {
		startDate = time.Now().Truncate(24 * time.Hour)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid start_date format (use YYYY-MM-DD)")
			return
		}
	}

	if endDateStr == "" {
		endDate = startDate.Add(24 * time.Hour).Add(-time.Second)
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid end_date format (use YYYY-MM-DD)")
			return
		}
		endDate = endDate.Add(24 * time.Hour).Add(-time.Second)
	}

	orders, err := h.orderService.GetOrderHistory(r.Context(), startDate, endDate)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to get order history")
		return
	}

	api.JSON(w, http.StatusOK, orders)
}

// HandleStationItems handles requests for station items
func (h *OrderHandler) HandleStationItems(w http.ResponseWriter, r *http.Request) {
	stationIDStr := r.PathValue("id")
	stationID, err := uuid.Parse(stationIDStr)
	if err != nil {
		api.Error(w, http.StatusBadRequest, "Invalid station ID in path")
		return
	}

	items, err := h.orderService.GetStationItems(r.Context(), stationID)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "Failed to get station items")
		return
	}

	api.JSON(w, http.StatusOK, items)
}

// broadcastOrderUpdate broadcasts an order update to all connected clients
func (h *OrderHandler) broadcastOrderUpdate(updateType, id string) {
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

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return
	}

	h.hub.BroadcastMessage(jsonMessage)
}

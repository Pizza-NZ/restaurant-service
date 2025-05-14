package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// OrderService handles order-related business logic
type OrderService struct {
	repos          *repository.Repositories
	printerService *PrinterService
}

// NewOrderService creates a new order service
func NewOrderService(repos *repository.Repositories, printerService *PrinterService) *OrderService {
	return &OrderService{
		repos:          repos,
		printerService: printerService,
	}
}

// GetOrder retrieves an order by ID
func (s *OrderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	return s.repos.Order.GetByID(ctx, id)
}

// ListOrders lists orders, optionally filtered by status
func (s *OrderService) ListOrders(ctx context.Context, status *models.OrderStatus) ([]models.Order, error) {
	return s.repos.Order.List(ctx, status)
}

// GetOrderHistory retrieves order history for a time range
func (s *OrderService) GetOrderHistory(ctx context.Context, startDate, endDate time.Time) ([]models.Order, error) {
	return s.repos.Order.GetOrderHistory(ctx, startDate, endDate)
}

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(ctx context.Context, req models.OrderRequest, userID uuid.UUID) (*models.Order, error) {
	// Generate order number based on current date and a random suffix
	orderNumber := time.Now().Format("20060102-") + uuid.New().String()[0:4]

	// Create the order
	order := models.Order{
		UserID:      userID,
		OrderNumber: orderNumber,
		Status:      models.OrderStatusNew,
		OrderedAt:   time.Now(),
	}

	// Create the order with items
	createdOrder, err := s.repos.Order.Create(ctx, order, req.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Process the order (route items to stations, print tickets)
	err = s.processNewOrder(ctx, createdOrder)
	if err != nil {
		// Log the error but don't fail the order creation
		// In a production system, you might want to handle this differently
		fmt.Printf("Error processing order: %v\n", err)
	}

	return createdOrder, nil
}

// processNewOrder routes items to stations and prints tickets
func (s *OrderService) processNewOrder(ctx context.Context, order *models.Order) error {
	// Update order status to in progress
	err := s.repos.Order.UpdateStatus(ctx, order.ID, models.OrderStatusInProgress)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Group items by station
	stationItems := make(map[uuid.UUID][]models.OrderItem)

	for _, item := range order.Items {
		stationItems[item.StationID] = append(stationItems[item.StationID], item)
	}

	// For each station, print a ticket
	for stationID, items := range stationItems {
		// Get the station
		station, err := s.repos.Station.GetByID(ctx, stationID)
		if err != nil {
			return fmt.Errorf("failed to get station: %w", err)
		}

		// If the station has a printer, print a ticket
		if station.PrinterID != nil {
			err = s.printerService.PrintOrderItems(ctx, order, items, *station.PrinterID)
			if err != nil {
				return fmt.Errorf("failed to print order items: %w", err)
			}
		}

		// Update items status to in progress
		for _, item := range items {
			err = s.repos.Order.UpdateItemStatus(ctx, item.ID, models.OrderItemStatusInProgress)
			if err != nil {
				return fmt.Errorf("failed to update item status: %w", err)
			}
		}
	}

	return nil
}

// UpdateOrderStatus updates an order's status
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error {
	return s.repos.Order.UpdateStatus(ctx, id, status)
}

// UpdateOrderItemStatus updates an order item's status
func (s *OrderService) UpdateOrderItemStatus(ctx context.Context, itemID uuid.UUID, status models.OrderItemStatus) error {
	return s.repos.Order.UpdateItemStatus(ctx, itemID, status)
}

// VoidOrderItem voids an order item
func (s *OrderService) VoidOrderItem(ctx context.Context, itemID uuid.UUID, reason string) error {
	return s.repos.Order.VoidItem(ctx, itemID, reason)
}

// GetStationItems gets all pending and in-progress items for a station
func (s *OrderService) GetStationItems(ctx context.Context, stationID uuid.UUID) ([]models.OrderItem, error) {
	return s.repos.Order.GetStationItems(ctx, stationID)
}

// ReprintOrderReceipt reprints an order receipt
func (s *OrderService) ReprintOrderReceipt(ctx context.Context, orderID uuid.UUID, printerID uuid.UUID) error {
	// Get the order
	order, err := s.repos.Order.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Print the receipt
	err = s.printerService.PrintReceipt(ctx, order, printerID)
	if err != nil {
		return fmt.Errorf("failed to print receipt: %w", err)
	}

	return nil
}

// GetOrderReceipt generates a receipt for an order
func (s *OrderService) GetOrderReceipt(ctx context.Context, orderID uuid.UUID) (string, error) {
	// Get the order
	order, err := s.repos.Order.GetByID(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("failed to get order: %w", err)
	}

	// Generate the receipt
	receipt, err := s.printerService.GenerateReceiptText(ctx, order)
	if err != nil {
		return "", fmt.Errorf("failed to generate receipt: %w", err)
	}

	return receipt, nil
}

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	orderCreationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "order_creation_duration_seconds",
		Help: "Duration of the CreateOrder function.",
	})
	ordersCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "orders_created_total",
		Help: "Total number of created orders.",
	})
)

// OrderRepository defines the full set of database methods needed by the OrderService.
type OrderRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	List(ctx context.Context, status *models.OrderStatus) ([]models.Order, error)
	Create(ctx context.Context, order models.Order, items []models.OrderItemRequest) (*models.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status models.OrderItemStatus) error
	VoidItem(ctx context.Context, itemID uuid.UUID, reason string) error
	GetStationItems(ctx context.Context, stationID uuid.UUID) ([]models.OrderItem, error)
	GetOrderHistory(ctx context.Context, startDate, endDate time.Time) ([]models.Order, error)
}

// The EventPublisher interface remains the same
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, orderID string) error
}

// OrderService handles order-related business logic.
// OrderService handles order-related business logic.
type OrderService struct {
	orderRepo OrderRepository // Use the interface type
	publisher EventPublisher  // Use the interface type
	logger    *slog.Logger
}

// NewOrderService creates a new order service.
func NewOrderService(repos OrderRepository, publisher EventPublisher) *OrderService { // Depend on interfaces
	return &OrderService{
		orderRepo: repos,
		publisher: publisher,
		logger:    slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

// GetOrder retrieves a single order by its ID.
func (s *OrderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	return s.orderRepo.GetByID(ctx, id)
}

// ListOrders retrieves a list of orders, optionally filtered by status.
func (s *OrderService) ListOrders(ctx context.Context, status *models.OrderStatus) ([]models.Order, error) {
	return s.orderRepo.List(ctx, status)
}

// GetOrderHistory retrieves orders within a specified date range.
func (s *OrderService) GetOrderHistory(ctx context.Context, startDate, endDate time.Time) ([]models.Order, error) {
	return s.orderRepo.GetOrderHistory(ctx, startDate, endDate)
}

// CreateOrder validates and saves a new order, then publishes an event for downstream processing.
func (s *OrderService) CreateOrder(ctx context.Context, req models.OrderRequest, userID uuid.UUID) (*models.Order, error) {
	startTime := time.Now()
	// Generate a unique, human-readable order number
	orderNumber := time.Now().Format("20060102-") + uuid.New().String()[0:4]

	order := models.Order{
		UserID:      userID,
		OrderNumber: orderNumber,
		Status:      models.OrderStatusNew,
		OrderedAt:   time.Now(),
	}

	// The repository handles the complex transaction of creating the order and all its items.
	createdOrder, err := s.orderRepo.Create(ctx, order, req.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to create order in database: %w", err)
	}

	// After the order is successfully saved, publish an event.
	// This decouples order creation from subsequent actions like printing.
	if err := s.publisher.PublishOrderCreated(ctx, createdOrder.ID.String()); err != nil {
		// This is a critical failure. The order is saved but won't be printed or sent to KDS.
		// Log this as a major error for monitoring and potential manual intervention.
		s.logger.Error("CRITICAL: Failed to publish order.created event", "order_id", createdOrder.ID, "error", err)
	}

	orderCreationDuration.Observe(time.Since(startTime).Seconds())
	ordersCreated.Inc()

	return createdOrder, nil
}

// UpdateOrderStatus updates the status of an entire order.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error {
	return s.orderRepo.UpdateStatus(ctx, id, status)
}

// UpdateOrderItemStatus updates the status of a single item within an order.
func (s *OrderService) UpdateOrderItemStatus(ctx context.Context, itemID uuid.UUID, status models.OrderItemStatus) error {
	return s.orderRepo.UpdateItemStatus(ctx, itemID, status)
}

// VoidOrderItem marks an item as void and adjusts the order total.
func (s *OrderService) VoidOrderItem(ctx context.Context, itemID uuid.UUID, reason string) error {
	return s.orderRepo.VoidItem(ctx, itemID, reason)
}

// GetStationItems retrieves all active items for a specific kitchen/bar station.
func (s *OrderService) GetStationItems(ctx context.Context, stationID uuid.UUID) ([]models.OrderItem, error) {
	return s.orderRepo.GetStationItems(ctx, stationID)
}

// GetOrderReceipt generates a formatted text receipt for an order.
// This is a read-only operation and is safe to keep in this service.
func (s *OrderService) GetOrderReceipt(ctx context.Context, orderID uuid.UUID) (string, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("failed to get order: %w", err)
	}
	return generateReceiptText(order)
}

// generateReceiptText is a helper to format order data into a string.
func generateReceiptText(order *models.Order) (string, error) {
	var sb strings.Builder

	sb.WriteString("===============================\n")
	sb.WriteString("          RECEIPT             \n")
	sb.WriteString("===============================\n\n")
	sb.WriteString(fmt.Sprintf("Order #: %s\n", order.OrderNumber))
	sb.WriteString(fmt.Sprintf("Date: %s\n", order.OrderedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString("\nItems:\n")
	sb.WriteString("-------------------------------\n")

	for _, item := range order.Items {
		sb.WriteString(fmt.Sprintf("%dx %s\n", item.Quantity, item.Name))
		for _, mod := range item.Modifiers {
			sb.WriteString(fmt.Sprintf("  + %s", mod.Name))
			if mod.PriceAdjustment > 0 {
				sb.WriteString(fmt.Sprintf(" (+$%.2f)", mod.PriceAdjustment))
			}
			sb.WriteString("\n")
		}
		if item.SpecialInstructions != nil && *item.SpecialInstructions != "" {
			sb.WriteString(fmt.Sprintf("  * %s\n", *item.SpecialInstructions))
		}
		sb.WriteString(fmt.Sprintf("  $%.2f\n\n", item.Price*float64(item.Quantity)))
	}

	sb.WriteString("-------------------------------\n")
	sb.WriteString(fmt.Sprintf("Total: $%.2f\n\n", order.Total))
	sb.WriteString("===============================\n")
	sb.WriteString("         Thank You!           \n")
	sb.WriteString("===============================\n")

	return sb.String(), nil
}

package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// OrderRepository handles order data access
type OrderRepository struct {
	db *sqlx.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	query := `
		SELECT id, user_id, order_number, status, total, ordered_at, completed_at, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var order models.Order
	err := r.db.GetContext(ctx, &order, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Get order items
	items, err := r.GetOrderItems(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	order.Items = items

	return &order, nil
}

// GetOrderItems retrieves items for an order
func (r *OrderRepository) GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]models.OrderItem, error) {
	query := `
		SELECT oi.id, oi.order_id, oi.menu_item_id, oi.station_id, oi.quantity, oi.price,
		       oi.status, oi.special_instructions, oi.sent_to_station_at, oi.completed_at, 
		       oi.created_at, oi.updated_at, 
		       mi.name as name
		FROM order_items oi
		JOIN menu_items mi ON oi.menu_item_id = mi.id
		WHERE oi.order_id = $1
		ORDER BY oi.created_at ASC
	`

	var items []models.OrderItem
	err := r.db.SelectContext(ctx, &items, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// For each item, get modifiers
	for i := range items {
		modifiers, err := r.GetOrderItemModifiers(ctx, items[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get item modifiers: %w", err)
		}
		items[i].Modifiers = modifiers
	}

	return items, nil
}

// GetOrderItemModifiers retrieves modifiers for an order item
func (r *OrderRepository) GetOrderItemModifiers(ctx context.Context, orderItemID uuid.UUID) ([]models.OrderItemModifier, error) {
	query := `
		SELECT oim.id, oim.order_item_id, oim.modifier_option_id, oim.price_adjustment, oim.created_at,
		       mo.name as name
		FROM order_item_modifiers oim
		JOIN modifier_options mo ON oim.modifier_option_id = mo.id
		WHERE oim.order_item_id = $1
	`

	var modifiers []models.OrderItemModifier
	err := r.db.SelectContext(ctx, &modifiers, query, orderItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order item modifiers: %w", err)
	}

	return modifiers, nil
}

// List retrieves orders, optionally filtered by status
func (r *OrderRepository) List(ctx context.Context, status *models.OrderStatus) ([]models.Order, error) {
	var query string
	var args []interface{}

	if status != nil {
		query = `
			SELECT id, user_id, order_number, status, total, ordered_at, completed_at, created_at, updated_at
			FROM orders
			WHERE status = $1
			ORDER BY ordered_at DESC
		`
		args = append(args, *status)
	} else {
		query = `
			SELECT id, user_id, order_number, status, total, ordered_at, completed_at, created_at, updated_at
			FROM orders
			ORDER BY ordered_at DESC
		`
	}

	// Apply a limit to avoid overwhelming the Pi
	query += " LIMIT 100"

	var orders []models.Order
	err := r.db.SelectContext(ctx, &orders, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	return orders, nil
}

// Create creates a new order with its items
func (r *OrderRepository) Create(ctx context.Context, order models.Order, itemRequests []models.OrderItemRequest) (*models.Order, error) {
	// Start a transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Insert the order
	orderQuery := `
		INSERT INTO orders (user_id, order_number, status, total, ordered_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, order_number, status, total, ordered_at, completed_at, created_at, updated_at
	`

	var createdOrder models.Order
	err = tx.GetContext(
		ctx,
		&createdOrder,
		orderQuery,
		order.UserID,
		order.OrderNumber,
		order.Status,
		order.Total,
		order.OrderedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Insert each order item
	createdOrder.Items = make([]models.OrderItem, 0, len(itemRequests))

	for _, itemReq := range itemRequests {
		// Get the menu item to determine routing
		var menuItem struct {
			Name string `db:"name"`
		}
		err = tx.GetContext(
			ctx,
			&menuItem,
			"SELECT name FROM menu_items WHERE id = $1",
			itemReq.MenuItemID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get menu item: %w", err)
		}

		// Get the routing station
		var stationID uuid.UUID
		err = tx.GetContext(
			ctx,
			&stationID,
			`SELECT station_id FROM routing_rules WHERE menu_item_id = $1 ORDER BY priority ASC LIMIT 1`,
			itemReq.MenuItemID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get routing station: %w", err)
		}

		// Insert the order item
		var createdItem models.OrderItem
		err = tx.GetContext(
			ctx,
			&createdItem,
			`INSERT INTO order_items 
			 (order_id, menu_item_id, station_id, quantity, price, status, special_instructions)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id, order_id, menu_item_id, station_id, quantity, price, status, 
			          special_instructions, sent_to_station_at, completed_at, created_at, updated_at`,
			createdOrder.ID,
			itemReq.MenuItemID,
			stationID,
			itemReq.Quantity,
			0.0, // We'll calculate the price after adding modifiers
			models.OrderItemStatusPending,
			itemReq.SpecialInstructions,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}

		// Set the item name from the menu item
		createdItem.Name = menuItem.Name

		// Get the base price from the menu item
		var basePrice float64
		err = tx.GetContext(
			ctx,
			&basePrice,
			"SELECT price FROM menu_items WHERE id = $1",
			itemReq.MenuItemID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get menu item price: %w", err)
		}

		// Calculate item price with modifiers
		price := basePrice

		// Add modifiers if any
		if len(itemReq.Modifiers) > 0 {
			createdItem.Modifiers = make([]models.OrderItemModifier, 0, len(itemReq.Modifiers))

			for _, mod := range itemReq.Modifiers {
				// Get the modifier option details
				var option struct {
					Name            string  `db:"name"`
					PriceAdjustment float64 `db:"price_adjustment"`
				}
				err = tx.GetContext(
					ctx,
					&option,
					"SELECT name, price_adjustment FROM modifier_options WHERE id = $1",
					mod.OptionID,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to get modifier option: %w", err)
				}

				// Add the price adjustment
				price += option.PriceAdjustment

				// Insert the order item modifier
				var createdMod models.OrderItemModifier
				err = tx.GetContext(
					ctx,
					&createdMod,
					`INSERT INTO order_item_modifiers 
					 (order_item_id, modifier_option_id, price_adjustment)
					 VALUES ($1, $2, $3)
					 RETURNING id, order_item_id, modifier_option_id, price_adjustment, created_at`,
					createdItem.ID,
					mod.OptionID,
					option.PriceAdjustment,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create order item modifier: %w", err)
				}

				createdMod.Name = option.Name
				createdItem.Modifiers = append(createdItem.Modifiers, createdMod)
			}
		}

		// Update the item price
		_, err = tx.ExecContext(
			ctx,
			"UPDATE order_items SET price = $1 WHERE id = $2",
			price,
			createdItem.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update order item price: %w", err)
		}

		createdItem.Price = price
		createdOrder.Items = append(createdOrder.Items, createdItem)

		// Update order total
		createdOrder.Total += price * float64(createdItem.Quantity)
	}

	// Update the order total
	_, err = tx.ExecContext(
		ctx,
		"UPDATE orders SET total = $1 WHERE id = $2",
		createdOrder.Total,
		createdOrder.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update order total: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &createdOrder, nil
}

// UpdateStatus updates an order's status
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
	`

	args := []interface{}{status, time.Now()}

	// If the status is completed, set the completed_at timestamp
	if status == models.OrderStatusCompleted {
		query += ", completed_at = $3 WHERE id = $4"
		now := time.Now()
		args = append(args, now, id)
	} else {
		query += " WHERE id = $3"
		args = append(args, id)
	}

	// internal/db/repository/order_repo.go (continued)
	// UpdateStatus continued
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("order not found")
	}

	return nil
}

// UpdateItemStatus updates an order item's status
func (r *OrderRepository) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status models.OrderItemStatus) error {
	query := `
		UPDATE order_items
		SET status = $1, updated_at = $2
	`

	args := []interface{}{status, time.Now()}

	// If the status is completed, set the completed_at timestamp
	if status == models.OrderItemStatusCompleted {
		query += ", completed_at = $3 WHERE id = $4"
		now := time.Now()
		args = append(args, now, itemID)
	} else if status == models.OrderItemStatusInProgress {
		// If the item is now in progress and wasn't sent to a station yet,
		// set the sent_to_station_at timestamp
		query += ", sent_to_station_at = CASE WHEN sent_to_station_at IS NULL THEN $3 ELSE sent_to_station_at END WHERE id = $4"
		now := time.Now()
		args = append(args, now, itemID)
	} else {
		query += " WHERE id = $3"
		args = append(args, itemID)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update order item status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("order item not found")
	}

	// Check if all items in the order are completed and update order status if needed
	if status == models.OrderItemStatusCompleted {
		// Get the order ID for this item
		var orderID uuid.UUID
		err = r.db.GetContext(
			ctx,
			&orderID,
			"SELECT order_id FROM order_items WHERE id = $1",
			itemID,
		)
		if err != nil {
			return fmt.Errorf("failed to get order ID for item: %w", err)
		}

		// Check if all items in the order are completed
		var pendingCount int
		err = r.db.GetContext(
			ctx,
			&pendingCount,
			`SELECT COUNT(*) FROM order_items 
			 WHERE order_id = $1 AND status != $2`,
			orderID, models.OrderItemStatusCompleted,
		)
		if err != nil {
			return fmt.Errorf("failed to check pending items: %w", err)
		}

		// If no pending items, mark the order as completed
		if pendingCount == 0 {
			err = r.UpdateStatus(ctx, orderID, models.OrderStatusCompleted)
			if err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}
		}
	}

	return nil
}

// GetStationItems gets all pending and in-progress items for a station
func (r *OrderRepository) GetStationItems(ctx context.Context, stationID uuid.UUID) ([]models.OrderItem, error) {
	query := `
		SELECT oi.id, oi.order_id, oi.menu_item_id, oi.station_id, oi.quantity, oi.price,
		       oi.status, oi.special_instructions, oi.sent_to_station_at, oi.completed_at, 
		       oi.created_at, oi.updated_at, 
		       mi.name as name,
		       o.order_number
		FROM order_items oi
		JOIN menu_items mi ON oi.menu_item_id = mi.id
		JOIN orders o ON oi.order_id = o.id
		WHERE oi.station_id = $1 
		  AND oi.status IN ($2, $3)
		  AND o.status IN ($4, $5)
		ORDER BY oi.sent_to_station_at ASC NULLS FIRST, oi.created_at ASC
	`

	var items []models.OrderItem
	err := r.db.SelectContext(
		ctx,
		&items,
		query,
		stationID,
		models.OrderItemStatusPending,
		models.OrderItemStatusInProgress,
		models.OrderStatusNew,
		models.OrderStatusInProgress,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get station items: %w", err)
	}

	// For each item, get modifiers
	for i := range items {
		modifiers, err := r.GetOrderItemModifiers(ctx, items[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get item modifiers: %w", err)
		}
		items[i].Modifiers = modifiers
	}

	return items, nil
}

// GetOrderHistory gets order history for a specified time range
func (r *OrderRepository) GetOrderHistory(ctx context.Context, startDate, endDate time.Time) ([]models.Order, error) {
	query := `
		SELECT id, user_id, order_number, status, total, ordered_at, completed_at, created_at, updated_at
		FROM orders
		WHERE ordered_at BETWEEN $1 AND $2
		ORDER BY ordered_at DESC
		LIMIT 500
	`

	var orders []models.Order
	err := r.db.SelectContext(ctx, &orders, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get order history: %w", err)
	}

	return orders, nil
}

// VoidItem voids an order item
func (r *OrderRepository) VoidItem(ctx context.Context, itemID uuid.UUID, reason string) error {
	// Start a transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Update the item status to cancelled
	_, err = tx.ExecContext(
		ctx,
		`UPDATE order_items 
		 SET status = $1, updated_at = $2, special_instructions = COALESCE(special_instructions, '') || E'\n[VOIDED: ' || $3 || ']'
		 WHERE id = $4`,
		models.OrderItemStatusCancelled,
		time.Now(),
		reason,
		itemID,
	)
	if err != nil {
		return fmt.Errorf("failed to void order item: %w", err)
	}

	// Get order ID and item price/quantity
	var orderInfo struct {
		OrderID  uuid.UUID `db:"order_id"`
		Price    float64   `db:"price"`
		Quantity int       `db:"quantity"`
	}
	err = tx.GetContext(
		ctx,
		&orderInfo,
		"SELECT order_id, price, quantity FROM order_items WHERE id = $1",
		itemID,
	)
	if err != nil {
		return fmt.Errorf("failed to get order info: %w", err)
	}

	// Update order total
	_, err = tx.ExecContext(
		ctx,
		"UPDATE orders SET total = total - $1, updated_at = $2 WHERE id = $3",
		orderInfo.Price*float64(orderInfo.Quantity),
		time.Now(),
		orderInfo.OrderID,
	)
	if err != nil {
		return fmt.Errorf("failed to update order total: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

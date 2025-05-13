package models

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "new"
	OrderStatusInProgress OrderStatus = "in_progress"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// OrderItemStatus represents the status of an order item
type OrderItemStatus string

const (
	OrderItemStatusPending    OrderItemStatus = "pending"
	OrderItemStatusInProgress OrderItemStatus = "in_progress"
	OrderItemStatusCompleted  OrderItemStatus = "completed"
	OrderItemStatusCancelled  OrderItemStatus = "cancelled"
)

// Order represents a customer order
type Order struct {
	ID          uuid.UUID   `db:"id" json:"id"`
	UserID      uuid.UUID   `db:"user_id" json:"user_id"`
	OrderNumber string      `db:"order_number" json:"order_number"`
	Status      OrderStatus `db:"status" json:"status"`
	Total       float64     `db:"total" json:"total"`
	OrderedAt   time.Time   `db:"ordered_at" json:"ordered_at"`
	CompletedAt *time.Time  `db:"completed_at" json:"completed_at"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updated_at"`

	// Not stored directly in the database
	Items []OrderItem `db:"-" json:"items,omitempty"`
	User  *User       `db:"-" json:"user,omitempty"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID                  uuid.UUID       `db:"id" json:"id"`
	OrderID             uuid.UUID       `db:"order_id" json:"order_id"`
	MenuItemID          uuid.UUID       `db:"menu_item_id" json:"menu_item_id"`
	StationID           uuid.UUID       `db:"station_id" json:"station_id"`
	Quantity            int             `db:"quantity" json:"quantity"`
	Price               float64         `db:"price" json:"price"`
	Status              OrderItemStatus `db:"status" json:"status"`
	SpecialInstructions *string         `db:"special_instructions" json:"special_instructions"`
	SentToStationAt     *time.Time      `db:"sent_to_station_at" json:"sent_to_station_at"`
	CompletedAt         *time.Time      `db:"completed_at" json:"completed_at"`
	CreatedAt           time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at" json:"updated_at"`

	// Not stored directly in the database
	Name      string              `db:"-" json:"name"`
	Modifiers []OrderItemModifier `db:"-" json:"modifiers,omitempty"`
	Station   *Station            `db:"-" json:"station,omitempty"`
}

// OrderItemModifier represents a modifier applied to an order item
type OrderItemModifier struct {
	ID               uuid.UUID `db:"id" json:"id"`
	OrderItemID      uuid.UUID `db:"order_item_id" json:"order_item_id"`
	ModifierOptionID uuid.UUID `db:"modifier_option_id" json:"modifier_option_id"`
	PriceAdjustment  float64   `db:"price_adjustment" json:"price_adjustment"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`

	// Not stored directly in the database
	Name string `db:"-" json:"name"`
}

// OrderRequest is used for order creation
type OrderRequest struct {
	Items []OrderItemRequest `json:"items" validate:"required,min=1,dive"`
}

// OrderItemRequest is used for order item creation
type OrderItemRequest struct {
	MenuItemID          uuid.UUID              `json:"menu_item_id" validate:"required"`
	Quantity            int                    `json:"quantity" validate:"required,min=1"`
	SpecialInstructions *string                `json:"special_instructions"`
	Modifiers           []OrderModifierRequest `json:"modifiers"`
}

// OrderModifierRequest is used for order item modifier creation
type OrderModifierRequest struct {
	OptionID uuid.UUID `json:"option_id" validate:"required"`
}

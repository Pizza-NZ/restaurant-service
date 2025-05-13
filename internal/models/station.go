package models

import (
	"time"

	"github.com/google/uuid"
)

// StationType represents a station type
type StationType string

const (
	StationTypeKitchen StationType = "kitchen"
	StationTypeBar     StationType = "bar"
	StationTypeCashier StationType = "cashier"
	StationTypeOther   StationType = "other"
)

// Station represents a preparation station
type Station struct {
	ID        uuid.UUID   `db:"id" json:"id"`
	Name      string      `db:"name" json:"name"`
	Type      StationType `db:"type" json:"type"`
	PrinterID *uuid.UUID  `db:"printer_id" json:"printer_id"`
	DisplayID *uuid.UUID  `db:"display_id" json:"display_id"`
	IsActive  bool        `db:"is_active" json:"is_active"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`

	// Not stored directly in database
	Printer *Printer `db:"-" json:"printer,omitempty"`
	Display *Display `db:"-" json:"display,omitempty"`
}

// RoutingRule represents a rule for routing menu items to stations
type RoutingRule struct {
	ID         uuid.UUID `db:"id" json:"id"`
	MenuItemID uuid.UUID `db:"menu_item_id" json:"menu_item_id"`
	StationID  uuid.UUID `db:"station_id" json:"station_id"`
	Priority   int       `db:"priority" json:"priority"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`

	// Not stored directly in database
	Station *Station `db:"-" json:"station,omitempty"`
}

// StationRequest is used for station creation/update
type StationRequest struct {
	Name      string      `json:"name" validate:"required,min=1,max=100"`
	Type      StationType `json:"type" validate:"required,oneof=kitchen bar cashier other"`
	PrinterID *uuid.UUID  `json:"printer_id"`
	DisplayID *uuid.UUID  `json:"display_id"`
	IsActive  bool        `json:"is_active"`
}

// RoutingRuleRequest is used for routing rule creation/update
type RoutingRuleRequest struct {
	MenuItemID uuid.UUID `json:"menu_item_id" validate:"required"`
	StationID  uuid.UUID `json:"station_id" validate:"required"`
	Priority   int       `json:"priority" validate:"gte=1"`
}

package models

import (
	"time"

	"github.com/google/uuid"
)

// PrinterType represents a printer type
type PrinterType string

const (
	PrinterTypeThermal PrinterType = "thermal"
	PrinterTypeKitchen PrinterType = "kitchen"
	PrinterTypeReceipt PrinterType = "receipt"
	PrinterTypeOther   PrinterType = "other"
)

// DisplayType represents a display type
type DisplayType string

const (
	DisplayTypeKitchen  DisplayType = "kitchen"
	DisplayTypeCustomer DisplayType = "customer"
	DisplayTypeOther    DisplayType = "other"
)

// Printer represents a physical printer
type Printer struct {
	ID        uuid.UUID   `db:"id" json:"id"`
	Name      string      `db:"name" json:"name"`
	Type      PrinterType `db:"type" json:"type"`
	IPAddress *string     `db:"ip_address" json:"ip_address"`
	Port      *int        `db:"port" json:"port"`
	Model     *string     `db:"model" json:"model"`
	IsDefault bool        `db:"is_default" json:"is_default"`
	IsActive  bool        `db:"is_active" json:"is_active"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`
}

// Display represents a display device
type Display struct {
	ID        uuid.UUID   `db:"id" json:"id"`
	Name      string      `db:"name" json:"name"`
	Type      DisplayType `db:"type" json:"type"`
	IPAddress *string     `db:"ip_address" json:"ip_address"`
	IsActive  bool        `db:"is_active" json:"is_active"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`
}

// PrinterRequest is used for printer creation/update
type PrinterRequest struct {
	Name      string      `json:"name" validate:"required,min=1,max=100"`
	Type      PrinterType `json:"type" validate:"required,oneof=thermal kitchen receipt other"`
	IPAddress *string     `json:"ip_address" validate:"omitempty,ip"`
	Port      *int        `json:"port" validate:"omitempty,min=1,max=65535"`
	Model     *string     `json:"model"`
	IsDefault bool        `json:"is_default"`
	IsActive  bool        `json:"is_active"`
}

// DisplayRequest is used for display creation/update
type DisplayRequest struct {
	Name      string      `json:"name" validate:"required,min=1,max=100"`
	Type      DisplayType `json:"type" validate:"required,oneof=kitchen customer other"`
	IPAddress *string     `json:"ip_address" validate:"omitempty,ip"`
	IsActive  bool        `json:"is_active"`
}

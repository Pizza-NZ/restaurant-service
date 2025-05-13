package models

import (
	"time"

	"github.com/google/uuid"
)

// MenuCategory represents a menu category
type MenuCategory struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	ColorCode    *string   `db:"color_code" json:"color_code"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// MenuItem represents a menu item
type MenuItem struct {
	ID          uuid.UUID `db:"id" json:"id"`
	CategoryID  uuid.UUID `db:"category_id" json:"category_id"`
	Name        string    `db:"name" json:"name"`
	Price       float64   `db:"price" json:"price"`
	Available   bool      `db:"available" json:"available"`
	Description *string   `db:"description" json:"description"`
	ImagePath   *string   `db:"image_path" json:"image_path"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`

	// These fields are not stored in the database directly
	Category  *MenuCategory      `db:"-" json:"category,omitempty"`
	Modifiers []MenuItemModifier `db:"-" json:"modifiers,omitempty"`
}

// Modifier represents a modifier group
type Modifier struct {
	ID         uuid.UUID `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	IsMultiple bool      `db:"is_multiple" json:"is_multiple"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`

	// Not stored directly in the database
	Options []ModifierOption `db:"-" json:"options,omitempty"`
}

// ModifierOption represents an option within a modifier group
type ModifierOption struct {
	ID              uuid.UUID `db:"id" json:"id"`
	ModifierID      uuid.UUID `db:"modifier_id" json:"modifier_id"`
	Name            string    `db:"name" json:"name"`
	PriceAdjustment float64   `db:"price_adjustment" json:"price_adjustment"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// MenuItemModifier represents the association between a menu item and a modifier
type MenuItemModifier struct {
	ID         uuid.UUID `db:"id" json:"id"`
	MenuItemID uuid.UUID `db:"menu_item_id" json:"menu_item_id"`
	ModifierID uuid.UUID `db:"modifier_id" json:"modifier_id"`
	Required   bool      `db:"required" json:"required"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`

	// Not stored directly in the database
	Modifier *Modifier `db:"-" json:"modifier,omitempty"`
}

// MenuCategoryRequest is used for category creation/update
type MenuCategoryRequest struct {
	Name         string  `json:"name" validate:"required,min=1,max=50"`
	DisplayOrder int     `json:"display_order"`
	ColorCode    *string `json:"color_code" validate:"omitempty,len=7"`
}

// MenuItemRequest is used for menu item creation/update
type MenuItemRequest struct {
	CategoryID  uuid.UUID   `json:"category_id" validate:"required"`
	Name        string      `json:"name" validate:"required,min=1,max=100"`
	Price       float64     `json:"price" validate:"required,gte=0"`
	Available   bool        `json:"available"`
	Description *string     `json:"description"`
	ImagePath   *string     `json:"image_path"`
	ModifierIDs []uuid.UUID `json:"modifier_ids"`
	StationID   string      `json:"station_id" validate:"required"`
}

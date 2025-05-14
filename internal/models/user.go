package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleCashier UserRole = "cashier"
	RoleKitchen UserRole = "kitchen"
)

type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"` // Never expose in JSON
	Name         string    `db:"name" json:"name"`
	Role         UserRole  `db:"role" json:"role"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// UserRequest is used for user creation/update requests
type UserRequest struct {
	Username string   `json:"username" validate:"required,min=3,max=50"`
	Password string   `json:"password" validate:"required,min=6"`
	Name     string   `json:"name" validate:"required,min=2,max=100"`
	Role     UserRole `json:"role" validate:"required,oneof=admin manager cashier kitchen"`
	IsActive bool     `json:"is_active"`
}

// UserUpdateRequest is used for updating user information
type UserUpdateRequest struct {
	Username string   `json:"username" validate:"required,min=3,max=50"`
	Name     string   `json:"name" validate:"required,min=2,max=100"`
	Role     UserRole `json:"role" validate:"required,oneof=admin manager cashier kitchen"`
	IsActive bool     `json:"is_active"`
}

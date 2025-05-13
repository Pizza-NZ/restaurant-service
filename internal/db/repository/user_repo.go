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

// UserRepository handles user data access
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, name, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, name, role, is_active, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// List retrieves all users
func (r *UserRepository) List(ctx context.Context) ([]models.User, error) {
	query := `
		SELECT id, username, password_hash, name, role, is_active, created_at, updated_at
		FROM users
		ORDER BY username ASC
	`

	var users []models.User
	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user models.User) (*models.User, error) {
	query := `
		INSERT INTO users (username, password_hash, name, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, password_hash, name, role, is_active, created_at, updated_at
	`

	var createdUser models.User
	err := r.db.GetContext(
		ctx,
		&createdUser,
		query,
		user.Username,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &createdUser, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user models.User) (*models.User, error) {
	query := `
		UPDATE users
		SET username = $1, name = $2, role = $3, is_active = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, username, password_hash, name, role, is_active, created_at, updated_at
	`

	var updatedUser models.User
	err := r.db.GetContext(
		ctx,
		&updatedUser,
		query,
		user.Username,
		user.Name,
		user.Role,
		user.IsActive,
		time.Now(),
		user.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &updatedUser, nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, passwordHash, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM users
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

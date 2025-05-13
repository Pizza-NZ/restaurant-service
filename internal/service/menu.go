// internal/service/menu.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// MenuService handles menu-related business logic
type MenuService struct {
	repos *repository.Factory
}

// NewMenuService creates a new menu service
func NewMenuService(repos *repository.Factory) *MenuService {
	return &MenuService{
		repos: repos,
	}
}

// GetCategories retrieves all menu categories
func (s *MenuService) GetCategories(ctx context.Context) ([]models.MenuCategory, error) {
	return s.repos.Menu.ListCategories(ctx)
}

// GetCategory retrieves a menu category by ID
func (s *MenuService) GetCategory(ctx context.Context, id uuid.UUID) (*models.MenuCategory, error) {
	return s.repos.Menu.GetCategoryByID(ctx, id)
}

// CreateCategory creates a new menu category
func (s *MenuService) CreateCategory(ctx context.Context, req models.MenuCategoryRequest) (*models.MenuCategory, error) {
	category := models.MenuCategory{
		Name:         req.Name,
		DisplayOrder: req.DisplayOrder,
		ColorCode:    req.ColorCode,
	}

	return s.repos.Menu.CreateCategory(ctx, category)
}

// UpdateCategory updates a menu category
func (s *MenuService) UpdateCategory(ctx context.Context, id uuid.UUID, req models.MenuCategoryRequest) (*models.MenuCategory, error) {
	// Get the existing category
	existingCategory, err := s.repos.Menu.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	// Update the fields
	existingCategory.Name = req.Name
	existingCategory.DisplayOrder = req.DisplayOrder
	existingCategory.ColorCode = req.ColorCode

	return s.repos.Menu.UpdateCategory(ctx, *existingCategory)
}

// DeleteCategory deletes a menu category
func (s *MenuService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return s.repos.Menu.DeleteCategory(ctx, id)
}

// GetItems retrieves menu items, optionally filtered by category
func (s *MenuService) GetItems(ctx context.Context, categoryID *uuid.UUID) ([]models.MenuItem, error) {
	return s.repos.Menu.ListItems(ctx, categoryID)
}

// GetItem retrieves a menu item by ID
func (s *MenuService) GetItem(ctx context.Context, id uuid.UUID) (*models.MenuItem, error) {
	return s.repos.Menu.GetItemByID(ctx, id)
}

// CreateItem creates a new menu item
func (s *MenuService) CreateItem(ctx context.Context, req models.MenuItemRequest) (*models.MenuItem, error) {
	// Verify the category exists
	_, err := s.repos.Menu.GetCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Verify the station exists
	stationID, err := uuid.Parse(req.StationID)
	if err != nil {
		return nil, fmt.Errorf("invalid station ID: %w", err)
	}

	_, err = s.repos.Station.GetByID(ctx, stationID)
	if err != nil {
		return nil, fmt.Errorf("invalid station ID: %w", err)
	}

	// Create the menu item
	item := models.MenuItem{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Price:       req.Price,
		Available:   req.Available,
		Description: req.Description,
		ImagePath:   req.ImagePath,
	}

	return s.repos.Menu.CreateItem(ctx, nil, item, req.ModifierIDs, stationID)
}

// UpdateItem updates a menu item
func (s *MenuService) UpdateItem(ctx context.Context, id uuid.UUID, req models.MenuItemRequest) (*models.MenuItem, error) {
	// Verify the item exists
	existingItem, err := s.repos.Menu.GetItemByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("menu item not found: %w", err)
	}

	// Verify the category exists
	_, err = s.repos.Menu.GetCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Start a transaction
	tx, err := s.repos.Menu.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Update the menu item
	_, err = tx.Exec(`
		UPDATE menu_items
		SET category_id = $1, name = $2, price = $3, available = $4, description = $5, image_path = $6, updated_at = $7
		WHERE id = $8
	`,
		req.CategoryID,
		req.Name,
		req.Price,
		req.Available,
		req.Description,
		req.ImagePath,
		time.Now(),
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update menu item: %w", err)
	}

	// Update modifiers (remove existing ones and add new ones)
	_, err = tx.Exec("DELETE FROM menu_item_modifiers WHERE menu_item_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to remove existing modifiers: %w", err)
	}

	for _, modID := range req.ModifierIDs {
		_, err = tx.Exec(
			"INSERT INTO menu_item_modifiers (menu_item_id, modifier_id, required) VALUES ($1, $2, $3)",
			id, modID, false,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add modifier: %w", err)
		}
	}

	// Update routing rule if station ID changed
	stationID, err := uuid.Parse(req.StationID)
	if err != nil {
		return nil, fmt.Errorf("invalid station ID: %w", err)
	}

	// Check if there's an existing routing rule
	var ruleID uuid.UUID
	err = tx.Get(&ruleID, "SELECT id FROM routing_rules WHERE menu_item_id = $1 LIMIT 1", id)
	if err == nil {
		// Update existing rule
		_, err = tx.Exec(
			"UPDATE routing_rules SET station_id = $1, updated_at = $2 WHERE id = $3",
			stationID, time.Now(), ruleID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update routing rule: %w", err)
		}
	} else {
		// Create new rule
		_, err = tx.Exec(
			"INSERT INTO routing_rules (menu_item_id, station_id, priority) VALUES ($1, $2, $3)",
			id, stationID, 1,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create routing rule: %w", err)
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get the updated item
	return s.repos.Menu.GetItemByID(ctx, id)
}

// DeleteItem deletes a menu item
func (s *MenuService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	// Start a transaction
	tx, err := s.repos.Menu.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Delete routing rules for this item
	_, err = tx.Exec("DELETE FROM routing_rules WHERE menu_item_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete routing rules: %w", err)
	}

	// Delete menu item modifiers
	_, err = tx.Exec("DELETE FROM menu_item_modifiers WHERE menu_item_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete menu item modifiers: %w", err)
	}

	// Delete the menu item
	_, err = tx.Exec("DELETE FROM menu_items WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete menu item: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetModifiers retrieves all modifiers
func (s *MenuService) GetModifiers(ctx context.Context) ([]models.Modifier, error) {
	query := `SELECT id, name, is_multiple, created_at, updated_at FROM modifiers ORDER BY name ASC`

	var modifiers []models.Modifier
	err := s.repos.Menu.DB.SelectContext(ctx, &modifiers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get modifiers: %w", err)
	}

	// Get options for each modifier
	for i := range modifiers {
		options, err := s.repos.Menu.GetModifierOptions(ctx, modifiers[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get modifier options: %w", err)
		}
		modifiers[i].Options = options
	}

	return modifiers, nil
}

// GetModifier retrieves a modifier by ID
func (s *MenuService) GetModifier(ctx context.Context, id uuid.UUID) (*models.Modifier, error) {
	query := `SELECT id, name, is_multiple, created_at, updated_at FROM modifiers WHERE id = $1`

	var modifier models.Modifier
	err := s.repos.Menu.DB.GetContext(ctx, &modifier, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get modifier: %w", err)
	}

	// Get options
	options, err := s.repos.Menu.GetModifierOptions(ctx, modifier.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get modifier options: %w", err)
	}
	modifier.Options = options

	return &modifier, nil
}

// CreateModifier creates a new modifier
func (s *MenuService) CreateModifier(ctx context.Context, name string, isMultiple bool, options []models.ModifierOptionRequest) (*models.Modifier, error) {
	// Start a transaction
	tx, err := s.repos.Menu.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Create the modifier
	var modifierID uuid.UUID
	err = tx.GetContext(
		ctx,
		&modifierID,
		"INSERT INTO modifiers (name, is_multiple) VALUES ($1, $2) RETURNING id",
		name, isMultiple,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create modifier: %w", err)
	}

	// Add options
	for _, opt := range options {
		_, err = tx.Exec(
			"INSERT INTO modifier_options (modifier_id, name, price_adjustment) VALUES ($1, $2, $3)",
			modifierID, opt.Name, opt.PriceAdjustment,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add modifier option: %w", err)
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get the created modifier
	return s.GetModifier(ctx, modifierID)
}

// UpdateModifier updates a modifier
func (s *MenuService) UpdateModifier(ctx context.Context, id uuid.UUID, name string, isMultiple bool, options []models.ModifierOptionRequest) (*models.Modifier, error) {
	// Start a transaction
	tx, err := s.repos.Menu.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Update the modifier
	_, err = tx.Exec(
		"UPDATE modifiers SET name = $1, is_multiple = $2, updated_at = $3 WHERE id = $4",
		name, isMultiple, time.Now(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update modifier: %w", err)
	}

	// Delete existing options
	_, err = tx.Exec("DELETE FROM modifier_options WHERE modifier_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing options: %w", err)
	}

	// Add new options
	for _, opt := range options {
		_, err = tx.Exec(
			"INSERT INTO modifier_options (modifier_id, name, price_adjustment) VALUES ($1, $2, $3)",
			id, opt.Name, opt.PriceAdjustment,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add modifier option: %w", err)
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get the updated modifier
	return s.GetModifier(ctx, id)
}

// DeleteModifier deletes a modifier
func (s *MenuService) DeleteModifier(ctx context.Context, id uuid.UUID) error {
	// Check if the modifier is used by any menu items
	var count int
	err := s.repos.Menu.DB.GetContext(
		ctx,
		&count,
		"SELECT COUNT(*) FROM menu_item_modifiers WHERE modifier_id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to check modifier usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete modifier used by %d menu items", count)
	}

	// Start a transaction
	tx, err := s.repos.Menu.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Delete options
	_, err = tx.Exec("DELETE FROM modifier_options WHERE modifier_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete modifier options: %w", err)
	}

	// Delete the modifier
	_, err = tx.Exec("DELETE FROM modifiers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete modifier: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

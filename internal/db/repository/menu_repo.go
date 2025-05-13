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

// MenuRepository handles menu data access
type MenuRepository struct {
	db *sqlx.DB
}

// NewMenuRepository creates a new menu repository
func NewMenuRepository(db *sqlx.DB) *MenuRepository {
	return &MenuRepository{db: db}
}

// BeginTransaction begins a new transaction
func (r *MenuRepository) beginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	return tx, nil
}

// GetCategoryByID retrieves a menu category by ID
func (r *MenuRepository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.MenuCategory, error) {
	query := `
		SELECT id, name, display_order, color_code, created_at, updated_at
		FROM menu_categories
		WHERE id = $1
	`

	var category models.MenuCategory
	err := r.db.GetContext(ctx, &category, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu category: %w", err)
	}

	return &category, nil
}

// ListCategories retrieves all menu categories
func (r *MenuRepository) ListCategories(ctx context.Context) ([]models.MenuCategory, error) {
	query := `
		SELECT id, name, display_order, color_code, created_at, updated_at
		FROM menu_categories
		ORDER BY display_order ASC, name ASC
	`

	var categories []models.MenuCategory
	err := r.db.SelectContext(ctx, &categories, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list menu categories: %w", err)
	}

	return categories, nil
}

// CreateCategory creates a new menu category
func (r *MenuRepository) CreateCategory(ctx context.Context, category models.MenuCategory) (*models.MenuCategory, error) {
	query := `
		INSERT INTO menu_categories (name, display_order, color_code)
		VALUES ($1, $2, $3)
		RETURNING id, name, display_order, color_code, created_at, updated_at
	`

	var createdCategory models.MenuCategory
	err := r.db.GetContext(
		ctx,
		&createdCategory,
		query,
		category.Name,
		category.DisplayOrder,
		category.ColorCode,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create menu category: %w", err)
	}

	return &createdCategory, nil
}

// UpdateCategory updates a menu category
func (r *MenuRepository) UpdateCategory(ctx context.Context, category models.MenuCategory) (*models.MenuCategory, error) {
	query := `
		UPDATE menu_categories
		SET name = $1, display_order = $2, color_code = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, name, display_order, color_code, created_at, updated_at
	`

	var updatedCategory models.MenuCategory
	err := r.db.GetContext(
		ctx,
		&updatedCategory,
		query,
		category.Name,
		category.DisplayOrder,
		category.ColorCode,
		time.Now(),
		category.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update menu category: %w", err)
	}

	return &updatedCategory, nil
}

// DeleteCategory deletes a menu category
func (r *MenuRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM menu_categories
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete menu category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("menu category not found")
	}

	return nil
}

// GetItemByID retrieves a menu item by ID
func (r *MenuRepository) GetItemByID(ctx context.Context, id uuid.UUID) (*models.MenuItem, error) {
	query := `
		SELECT id, category_id, name, price, available, description, image_path, created_at, updated_at
		FROM menu_items
		WHERE id = $1
	`

	var item models.MenuItem
	err := r.db.GetContext(ctx, &item, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu item: %w", err)
	}

	// Get associated category
	category, err := r.GetCategoryByID(ctx, item.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item category: %w", err)
	}
	item.Category = category

	// Get modifiers
	modifiers, err := r.GetItemModifiers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get item modifiers: %w", err)
	}
	item.Modifiers = modifiers

	return &item, nil
}

// GetItemModifiers retrieves modifiers for a menu item
func (r *MenuRepository) GetItemModifiers(ctx context.Context, itemID uuid.UUID) ([]models.MenuItemModifier, error) {
	query := `
		SELECT mim.id, mim.menu_item_id, mim.modifier_id, mim.required, mim.created_at,
		       m.id as "modifier.id", m.name as "modifier.name", m.is_multiple as "modifier.is_multiple"
		FROM menu_item_modifiers mim
		JOIN modifiers m ON mim.modifier_id = m.id
		WHERE mim.menu_item_id = $1
	`

	rows, err := r.db.QueryxContext(ctx, query, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to query item modifiers: %w", err)
	}
	defer rows.Close()

	var modifiers []models.MenuItemModifier
	for rows.Next() {
		var mim models.MenuItemModifier
		var modifier models.Modifier

		// Use a map to handle the nested structure
		dest := map[string]interface{}{
			"id":                   &mim.ID,
			"menu_item_id":         &mim.MenuItemID,
			"modifier_id":          &mim.ModifierID,
			"required":             &mim.Required,
			"created_at":           &mim.CreatedAt,
			"modifier.id":          &modifier.ID,
			"modifier.name":        &modifier.Name,
			"modifier.is_multiple": &modifier.IsMultiple,
		}

		err := rows.MapScan(dest)
		if err != nil {
			return nil, fmt.Errorf("failed to scan modifier row: %w", err)
		}

		// Get options for this modifier
		options, err := r.GetModifierOptions(ctx, modifier.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get modifier options: %w", err)
		}
		modifier.Options = options

		mim.Modifier = &modifier
		modifiers = append(modifiers, mim)
	}

	return modifiers, nil
}

// GetModifierOptions retrieves options for a modifier
func (r *MenuRepository) GetModifierOptions(ctx context.Context, modifierID uuid.UUID) ([]models.ModifierOption, error) {
	query := `
		SELECT id, modifier_id, name, price_adjustment, created_at, updated_at
		FROM modifier_options
		WHERE modifier_id = $1
		ORDER BY name ASC
	`

	var options []models.ModifierOption
	err := r.db.SelectContext(ctx, &options, query, modifierID)
	if err != nil {
		return nil, fmt.Errorf("failed to get modifier options: %w", err)
	}

	return options, nil
}

// ListItems retrieves all menu items, optionally filtered by category
func (r *MenuRepository) ListItems(ctx context.Context, categoryID *uuid.UUID) ([]models.MenuItem, error) {
	var query string
	var args []interface{}

	if categoryID != nil {
		query = `
			SELECT id, category_id, name, price, available, description, image_path, created_at, updated_at
			FROM menu_items
			WHERE category_id = $1
			ORDER BY name ASC
		`
		args = append(args, *categoryID)
	} else {
		query = `
			SELECT id, category_id, name, price, available, description, image_path, created_at, updated_at
			FROM menu_items
			ORDER BY name ASC
		`
	}

	var items []models.MenuItem
	err := r.db.SelectContext(ctx, &items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list menu items: %w", err)
	}

	// For each item, get its category (but not modifiers to avoid too many queries)
	categories := make(map[uuid.UUID]*models.MenuCategory)
	for i := range items {
		if _, ok := categories[items[i].CategoryID]; !ok {
			category, err := r.GetCategoryByID(ctx, items[i].CategoryID)
			if err != nil {
				return nil, fmt.Errorf("failed to get category for item: %w", err)
			}
			categories[items[i].CategoryID] = category
		}
		items[i].Category = categories[items[i].CategoryID]
	}

	return items, nil
}

// CreateItem creates a new menu item with modifiers and routing
func (r *MenuRepository) CreateItem(ctx context.Context, tx *sqlx.Tx, item models.MenuItem, modifierIDs []uuid.UUID, stationID uuid.UUID) (*models.MenuItem, error) {
	// Determine if we're using a provided transaction or creating our own
	var err error

	// Verify transaction in process
	if tx == nil {
		tx, err = r.beginTransaction(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
	}

	// Insert the menu item
	query := `
		INSERT INTO menu_items (category_id, name, price, available, description, image_path)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, category_id, name, price, available, description, image_path, created_at, updated_at
	`

	var createdItem models.MenuItem
	err = tx.GetContext(
		ctx,
		&createdItem,
		query,
		item.CategoryID,
		item.Name,
		item.Price,
		item.Available,
		item.Description,
		item.ImagePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create menu item: %w", err)
	}

	// Add modifiers if any
	for _, modID := range modifierIDs {
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO menu_item_modifiers (menu_item_id, modifier_id, required) VALUES ($1, $2, $3)`,
			createdItem.ID, modID, false,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add modifier to item: %w", err)
		}
	}

	// Add routing rule
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO routing_rules (menu_item_id, station_id, priority) VALUES ($1, $2, $3)`,
		createdItem.ID, stationID, 1,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add routing rule for item: %w", err)
	}

	// If we started the transaction, we'll commit it in the defer function

	// Get the fully populated item
	return r.GetItemByID(ctx, createdItem.ID)
}

// UpdateItem updates a menu item
func (r *MenuRepository) UpdateItem(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, req models.MenuItemRequest) (*models.MenuItem, error) {
	var err error

	// Verify transaction in process
	if tx == nil {
		tx, err = r.beginTransaction(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
	}

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

	return r.GetItemByID(ctx, id)
}

// DeleteItem deletes a menu item
// This function will also delete associated routing rules and modifiers
func (r *MenuRepository) DeleteItem(ctx context.Context, id uuid.UUID) error {
	tx, err := r.beginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

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

// ListModifiers retrieves all modifiers
func (r *MenuRepository) ListModifiers(ctx context.Context) ([]models.Modifier, error) {
	query := `
		SELECT id, name, is_multiple, created_at, updated_at
		FROM modifiers
		ORDER BY name ASC
	`

	var modifiers []models.Modifier

	err := r.db.GetContext(ctx, &modifiers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get modifier: %w", err)
	}

	return modifiers, nil
}

// ListModifierWithOptions retrieves all modifiers with their options
func (r *MenuRepository) ListModifierWithOptions(ctx context.Context) ([]models.Modifier, error) {
	modifiers, err := r.ListModifiers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list modifiers: %w", err)
	}

	// Get options for modifiers
	for i := range modifiers {
		options, err := r.GetModifierOptions(ctx, modifiers[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get modifier options: %w", err)
		}
		modifiers[i].Options = options
	}

	return modifiers, nil

}

// GetModifier retrieves a modifier by ID
func (r *MenuRepository) GetModifier(ctx context.Context, id uuid.UUID) (*models.Modifier, error) {
	query := `
		SELECT id, name, is_multiple, created_at, updated_at
		FROM modifiers
		WHERE id = $1
	`

	var modifier models.Modifier

	err := r.db.GetContext(ctx, &modifier, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get modifier: %w", err)
	}

	// Get options for this mod
	options, err := r.GetModifierOptions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get modifier options: %w", err)
	}
	modifier.Options = options

	return &modifier, nil
}

// CreateModifier creates a new modifier
func (r *MenuRepository) CreateModifier(ctx context.Context, name string, isMultiple bool, options []models.ModifierOption) (*models.Modifier, error) {
	// Start a transaction
	tx, err := r.beginTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

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
	return r.GetModifier(ctx, modifierID)
}

// UpdateModifier updates a modifier
func (r *MenuRepository) UpdateModifier(ctx context.Context, id uuid.UUID, name string, isMultiple bool, options []models.ModifierOption) (*models.Modifier, error) {
	// Start a transaction
	tx, err := r.beginTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

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
	return r.GetModifier(ctx, id)
}

// DeleteModifier deletes a modifier
func (r *MenuRepository) DeleteModifier(ctx context.Context, id uuid.UUID) error {
	// Check if the modifier is used by any menu items
	var count int
	err := r.db.GetContext(
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
	tx, err := r.beginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

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

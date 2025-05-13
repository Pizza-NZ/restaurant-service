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

	tempTx := tx

	if tempTx == nil {
		tempTx, err = r.db.BeginTxx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			} else {
				_ = tx.Commit()
			}
		}()
	}

	// Insert the menu item
	query := `
		INSERT INTO menu_items (category_id, name, price, available, description, image_path)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, category_id, name, price, available, description, image_path, created_at, updated_at
	`

	var createdItem models.MenuItem
	err = tempTx.GetContext(
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
		_, err = tempTx.ExecContext(
			ctx,
			`INSERT INTO menu_item_modifiers (menu_item_id, modifier_id, required) VALUES ($1, $2, $3)`,
			createdItem.ID, modID, false,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add modifier to item: %w", err)
		}
	}

	// Add routing rule
	_, err = tempTx.ExecContext(
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

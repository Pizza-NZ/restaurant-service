// internal/service/menu.go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// MenuRepository defines the database methods needed by the MenuService.
type MenuRepository interface {
	ListCategories(ctx context.Context) ([]models.MenuCategory, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.MenuCategory, error)
	CreateCategory(ctx context.Context, category models.MenuCategory) (*models.MenuCategory, error)
	UpdateCategory(ctx context.Context, category models.MenuCategory) (*models.MenuCategory, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	ListItems(ctx context.Context, categoryID *uuid.UUID) ([]models.MenuItem, error)
	GetItemByID(ctx context.Context, id uuid.UUID) (*models.MenuItem, error)
	CreateItem(ctx context.Context, tx *sqlx.Tx, item models.MenuItem, modifierIDs []uuid.UUID, stationID uuid.UUID) (*models.MenuItem, error)
	UpdateItem(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, req models.MenuItemRequest) (*models.MenuItem, error)
	DeleteItem(ctx context.Context, id uuid.UUID) error
	ListModifiers(ctx context.Context) ([]models.Modifier, error)
	GetModifier(ctx context.Context, id uuid.UUID) (*models.Modifier, error)
	CreateModifier(ctx context.Context, name string, isMultiple bool, options []models.ModifierOptionRequest) (*models.Modifier, error)
	UpdateModifier(ctx context.Context, id uuid.UUID, name string, isMultiple bool, options []models.ModifierOptionRequest) (*models.Modifier, error)
	DeleteModifier(ctx context.Context, id uuid.UUID) error
}

// StationRepositoryForMenu defines the station-related methods needed by the MenuService.
type StationRepositoryForMenu interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Station, error)
}

// MenuService handles menu-related business logic
type MenuService struct {
	menuRepo    MenuRepository
	stationRepo StationRepositoryForMenu
}

// NewMenuService creates a new menu service
func NewMenuService(menuRepo MenuRepository, stationRepo StationRepositoryForMenu) *MenuService {
	return &MenuService{
		menuRepo:    menuRepo,
		stationRepo: stationRepo,
	}
}

// GetCategories retrieves all menu categories
func (s *MenuService) GetCategories(ctx context.Context) ([]models.MenuCategory, error) {
	return s.menuRepo.ListCategories(ctx)
}

// GetCategory retrieves a menu category by ID
func (s *MenuService) GetCategory(ctx context.Context, id uuid.UUID) (*models.MenuCategory, error) {
	return s.menuRepo.GetCategoryByID(ctx, id)
}

// CreateCategory creates a new menu category
func (s *MenuService) CreateCategory(ctx context.Context, req models.MenuCategoryRequest) (*models.MenuCategory, error) {
	category := models.MenuCategory{
		Name:         req.Name,
		DisplayOrder: req.DisplayOrder,
		ColorCode:    req.ColorCode,
	}

	return s.menuRepo.CreateCategory(ctx, category)
}

// UpdateCategory updates a menu category
func (s *MenuService) UpdateCategory(ctx context.Context, id uuid.UUID, req models.MenuCategoryRequest) (*models.MenuCategory, error) {
	// Get the existing category
	existingCategory, err := s.menuRepo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	// Update the fields
	existingCategory.Name = req.Name
	existingCategory.DisplayOrder = req.DisplayOrder
	existingCategory.ColorCode = req.ColorCode

	return s.menuRepo.UpdateCategory(ctx, *existingCategory)
}

// DeleteCategory deletes a menu category
func (s *MenuService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return s.menuRepo.DeleteCategory(ctx, id)
}

// GetItems retrieves menu items, optionally filtered by category
func (s *MenuService) GetItems(ctx context.Context, categoryID *uuid.UUID) ([]models.MenuItem, error) {
	return s.menuRepo.ListItems(ctx, categoryID)
}

// GetItem retrieves a menu item by ID
func (s *MenuService) GetItem(ctx context.Context, id uuid.UUID) (*models.MenuItem, error) {
	return s.menuRepo.GetItemByID(ctx, id)
}

// CreateItem creates a new menu item
func (s *MenuService) CreateItem(ctx context.Context, req models.MenuItemRequest) (*models.MenuItem, error) {
	// Verify the category exists
	_, err := s.menuRepo.GetCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Verify the station exists
	stationID, err := uuid.Parse(req.StationID)
	if err != nil {
		return nil, fmt.Errorf("invalid station ID format: %w", err)
	}

	_, err = s.stationRepo.GetByID(ctx, stationID)
	if err != nil {
		return nil, fmt.Errorf("station with given ID not found: %w", err)
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

	return s.menuRepo.CreateItem(ctx, nil, item, req.ModifierIDs, stationID)
}

// UpdateItem updates a menu item
func (s *MenuService) UpdateItem(ctx context.Context, id uuid.UUID, req models.MenuItemRequest) (*models.MenuItem, error) {
	// Verify the item exists
	_, err := s.menuRepo.GetItemByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("menu item not found: %w", err)
	}

	// Verify the category exists
	_, err = s.menuRepo.GetCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Verify the station exists
	stationID, err := uuid.Parse(req.StationID)
	if err != nil {
		return nil, fmt.Errorf("invalid station ID format: %w", err)
	}
	_, err = s.stationRepo.GetByID(ctx, stationID)
	if err != nil {
		return nil, fmt.Errorf("station with given ID not found: %w", err)
	}

	// Get the updated item
	return s.menuRepo.UpdateItem(ctx, nil, id, req)
}

// DeleteItem deletes a menu item
func (s *MenuService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	return s.menuRepo.DeleteItem(ctx, id)
}

// GetModifiers retrieves all modifiers
func (s *MenuService) GetModifiers(ctx context.Context) ([]models.Modifier, error) {
	return s.menuRepo.ListModifiers(ctx)
}

// GetModifier retrieves a modifier by ID
func (s *MenuService) GetModifier(ctx context.Context, id uuid.UUID) (*models.Modifier, error) {
	return s.menuRepo.GetModifier(ctx, id)
}

// CreateModifier creates a new modifier
func (s *MenuService) CreateModifier(ctx context.Context, name string, isMultiple bool, options []models.ModifierOptionRequest) (*models.Modifier, error) {
	return s.menuRepo.CreateModifier(ctx, name, isMultiple, options)
}

// UpdateModifier updates a modifier
func (s *MenuService) UpdateModifier(ctx context.Context, id uuid.UUID, name string, isMultiple bool, options []models.ModifierOptionRequest) (*models.Modifier, error) {
	return s.menuRepo.UpdateModifier(ctx, id, name, isMultiple, options)
}

// DeleteModifier deletes a modifier
func (s *MenuService) DeleteModifier(ctx context.Context, id uuid.UUID) error {
	return s.menuRepo.DeleteModifier(ctx, id)
}

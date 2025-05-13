// internal/service/menu.go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// MenuService handles menu-related business logic
type MenuService struct {
	repos *repository.Repositories
}

// NewMenuService creates a new menu service
func NewMenuService(repos *repository.Repositories) *MenuService {
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
	_, err := s.repos.Menu.GetItemByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("menu item not found: %w", err)
	}

	// Verify the category exists
	_, err = s.repos.Menu.GetCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	// Get the updated item
	return s.repos.Menu.UpdateItem(ctx, nil, id, req)
}

// DeleteItem deletes a menu item
func (s *MenuService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	return s.repos.Menu.DeleteItem(ctx, id)
}

// GetModifiers retrieves all modifiers
func (s *MenuService) GetModifiers(ctx context.Context) ([]models.Modifier, error) {
	return s.repos.Menu.ListModifiers(ctx)
}

// GetModifier retrieves a modifier by ID
func (s *MenuService) GetModifier(ctx context.Context, id uuid.UUID) (*models.Modifier, error) {
	return s.repos.Menu.GetModifier(ctx, id)
}

// CreateModifier creates a new modifier
func (s *MenuService) CreateModifier(ctx context.Context, name string, isMultiple bool, options []models.ModifierOption) (*models.Modifier, error) {
	return s.repos.Menu.CreateModifier(ctx, name, isMultiple, options)
}

// UpdateModifier updates a modifier
func (s *MenuService) UpdateModifier(ctx context.Context, id uuid.UUID, name string, isMultiple bool, options []models.ModifierOption) (*models.Modifier, error) {
	return s.repos.Menu.UpdateModifier(ctx, id, name, isMultiple, options)
}

// DeleteModifier deletes a modifier
func (s *MenuService) DeleteModifier(ctx context.Context, id uuid.UUID) error {
	return s.repos.Menu.DeleteModifier(ctx, id)
}

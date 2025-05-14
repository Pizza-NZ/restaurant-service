package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// StationService handles station-related business logic
type StationService struct {
	repos *repository.Repositories
}

// NewStationService creates a new station service
func NewStationService(repos *repository.Repositories) *StationService {
	return &StationService{
		repos: repos,
	}
}

// GetStations retrieves all stations
func (s *StationService) GetStations(ctx context.Context) ([]models.Station, error) {
	return s.repos.Station.List(ctx)
}

// GetStation retrieves a station by ID
func (s *StationService) GetStation(ctx context.Context, id uuid.UUID) (*models.Station, error) {
	return s.repos.Station.GetByID(ctx, id)
}

// CreateStation creates a new station
func (s *StationService) CreateStation(ctx context.Context, req models.StationRequest) (*models.Station, error) {
	// Validate printer ID if provided
	if req.PrinterID != nil {
		_, err := s.repos.Printer.GetPrinterByID(ctx, *req.PrinterID)
		if err != nil {
			return nil, fmt.Errorf("invalid printer ID: %w", err)
		}
	}

	// Validate display ID if provided
	if req.DisplayID != nil {
		_, err := s.repos.Printer.GetDisplayByID(ctx, *req.DisplayID)
		if err != nil {
			return nil, fmt.Errorf("invalid display ID: %w", err)
		}
	}

	station := models.Station{
		Name:      req.Name,
		Type:      req.Type,
		PrinterID: req.PrinterID,
		DisplayID: req.DisplayID,
		IsActive:  req.IsActive,
	}

	return s.repos.Station.Create(ctx, station)
}

// UpdateStation updates a station
func (s *StationService) UpdateStation(ctx context.Context, id uuid.UUID, req models.StationRequest) (*models.Station, error) {
	// Verify the station exists
	_, err := s.repos.Station.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("station not found: %w", err)
	}

	// Validate printer ID if provided
	if req.PrinterID != nil {
		_, err := s.repos.Printer.GetPrinterByID(ctx, *req.PrinterID)
		if err != nil {
			return nil, fmt.Errorf("invalid printer ID: %w", err)
		}
	}

	// Validate display ID if provided
	if req.DisplayID != nil {
		_, err := s.repos.Printer.GetDisplayByID(ctx, *req.DisplayID)
		if err != nil {
			return nil, fmt.Errorf("invalid display ID: %w", err)
		}
	}

	station := models.Station{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		PrinterID: req.PrinterID,
		DisplayID: req.DisplayID,
		IsActive:  req.IsActive,
	}

	return s.repos.Station.Update(ctx, station)
}

// DeleteStation deletes a station
func (s *StationService) DeleteStation(ctx context.Context, id uuid.UUID) error {
	return s.repos.Station.Delete(ctx, id)
}

// GetRoutingRules retrieves routing rules for a menu item
func (s *StationService) GetRoutingRules(ctx context.Context, menuItemID uuid.UUID) ([]models.RoutingRule, error) {
	// Implementation would retrieve routing rules from the database
	// For now, we'll just return a placeholder
	return []models.RoutingRule{}, nil
}

// UpdateRoutingRule updates a routing rule
func (s *StationService) UpdateRoutingRule(ctx context.Context, id uuid.UUID, req models.RoutingRuleRequest) (*models.RoutingRule, error) {
	// Implementation would update a routing rule in the database
	// For now, we'll just return a placeholder
	return &models.RoutingRule{}, nil
}

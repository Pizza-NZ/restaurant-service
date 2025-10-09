package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// StationRepository defines the database methods needed by the StationService.
type StationRepository interface {
	List(ctx context.Context) ([]models.Station, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Station, error)
	Create(ctx context.Context, station models.Station) (*models.Station, error)
	Update(ctx context.Context, station models.Station) (*models.Station, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PrinterRepositoryForStation defines the printer-related methods needed by the StationService.
type PrinterRepositoryForStation interface {
	GetPrinterByID(ctx context.Context, id uuid.UUID) (*models.Printer, error)
}

// DisplayRepositoryForStation defines the display-related methods needed by the StationService.
type DisplayRepositoryForStation interface {
	GetDisplayByID(ctx context.Context, id uuid.UUID) (*models.Display, error)
}

// StationService handles station-related business logic
type StationService struct {
	stationRepo StationRepository
	printerRepo PrinterRepositoryForStation
	displayRepo DisplayRepositoryForStation
}

// NewStationService creates a new station service
func NewStationService(stationRepo StationRepository, printerRepo PrinterRepositoryForStation, displayRepo DisplayRepositoryForStation) *StationService {
	return &StationService{
		stationRepo: stationRepo,
		printerRepo: printerRepo,
		displayRepo: displayRepo,
	}
}

// GetStations retrieves all stations
func (s *StationService) GetStations(ctx context.Context) ([]models.Station, error) {
	return s.stationRepo.List(ctx)
}

// GetStation retrieves a station by ID
func (s *StationService) GetStation(ctx context.Context, id uuid.UUID) (*models.Station, error) {
	return s.stationRepo.GetByID(ctx, id)
}

// CreateStation creates a new station
func (s *StationService) CreateStation(ctx context.Context, req models.StationRequest) (*models.Station, error) {
	// Validate printer ID if provided
	if req.PrinterID != nil {
		_, err := s.printerRepo.GetPrinterByID(ctx, *req.PrinterID)
		if err != nil {
			return nil, fmt.Errorf("invalid printer ID: %w", err)
		}
	}

	// Validate display ID if provided
	if req.DisplayID != nil {
		_, err := s.displayRepo.GetDisplayByID(ctx, *req.DisplayID)
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

	return s.stationRepo.Create(ctx, station)
}

// UpdateStation updates a station
func (s *StationService) UpdateStation(ctx context.Context, id uuid.UUID, req models.StationRequest) (*models.Station, error) {
	// Verify the station exists
	_, err := s.stationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("station not found: %w", err)
	}

	// Validate printer ID if provided
	if req.PrinterID != nil {
		_, err := s.printerRepo.GetPrinterByID(ctx, *req.PrinterID)
		if err != nil {
			return nil, fmt.Errorf("invalid printer ID: %w", err)
		}
	}

	// Validate display ID if provided
	if req.DisplayID != nil {
		_, err := s.displayRepo.GetDisplayByID(ctx, *req.DisplayID)
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

	return s.stationRepo.Update(ctx, station)
}

// DeleteStation deletes a station
func (s *StationService) DeleteStation(ctx context.Context, id uuid.UUID) error {
	return s.stationRepo.Delete(ctx, id)
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

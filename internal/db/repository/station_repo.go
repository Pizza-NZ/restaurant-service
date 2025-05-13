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

// StationRepository handles station data access
type StationRepository struct {
	db *sqlx.DB
}

// NewStationRepository creates a new station repository
func NewStationRepository(db *sqlx.DB) *StationRepository {
	return &StationRepository{db: db}
}

// GetByID retrieves a station by ID
func (r *StationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Station, error) {
	query := `
		SELECT id, name, type, printer_id, display_id, is_active, created_at, updated_at
		FROM stations
		WHERE id = $1
	`

	var station models.Station
	err := r.db.GetContext(ctx, &station, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	// Get printer if associated
	if station.PrinterID != nil {
		printer, err := r.getPrinter(ctx, *station.PrinterID)
		if err != nil {
			return nil, fmt.Errorf("failed to get station printer: %w", err)
		}
		station.Printer = printer
	}

	// Get display if associated
	if station.DisplayID != nil {
		display, err := r.getDisplay(ctx, *station.DisplayID)
		if err != nil {
			return nil, fmt.Errorf("failed to get station display: %w", err)
		}
		station.Display = display
	}

	return &station, nil
}

// getPrinter retrieves a printer by ID (helper method)
func (r *StationRepository) getPrinter(ctx context.Context, id uuid.UUID) (*models.Printer, error) {
	query := `
		SELECT id, name, type, ip_address, port, model, is_default, is_active, created_at, updated_at
		FROM printers
		WHERE id = $1
	`

	var printer models.Printer
	err := r.db.GetContext(ctx, &printer, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get printer: %w", err)
	}

	return &printer, nil
}

// getDisplay retrieves a display by ID (helper method)
func (r *StationRepository) getDisplay(ctx context.Context, id uuid.UUID) (*models.Display, error) {
	query := `
		SELECT id, name, type, ip_address, is_active, created_at, updated_at
		FROM displays
		WHERE id = $1
	`

	var display models.Display
	err := r.db.GetContext(ctx, &display, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get display: %w", err)
	}

	return &display, nil
}

// List retrieves all stations
func (r *StationRepository) List(ctx context.Context) ([]models.Station, error) {
	query := `
		SELECT id, name, type, printer_id, display_id, is_active, created_at, updated_at
		FROM stations
		ORDER BY name ASC
	`

	var stations []models.Station
	err := r.db.SelectContext(ctx, &stations, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list stations: %w", err)
	}

	// Load printers and displays in a batch to avoid N+1 queries
	printerIDs := make([]uuid.UUID, 0)
	displayIDs := make([]uuid.UUID, 0)

	for _, station := range stations {
		if station.PrinterID != nil {
			printerIDs = append(printerIDs, *station.PrinterID)
		}
		if station.DisplayID != nil {
			displayIDs = append(displayIDs, *station.DisplayID)
		}
	}

	// Get all printers in one query
	printers := make(map[uuid.UUID]*models.Printer)
	if len(printerIDs) > 0 {
		query := `
			SELECT id, name, type, ip_address, port, model, is_default, is_active, created_at, updated_at
			FROM printers
			WHERE id IN (?)
		`
		query, args, err := sqlx.In(query, printerIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare printer query: %w", err)
		}

		query = r.db.Rebind(query)
		var printersList []models.Printer
		err = r.db.SelectContext(ctx, &printersList, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to get printers: %w", err)
		}

		for i := range printersList {
			printers[printersList[i].ID] = &printersList[i]
		}
	}

	// Get all displays in one query
	displays := make(map[uuid.UUID]*models.Display)
	if len(displayIDs) > 0 {
		query := `
			SELECT id, name, type, ip_address, is_active, created_at, updated_at
			FROM displays
			WHERE id IN (?)
		`
		query, args, err := sqlx.In(query, displayIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare display query: %w", err)
		}

		query = r.db.Rebind(query)
		var displaysList []models.Display
		err = r.db.SelectContext(ctx, &displaysList, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to get displays: %w", err)
		}

		for i := range displaysList {
			displays[displaysList[i].ID] = &displaysList[i]
		}
	}

	// Associate printers and displays with stations
	for i := range stations {
		if stations[i].PrinterID != nil {
			if printer, ok := printers[*stations[i].PrinterID]; ok {
				stations[i].Printer = printer
			}
		}

		if stations[i].DisplayID != nil {
			if display, ok := displays[*stations[i].DisplayID]; ok {
				stations[i].Display = display
			}
		}
	}

	return stations, nil
}

// Create creates a new station
func (r *StationRepository) Create(ctx context.Context, station models.Station) (*models.Station, error) {
	query := `
		INSERT INTO stations (name, type, printer_id, display_id, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, type, printer_id, display_id, is_active, created_at, updated_at
	`

	var createdStation models.Station
	err := r.db.GetContext(
		ctx,
		&createdStation,
		query,
		station.Name,
		station.Type,
		station.PrinterID,
		station.DisplayID,
		station.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create station: %w", err)
	}

	// Get printer if associated
	if createdStation.PrinterID != nil {
		printer, err := r.getPrinter(ctx, *createdStation.PrinterID)
		if err != nil {
			return nil, fmt.Errorf("failed to get station printer: %w", err)
		}
		createdStation.Printer = printer
	}

	// Get display if associated
	if createdStation.DisplayID != nil {
		display, err := r.getDisplay(ctx, *createdStation.DisplayID)
		if err != nil {
			return nil, fmt.Errorf("failed to get station display: %w", err)
		}
		createdStation.Display = display
	}

	return &createdStation, nil
}

// Update updates a station
func (r *StationRepository) Update(ctx context.Context, station models.Station) (*models.Station, error) {
	query := `
		UPDATE stations
		SET name = $1, type = $2, printer_id = $3, display_id = $4, is_active = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, name, type, printer_id, display_id, is_active, created_at, updated_at
	`

	var updatedStation models.Station
	err := r.db.GetContext(
		ctx,
		&updatedStation,
		query,
		station.Name,
		station.Type,
		station.PrinterID,
		station.DisplayID,
		station.IsActive,
		time.Now(),
		station.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update station: %w", err)
	}

	// Get printer if associated
	if updatedStation.PrinterID != nil {
		printer, err := r.getPrinter(ctx, *updatedStation.PrinterID)
		if err != nil {
			return nil, fmt.Errorf("failed to get station printer: %w", err)
		}
		updatedStation.Printer = printer
	}

	// Get display if associated
	if updatedStation.DisplayID != nil {
		display, err := r.getDisplay(ctx, *updatedStation.DisplayID)
		if err != nil {
			return nil, fmt.Errorf("failed to get station display: %w", err)
		}
		updatedStation.Display = display
	}

	return &updatedStation, nil
}

// Delete deletes a station
func (r *StationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if there are any routing rules using this station
	var count int
	err := r.db.GetContext(
		ctx,
		&count,
		"SELECT COUNT(*) FROM routing_rules WHERE station_id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to check station usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete station with %d routing rules", count)
	}

	// Delete the station
	query := `
		DELETE FROM stations
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete station: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("station not found")
	}

	return nil
}

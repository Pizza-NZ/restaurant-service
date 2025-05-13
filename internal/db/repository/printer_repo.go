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

// PrinterRepository handles printer and display data access
type PrinterRepository struct {
	db *sqlx.DB
}

// NewPrinterRepository creates a new printer repository
func NewPrinterRepository(db *sqlx.DB) *PrinterRepository {
	return &PrinterRepository{db: db}
}

// GetPrinterByID retrieves a printer by ID
func (r *PrinterRepository) GetPrinterByID(ctx context.Context, id uuid.UUID) (*models.Printer, error) {
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

// ListPrinters retrieves all printers
func (r *PrinterRepository) ListPrinters(ctx context.Context) ([]models.Printer, error) {
	query := `
		SELECT id, name, type, ip_address, port, model, is_default, is_active, created_at, updated_at
		FROM printers
		ORDER BY name ASC
	`

	var printers []models.Printer
	err := r.db.SelectContext(ctx, &printers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list printers: %w", err)
	}

	return printers, nil
}

// GetDefaultPrinter retrieves the default printer
func (r *PrinterRepository) GetDefaultPrinter(ctx context.Context) (*models.Printer, error) {
	query := `
		SELECT id, name, type, ip_address, port, model, is_default, is_active, created_at, updated_at
		FROM printers
		WHERE is_default = true AND is_active = true
		LIMIT 1
	`

	var printer models.Printer
	err := r.db.GetContext(ctx, &printer, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get default printer: %w", err)
	}

	return &printer, nil
}

// CreatePrinter creates a new printer
func (r *PrinterRepository) CreatePrinter(ctx context.Context, printer models.Printer) (*models.Printer, error) {
	// Start a transaction to handle the default printer logic
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// If this printer is set as default, unset any existing default
	if printer.IsDefault {
		_, err = tx.ExecContext(
			ctx,
			"UPDATE printers SET is_default = false WHERE is_default = true",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to unset default printers: %w", err)
		}
	}

	// Insert the printer
	query := `
		INSERT INTO printers (name, type, ip_address, port, model, is_default, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, type, ip_address, port, model, is_default, is_active, created_at, updated_at
	`

	var createdPrinter models.Printer
	err = tx.GetContext(
		ctx,
		&createdPrinter,
		query,
		printer.Name,
		printer.Type,
		printer.IPAddress,
		printer.Port,
		printer.Model,
		printer.IsDefault,
		printer.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create printer: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &createdPrinter, nil
}

// UpdatePrinter updates a printer
func (r *PrinterRepository) UpdatePrinter(ctx context.Context, printer models.Printer) (*models.Printer, error) {
	// Start a transaction to handle the default printer logic
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// If this printer is set as default, unset any existing default
	if printer.IsDefault {
		_, err = tx.ExecContext(
			ctx,
			"UPDATE printers SET is_default = false WHERE is_default = true AND id != $1",
			printer.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to unset default printers: %w", err)
		}
	}

	// Update the printer
	query := `
		UPDATE printers
		SET name = $1, type = $2, ip_address = $3, port = $4, model = $5, is_default = $6, is_active = $7, updated_at = $8
		WHERE id = $9
		RETURNING id, name, type, ip_address, port, model, is_default, is_active, created_at, updated_at
	`

	var updatedPrinter models.Printer
	err = tx.GetContext(
		ctx,
		&updatedPrinter,
		query,
		printer.Name,
		printer.Type,
		printer.IPAddress,
		printer.Port,
		printer.Model,
		printer.IsDefault,
		printer.IsActive,
		time.Now(),
		printer.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update printer: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &updatedPrinter, nil
}

// DeletePrinter deletes a printer
func (r *PrinterRepository) DeletePrinter(ctx context.Context, id uuid.UUID) error {
	// Check if there are any stations using this printer
	var count int
	err := r.db.GetContext(
		ctx,
		&count,
		"SELECT COUNT(*) FROM stations WHERE printer_id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to check printer usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete printer used by %d stations", count)
	}

	// Delete the printer
	query := `DELETE FROM printers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete printer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("printer not found")
	}

	return nil
}

// GetDisplayByID retrieves a display by ID
func (r *PrinterRepository) GetDisplayByID(ctx context.Context, id uuid.UUID) (*models.Display, error) {
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

// ListDisplays retrieves all displays
func (r *PrinterRepository) ListDisplays(ctx context.Context) ([]models.Display, error) {
	query := `
		SELECT id, name, type, ip_address, is_active, created_at, updated_at
		FROM displays
		ORDER BY name ASC
	`

	var displays []models.Display
	err := r.db.SelectContext(ctx, &displays, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list displays: %w", err)
	}

	return displays, nil
}

// CreateDisplay creates a new display
func (r *PrinterRepository) CreateDisplay(ctx context.Context, display models.Display) (*models.Display, error) {
	query := `
		INSERT INTO displays (name, type, ip_address, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, type, ip_address, is_active, created_at, updated_at
	`

	var createdDisplay models.Display
	err := r.db.GetContext(
		ctx,
		&createdDisplay,
		query,
		display.Name,
		display.Type,
		display.IPAddress,
		display.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create display: %w", err)
	}

	return &createdDisplay, nil
}

// UpdateDisplay updates a display
func (r *PrinterRepository) UpdateDisplay(ctx context.Context, display models.Display) (*models.Display, error) {
	query := `
		UPDATE displays
		SET name = $1, type = $2, ip_address = $3, is_active = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, name, type, ip_address, is_active, created_at, updated_at
	`

	var updatedDisplay models.Display
	err := r.db.GetContext(
		ctx,
		&updatedDisplay,
		query,
		display.Name,
		display.Type,
		display.IPAddress,
		display.IsActive,
		time.Now(),
		display.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update display: %w", err)
	}

	return &updatedDisplay, nil
}

// DeleteDisplay deletes a display
func (r *PrinterRepository) DeleteDisplay(ctx context.Context, id uuid.UUID) error {
	// Check if there are any stations using this display
	var count int
	err := r.db.GetContext(
		ctx,
		&count,
		"SELECT COUNT(*) FROM stations WHERE display_id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to check display usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete display used by %d stations", count)
	}

	// Delete the display
	query := `DELETE FROM displays WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete display: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("display not found")
	}

	return nil
}

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// PrinterRepository defines the printer-related database methods needed by the PrinterService.
type PrinterRepository interface {
	GetPrinterByID(ctx context.Context, id uuid.UUID) (*models.Printer, error)
	ListPrinters(ctx context.Context) ([]models.Printer, error)
	CreatePrinter(ctx context.Context, printer models.Printer) (*models.Printer, error)
	UpdatePrinter(ctx context.Context, printer models.Printer) (*models.Printer, error)
	DeletePrinter(ctx context.Context, id uuid.UUID) error
	GetDefaultPrinter(ctx context.Context) (*models.Printer, error)
}

// DisplayRepository defines the display-related database methods needed by the PrinterService.
type DisplayRepository interface {
	GetDisplayByID(ctx context.Context, id uuid.UUID) (*models.Display, error)
	ListDisplays(ctx context.Context) ([]models.Display, error)
	CreateDisplay(ctx context.Context, display models.Display) (*models.Display, error)
	UpdateDisplay(ctx context.Context, display models.Display) (*models.Display, error)
	DeleteDisplay(ctx context.Context, id uuid.UUID) error
}

// OrderRepositoryForPrinter defines the order-related database methods needed by the PrinterService.
type OrderRepositoryForPrinter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
}

// StationRepositoryForPrinter defines the station-related methods needed by the PrinterService.
type StationRepositoryForPrinter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Station, error)
}

// PrinterService handles printer-related operations
type PrinterService struct {
	printerRepo PrinterRepository
	displayRepo DisplayRepository
	orderRepo   OrderRepositoryForPrinter
	stationRepo StationRepositoryForPrinter
}

// NewPrinterService creates a new printer service
func NewPrinterService(printerRepo PrinterRepository, displayRepo DisplayRepository, orderRepo OrderRepositoryForPrinter, stationRepo StationRepositoryForPrinter) *PrinterService {
	return &PrinterService{
		printerRepo: printerRepo,
		displayRepo: displayRepo,
		orderRepo:   orderRepo,
		stationRepo: stationRepo,
	}
}

// ProcessAndPrintOrder fetches order details and routes items to the correct printers.
func (s *PrinterService) ProcessAndPrintOrder(ctx context.Context, orderID uuid.UUID) error {
	logger.Info("Processing order", "order_id", orderID)

	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("could not fetch order details: %w", err)
	}

	// Group items by their assigned station
	stationItems := make(map[uuid.UUID][]models.OrderItem)
	for _, item := range order.Items {
		stationItems[item.StationID] = append(stationItems[item.StationID], item)
	}

	// For each station, find its printer and print the relevant items
	for stationID, items := range stationItems {
		station, err := s.stationRepo.GetByID(ctx, stationID)
		if err != nil {
			logger.Warn("Could not find station for order. Skipping items.", "station_id", stationID, "order_id", orderID)
			continue
		}

		if station.PrinterID == nil {
			logger.Info("Station has no printer assigned. Skipping items.", "station_name", station.Name)
			continue
		}

		// This is the network operation that might fail and cause a retry.
		err = s.PrintOrderItems(ctx, order, items, *station.PrinterID)
		if err != nil {
			// Return the error to trigger the backoff retry.
			return fmt.Errorf("failed to print for station '%s' (printer %s): %w", station.Name, *station.PrinterID, err)
		}
	}

	// After all station tickets are printed, print the main customer receipt.
	receiptPrinter, err := s.printerRepo.GetDefaultPrinter(ctx)
	if err != nil {
		logger.Warn("No default receipt printer found for order.", "order_id", orderID)
	} else {
		if err := s.PrintReceipt(ctx, order, receiptPrinter.ID); err != nil {
			// This is a non-critical error, so we just log it and don't fail the whole operation.
			logger.Error("Error printing customer receipt for order", "order_id", orderID, "error", err)
		}
	}

	logger.Info("Successfully processed and printed all items for order", "order_id", orderID)
	return nil
}

// PrintOrderItems prints order items to a specific printer
func (s *PrinterService) PrintOrderItems(ctx context.Context, order *models.Order, items []models.OrderItem, printerID uuid.UUID) error {
	printer, err := s.printerRepo.GetPrinterByID(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to get printer: %w", err)
	}

	// In a real implementation, this would connect to the printer's IP and send ESC/POS commands.
	// For now, we'll just log it to simulate the action.
	content := s.generateItemsText(order, items)
	fmt.Printf("--- PRINTING TO %s (%s) ---\n%s\n--- END PRINT ---\n", printer.Name, printer.Type, content)

	return nil
}

// PrintReceipt prints an order receipt to a specific printer
func (s *PrinterService) PrintReceipt(ctx context.Context, order *models.Order, printerID uuid.UUID) error {
	printer, err := s.printerRepo.GetPrinterByID(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to get printer: %w", err)
	}

	content, err := s.GenerateReceiptText(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("failed to generate receipt text: %w", err)
	}

	fmt.Printf("--- PRINTING RECEIPT TO %s (%s) ---\n%s\n--- END PRINT ---\n", printer.Name, printer.Type, content)
	return nil
}

// ReprintReceipt handles the logic for re-printing a receipt for a given order.
func (s *PrinterService) ReprintReceipt(ctx context.Context, orderID uuid.UUID, printerID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order for reprint: %w", err)
	}

	return s.PrintReceipt(ctx, order, printerID)
}

// GenerateReceiptText generates a formatted text receipt for an order.
func (s *PrinterService) GenerateReceiptText(ctx context.Context, orderID uuid.UUID) (string, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("failed to get order: %w", err)
	}
	var sb strings.Builder

	sb.WriteString("===============================\n")
	sb.WriteString("          RECEIPT             \n")
	sb.WriteString("===============================\n\n")
	sb.WriteString(fmt.Sprintf("Order #: %s\n", order.OrderNumber))
	sb.WriteString(fmt.Sprintf("Date: %s\n", order.OrderedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString("\nItems:\n")
	sb.WriteString("-------------------------------\n")

	for _, item := range order.Items {
		sb.WriteString(fmt.Sprintf("%dx %s\n", item.Quantity, item.Name))
		for _, mod := range item.Modifiers {
			sb.WriteString(fmt.Sprintf("  + %s", mod.Name))
			if mod.PriceAdjustment > 0 {
				sb.WriteString(fmt.Sprintf(" (+$%.2f)", mod.PriceAdjustment))
			}
			sb.WriteString("\n")
		}
		if item.SpecialInstructions != nil && *item.SpecialInstructions != "" {
			sb.WriteString(fmt.Sprintf("  * %s\n", *item.SpecialInstructions))
		}
		sb.WriteString(fmt.Sprintf("  $%.2f\n\n", item.Price*float64(item.Quantity)))
	}

	sb.WriteString("-------------------------------\n")
	sb.WriteString(fmt.Sprintf("Total: $%.2f\n\n", order.Total))
	sb.WriteString("===============================\n")
	sb.WriteString("         Thank You!           \n")
	sb.WriteString("===============================\n")

	return sb.String(), nil
}

// generateItemsText generates text for printing order items to a kitchen/bar station.
func (s *PrinterService) generateItemsText(order *models.Order, items []models.OrderItem) string {
	var sb strings.Builder

	sb.WriteString("===============================\n")
	sb.WriteString(fmt.Sprintf("          ORDER #%s         \n", order.OrderNumber))
	sb.WriteString("===============================\n\n")
	sb.WriteString(fmt.Sprintf("Time: %s\n\n", order.OrderedAt.Format("15:04:05")))

	for _, item := range items {
		sb.WriteString(fmt.Sprintf("%dx %s\n", item.Quantity, item.Name))
		for _, mod := range item.Modifiers {
			sb.WriteString(fmt.Sprintf("  + %s\n", mod.Name))
		}
		if item.SpecialInstructions != nil && *item.SpecialInstructions != "" {
			sb.WriteString(fmt.Sprintf("  *** %s\n", *item.SpecialInstructions))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("-------------------------------\n")
	return sb.String()
}

// GetPrinters retrieves all printers
func (s *PrinterService) GetPrinters(ctx context.Context) ([]models.Printer, error) {
	return s.printerRepo.ListPrinters(ctx)
}

// GetPrinter retrieves a printer by ID
func (s *PrinterService) GetPrinter(ctx context.Context, id uuid.UUID) (*models.Printer, error) {
	return s.printerRepo.GetPrinterByID(ctx, id)
}

// CreatePrinter creates a new printer
func (s *PrinterService) CreatePrinter(ctx context.Context, req models.PrinterRequest) (*models.Printer, error) {
	printer := models.Printer{
		Name:      req.Name,
		Type:      req.Type,
		IPAddress: req.IPAddress,
		Port:      req.Port,
		Model:     req.Model,
		IsDefault: req.IsDefault,
		IsActive:  req.IsActive,
	}

	return s.printerRepo.CreatePrinter(ctx, printer)
}

// UpdatePrinter updates a printer
func (s *PrinterService) UpdatePrinter(ctx context.Context, id uuid.UUID, req models.PrinterRequest) (*models.Printer, error) {
	printer := models.Printer{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		IPAddress: req.IPAddress,
		Port:      req.Port,
		Model:     req.Model,
		IsDefault: req.IsDefault,
		IsActive:  req.IsActive,
	}

	return s.printerRepo.UpdatePrinter(ctx, printer)
}

// DeletePrinter deletes a printer
func (s *PrinterService) DeletePrinter(ctx context.Context, id uuid.UUID) error {
	return s.printerRepo.DeletePrinter(ctx, id)
}

// GetDisplays retrieves all displays
func (s *PrinterService) GetDisplays(ctx context.Context) ([]models.Display, error) {
	return s.displayRepo.ListDisplays(ctx)
}

// GetDisplay retrieves a display by ID
func (s *PrinterService) GetDisplay(ctx context.Context, id uuid.UUID) (*models.Display, error) {
	return s.displayRepo.GetDisplayByID(ctx, id)
}

// CreateDisplay creates a new display
func (s *PrinterService) CreateDisplay(ctx context.Context, req models.DisplayRequest) (*models.Display, error) {
	display := models.Display{
		Name:      req.Name,
		Type:      req.Type,
		IPAddress: req.IPAddress,
		IsActive:  req.IsActive,
	}

	return s.displayRepo.CreateDisplay(ctx, display)
}

// UpdateDisplay updates a display
func (s *PrinterService) UpdateDisplay(ctx context.Context, id uuid.UUID, req models.DisplayRequest) (*models.Display, error) {
	display := models.Display{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		IPAddress: req.IPAddress,
		IsActive:  req.IsActive,
	}

	return s.displayRepo.UpdateDisplay(ctx, display)
}

// DeleteDisplay deletes a display
func (s *PrinterService) DeleteDisplay(ctx context.Context, id uuid.UUID) error {
	return s.displayRepo.DeleteDisplay(ctx, id)
}

// TestPrinter sends a test page to a printer.
func (s *PrinterService) TestPrinter(ctx context.Context, id uuid.UUID) error {
	printer, err := s.printerRepo.GetPrinterByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get printer: %w", err)
	}

	content := "===============================\n" +
		"          TEST PAGE            \n" +
		"===============================\n\n" +
		"This is a test page.\n" +
		"If you can read this, the printer is working.\n\n" +
		fmt.Sprintf("Printer: %s\n", printer.Name) +
		fmt.Sprintf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05")) +
		"\n===============================\n"

	fmt.Printf("--- PRINTING TEST PAGE TO %s (%s) ---\n%s\n--- END PRINT ---\n", printer.Name, printer.Type, content)
	return nil
}

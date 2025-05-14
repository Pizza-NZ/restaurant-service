package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/models"
)

// PrinterService handles printer-related operations
type PrinterService struct {
	repos *repository.Repositories
}

// NewPrinterService creates a new printer service
func NewPrinterService(repos *repository.Repositories) *PrinterService {
	return &PrinterService{
		repos: repos,
	}
}

// PrintOrderItems prints order items to a specific printer
func (s *PrinterService) PrintOrderItems(ctx context.Context, order *models.Order, items []models.OrderItem, printerID uuid.UUID) error {
	// Get the printer
	printer, err := s.repos.Printer.GetPrinterByID(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to get printer: %w", err)
	}

	// Generate the print content
	content := s.generateItemsText(order, items)

	// In a real implementation, this would send the content to the printer
	// For now, we'll just log it
	fmt.Printf("Printing to %s (%s):\n%s\n", printer.Name, printer.ID, content)

	return nil
}

// PrintReceipt prints an order receipt to a specific printer
func (s *PrinterService) PrintReceipt(ctx context.Context, order *models.Order, printerID uuid.UUID) error {
	// Get the printer
	printer, err := s.repos.Printer.GetPrinterByID(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to get printer: %w", err)
	}

	// Generate the receipt content
	content, err := s.GenerateReceiptText(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to generate receipt: %w", err)
	}

	// In a real implementation, this would send the content to the printer
	// For now, we'll just log it
	fmt.Printf("Printing receipt to %s (%s):\n%s\n", printer.Name, printer.ID, content)

	return nil
}

// GenerateReceiptText generates a text receipt for an order
func (s *PrinterService) GenerateReceiptText(ctx context.Context, order *models.Order) (string, error) {
	var sb strings.Builder

	// Add header
	sb.WriteString("===============================\n")
	sb.WriteString("          RECEIPT             \n")
	sb.WriteString("===============================\n\n")

	// Add order info
	sb.WriteString(fmt.Sprintf("Order #: %s\n", order.OrderNumber))
	sb.WriteString(fmt.Sprintf("Date: %s\n", order.OrderedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString("\n")

	// Add items
	sb.WriteString("Items:\n")
	sb.WriteString("-------------------------------\n")

	for _, item := range order.Items {
		sb.WriteString(fmt.Sprintf("%dx %s\n", item.Quantity, item.Name))

		// Add modifiers
		for _, mod := range item.Modifiers {
			sb.WriteString(fmt.Sprintf("  + %s", mod.Name))
			if mod.PriceAdjustment > 0 {
				sb.WriteString(fmt.Sprintf(" (+$%.2f)", mod.PriceAdjustment))
			}
			sb.WriteString("\n")
		}

		// Add special instructions
		if item.SpecialInstructions != nil && *item.SpecialInstructions != "" {
			sb.WriteString(fmt.Sprintf("  * %s\n", *item.SpecialInstructions))
		}

		// Add item price
		sb.WriteString(fmt.Sprintf("  $%.2f\n", item.Price*float64(item.Quantity)))
		sb.WriteString("\n")
	}

	// Add totals
	sb.WriteString("-------------------------------\n")
	sb.WriteString(fmt.Sprintf("Total: $%.2f\n", order.Total))
	sb.WriteString("\n")

	// Add footer
	sb.WriteString("===============================\n")
	sb.WriteString("         Thank You!           \n")
	sb.WriteString("===============================\n")

	return sb.String(), nil
}

// generateItemsText generates text for printing order items
func (s *PrinterService) generateItemsText(order *models.Order, items []models.OrderItem) string {
	var sb strings.Builder

	// Add header
	sb.WriteString("===============================\n")
	sb.WriteString("          ORDER ITEMS         \n")
	sb.WriteString("===============================\n\n")

	// Add order info
	sb.WriteString(fmt.Sprintf("Order #: %s\n", order.OrderNumber))
	sb.WriteString(fmt.Sprintf("Date: %s\n", order.OrderedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString("\n")

	// Add items
	sb.WriteString("Items:\n")
	sb.WriteString("-------------------------------\n")

	for _, item := range items {
		sb.WriteString(fmt.Sprintf("%dx %s\n", item.Quantity, item.Name))

		// Add modifiers
		for _, mod := range item.Modifiers {
			sb.WriteString(fmt.Sprintf("  + %s\n", mod.Name))
		}

		// Add special instructions
		if item.SpecialInstructions != nil && *item.SpecialInstructions != "" {
			sb.WriteString(fmt.Sprintf("  * %s\n", *item.SpecialInstructions))
		}

		sb.WriteString("\n")
	}

	// Add footer with current time
	sb.WriteString("-------------------------------\n")
	sb.WriteString(fmt.Sprintf("Printed: %s\n", time.Now().Format("15:04:05")))

	return sb.String()
}

// GetPrinters retrieves all printers
func (s *PrinterService) GetPrinters(ctx context.Context) ([]models.Printer, error) {
	return s.repos.Printer.ListPrinters(ctx)
}

// GetPrinter retrieves a printer by ID
func (s *PrinterService) GetPrinter(ctx context.Context, id uuid.UUID) (*models.Printer, error) {
	return s.repos.Printer.GetPrinterByID(ctx, id)
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

	return s.repos.Printer.CreatePrinter(ctx, printer)
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

	return s.repos.Printer.UpdatePrinter(ctx, printer)
}

// DeletePrinter deletes a printer
func (s *PrinterService) DeletePrinter(ctx context.Context, id uuid.UUID) error {
	return s.repos.Printer.DeletePrinter(ctx, id)
}

// GetDisplays retrieves all displays
func (s *PrinterService) GetDisplays(ctx context.Context) ([]models.Display, error) {
	return s.repos.Printer.ListDisplays(ctx)
}

// GetDisplay retrieves a display by ID
func (s *PrinterService) GetDisplay(ctx context.Context, id uuid.UUID) (*models.Display, error) {
	return s.repos.Printer.GetDisplayByID(ctx, id)
}

// CreateDisplay creates a new display
func (s *PrinterService) CreateDisplay(ctx context.Context, req models.DisplayRequest) (*models.Display, error) {
	display := models.Display{
		Name:      req.Name,
		Type:      req.Type,
		IPAddress: req.IPAddress,
		IsActive:  req.IsActive,
	}

	return s.repos.Printer.CreateDisplay(ctx, display)
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

	return s.repos.Printer.UpdateDisplay(ctx, display)
}

// DeleteDisplay deletes a display
func (s *PrinterService) DeleteDisplay(ctx context.Context, id uuid.UUID) error {
	return s.repos.Printer.DeleteDisplay(ctx, id)
}

// TestPrinter tests a printer by sending a test page
func (s *PrinterService) TestPrinter(ctx context.Context, id uuid.UUID) error {
	// Get the printer
	printer, err := s.repos.Printer.GetPrinterByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get printer: %w", err)
	}

	// Generate test content
	content := "===============================\n" +
		"          TEST PAGE            \n" +
		"===============================\n\n" +
		"This is a test page.\n" +
		"If you can read this, the printer is working.\n\n" +
		fmt.Sprintf("Printer: %s\n", printer.Name) +
		fmt.Sprintf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05")) +
		"\n===============================\n"

	// In a real implementation, this would send the content to the printer
	// For now, we'll just log it
	fmt.Printf("Printing test page to %s (%s):\n%s\n", printer.Name, printer.ID, content)

	return nil
}

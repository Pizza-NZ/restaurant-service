package repository

import (
	"github.com/jmoiron/sqlx"
)

// Factory provides access to all repositories
type Factory struct {
	User    *UserRepository
	Menu    *MenuRepository
	Order   *OrderRepository
	Station *StationRepository
	Printer *PrinterRepository
}

// NewFactory creates a new repository factory
func NewFactory(db *sqlx.DB) *Factory {
	return &Factory{
		User:    NewUserRepository(db),
		Menu:    NewMenuRepository(db),
		Order:   NewOrderRepository(db),
		Station: NewStationRepository(db),
		Printer: NewPrinterRepository(db),
	}
}

package repository

import (
	"github.com/pizza-nz/restaurant-service/internal/db"
)

// Repositories provides access to all repository instances
type Repositories struct {
	User    *UserRepository
	Menu    *MenuRepository
	Order   *OrderRepository
	Station *StationRepository
	Printer *PrinterRepository
}

// NewRepositories creates a new repositories container
func NewRepositories(database *db.Postgres) *Repositories {
	return &Repositories{
		User:    NewUserRepository(database.DB),
		Menu:    NewMenuRepository(database.DB),
		Order:   NewOrderRepository(database.DB),
		Station: NewStationRepository(database.DB),
		Printer: NewPrinterRepository(database.DB),
	}
}

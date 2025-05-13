package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pizza-nz/restaurant-service/internal/config"
)

type Postgres struct {
	DB *sqlx.DB
}

func NewPostgres(cfg config.Database) (*Postgres, error) {
	// Create connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	// Connect with retries - helpful for system startup scenarios
	var db *sqlx.DB
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("postgres", connStr)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Duration(i+1) * 2 * time.Second) // Exponential backoff
	}

	if err != nil {
		return nil, fmt.Errorf("could not connect to database after %d attempts: %w", maxRetries, err)
	}

	// Configure connection pool for low-resource environment
	// These settings are conservative for Raspberry Pi
	db.SetMaxOpenConns(10)                  // Limit concurrent connections
	db.SetMaxIdleConns(5)                   // Keep some connections ready
	db.SetConnMaxLifetime(time.Hour)        // Recycle connections periodically
	db.SetConnMaxIdleTime(30 * time.Minute) // Close idle connections

	// Verify connection is working
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return &Postgres{DB: db}, nil
}

// // NewPostgres creates a new database connection
// func NewPostgres(cfg config.Database) (*Postgres, error) {
// 	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
// 		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

// 	db, err := sqlx.Connect("postgres", connStr)
// 	if err != nil {
// 		return nil, err
// 	}

// 	con := &config.Config{
// 		Database: cfg,
// 	}

// 	// Set connection pool settings
// 	db.SetMaxOpenConns(25)
// 	db.SetMaxIdleConns(5)

// 	return &Postgres{DB: db, cfg: con}, nil
// }

// Close closes the database connection
func (p *Postgres) Close() error {
	return p.DB.Close()
}

// Migrate runs database migrations
func (p *Postgres) Migrate(cfg config.Database) error {
	// Set up migration source and target
	migrationsPath := "file://migrations"
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	// Initialize migrate instance
	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// HealthCheck performs a database health check
func (p *Postgres) HealthCheck(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}

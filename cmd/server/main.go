package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pizza-nz/restaurant-service/internal/config"
	"github.com/pizza-nz/restaurant-service/internal/db"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/router"
	"github.com/pizza-nz/restaurant-service/internal/service"
	"github.com/pizza-nz/restaurant-service/internal/telemetry"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Initialize OpenTelemetry
	tr, err := telemetry.NewTracerProvider("restaurant-service")
	if err != nil {
		logger.Error("Failed to create tracer provider", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := tr.Shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown tracer provider", "error", err)
		}
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize database
	database, err := db.NewPostgres(cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Error("Failed to close database", "error", err)
		}
	}()

	// Run database migrations
	if err := database.Migrate(cfg.Database); err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Repositories (Data Access Layer)
	repos := repository.NewRepositories(database)

	// Initialize WebSocket hub
	hub := websockets.NewHub()
	go hub.Run()

	// Initialize Auth Service
	authService := service.NewAuthService(repos.User, service.JWTConfig(cfg.JWT))

	// Initialize router and middleware chain
	mainHandler := router.New(repos, authService, hub)

	// Create HTTP server
	server := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: mainHandler, // Use the fully configured handler
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", "address", cfg.Server.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited properly")
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pizza-nz/restaurant-service/internal/config"
	"github.com/pizza-nz/restaurant-service/internal/db"
	"github.com/pizza-nz/restaurant-service/internal/router"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	database, err := db.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run database migrations
	if err := database.Migrate(cfg.Database); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize WebSocket hub
	hub := websockets.NewHub()
	go hub.Run()

	// Initialize router
	r := router.New(database, hub, cfg)

	// Create HTTP server
	server := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", cfg.Server.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

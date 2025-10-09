package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/pizza-nz/restaurant-service/internal/config"
	"github.com/pizza-nz/restaurant-service/internal/db"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/service"
	"github.com/pizza-nz/restaurant-service/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// The printerService is the single dependency needed for handling events.
var printerService *service.PrinterService
var logger *slog.Logger

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Initialize OpenTelemetry
	tr, err := telemetry.NewTracerProvider("printer-service")
	if err != nil {
		logger.Error("Failed to create tracer provider", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := tr.Shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown tracer provider", "error", err)
		}
	}()

	// --- Initialize Dependencies ---
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

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

	// Initialize the central repository container
	repos := repository.NewRepositories(database)

	// Correctly instantiate the PrinterService with its required repository interfaces.
	// Note: repos.Printer satisfies both PrinterRepository and DisplayRepository interfaces.
	printerService = service.NewPrinterService(repos.Printer, repos.Printer, repos.Order, repos.Station)

	// --- NATS Connection ---
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Close()
	logger.Info("Connected to NATS", "url", natsURL)

	// --- Graceful Shutdown Setup ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- NATS Subscription ---
	sub, err := nc.Subscribe("orders.created", handleOrderCreated)
	if err != nil {
		logger.Error("Failed to subscribe to 'orders.created'", "error", err)
		os.Exit(1)
	}

	logger.Info("Printer service is running, waiting for orders...")
	<-ctx.Done() // Block until a shutdown signal is received

	logger.Info("Shutdown signal received. Draining NATS subscription...")
	if err := sub.Drain(); err != nil {
		logger.Error("Error draining NATS subscription", "error", err)
	}
	logger.Info("Printer service shut down gracefully.")
}

// handleOrderCreated is the callback function for processing `orders.created` events.
func handleOrderCreated(m *nats.Msg) {
	// Extract the trace context from the NATS message headers.
	propagator := otel.GetTextMapPropagator()
	ctx := propagator.Extract(context.Background(), propagation.HeaderCarrier(m.Header))

	tracer := otel.Tracer("printer-service")
	ctx, span := tracer.Start(ctx, "orders.created", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	orderIDStr := string(m.Data)
	logger.Info("Received 'orders.created' event", "order_id", orderIDStr)

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		logger.Error("Invalid order ID received", "order_id", orderIDStr, "error", err)
		return // Acknowledge and drop the message.
	}

	// This is the operation we want to retry. It now calls the service method.
	operation := func() error {
		// Use a context with a timeout for each attempt.
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		// Delegate the business logic to the PrinterService
		return printerService.ProcessAndPrintOrder(ctx, orderID)
	}

	// Configure exponential backoff policy
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 2 * time.Second
	b.Multiplier = 1.5
	b.MaxInterval = 30 * time.Second
	b.MaxElapsedTime = 5 * time.Minute

	if err := backoff.Retry(operation, b); err != nil {
		logger.Error("CRITICAL: Failed to process and print order after all retries", "order_id", orderID, "error", err)
		// In a real production system, this message would be moved to a dead-letter queue for manual inspection.
	}
}

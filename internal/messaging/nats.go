package messaging

import (
	"context"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// EventPublisher is a wrapper around a NATS connection for publishing events.
type EventPublisher struct {
	nc *nats.Conn
}

// NewEventPublisher connects to NATS and returns a new publisher.
func NewEventPublisher() (*EventPublisher, error) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	slog.Info("Successfully connected to NATS", "url", natsURL)
	return &EventPublisher{nc: nc}, nil
}

// PublishOrderCreated publishes an event indicating a new order has been created.
// The payload is simply the order ID, which consumers can use to look up details.
func (p *EventPublisher) PublishOrderCreated(ctx context.Context, orderID string) error {
	subject := "orders.created"
	payload := []byte(orderID)

	// Inject the trace context into the NATS message headers.
	msg := nats.NewMsg(subject)
	msg.Data = payload
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(msg.Header))

	slog.Info("Publishing event", "subject", subject, "orderID", orderID)
	return p.nc.PublishMsg(msg)
}

// Close gracefully closes the NATS connection.
func (p *EventPublisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

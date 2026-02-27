package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	PublishEvent(ctx context.Context, event DomainEvent) error
	Close() error
}

// WatermillPublisher implements EventPublisher using Watermill
type WatermillPublisher struct {
	pubSub *gochannel.GoChannel
	topic  string
}

var (
	// Global in-memory pubsub for event distribution
	globalPubSub *gochannel.GoChannel
	pubSubOnce   sync.Once
)

// getGlobalPubSub ensures we have a single global pubsub instance
func getGlobalPubSub() *gochannel.GoChannel {
	pubSubOnce.Do(func() {
		globalPubSub = gochannel.NewGoChannel(
			gochannel.Config{
				OutputChannelBuffer: 1000,
			},
			watermill.NewStdLogger(false, false),
		)
	})
	return globalPubSub
}

// NewWatermillPublisher creates a Watermill publisher
// Uses in-memory pub/sub. In production, replace with PostgreSQL pubsub for multi-replica support
func NewWatermillPublisher(db *pgxpool.Pool, topic string) (*WatermillPublisher, error) {
	pubSub := getGlobalPubSub()

	return &WatermillPublisher{
		pubSub: pubSub,
		topic:  topic,
	}, nil
}

// PublishEvent publishes a domain event to the message bus
func (wp *WatermillPublisher) PublishEvent(ctx context.Context, event DomainEvent) error {
	eventData, err := Serialize(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	msg := message.NewMessage(
		event.StreamID().String(), // Message ID for idempotency
		eventData,
	)

	// Add metadata for subscribers
	msg.Metadata.Set("event_type", event.EventType())
	msg.Metadata.Set("stream_id", event.StreamID().String())

	if err := wp.pubSub.Publish(wp.topic, msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Close closes the publisher
func (wp *WatermillPublisher) Close() error {
	return nil
}

// NOTE: Multi-replica support requires upgrading to watermill-postgres v2
// For now, this is a single-instance implementation with in-memory pub/sub
// To enable multi-replica support:
// 1. Add watermill-postgres/v2 dependency
// 2. Replace getGlobalPubSub() with postgres.NewPubSub()
// 3. Update configuration to support PostgreSQL broker parameters

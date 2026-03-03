package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventStore defines the interface for event persistence
type EventStore interface {
	// AppendToStream persists events to a stream within a transaction
	AppendToStream(ctx context.Context, tx pgx.Tx, streamID uuid.UUID, events []DomainEvent, expectedVersion int) error

	// GetEvents loads all events for a stream starting from a specific version
	GetEvents(ctx context.Context, streamID uuid.UUID, fromVersion int) ([]DomainEvent, error)
}

// PostgresEventStore implements EventStore using PostgreSQL
type PostgresEventStore struct {
	db *pgxpool.Pool
}

// NewPostgresEventStore creates a new event store
func NewPostgresEventStore(db *pgxpool.Pool) *PostgresEventStore {
	return &PostgresEventStore{db: db}
}

// AppendToStream persists events with optimistic locking
func (es *PostgresEventStore) AppendToStream(
	ctx context.Context,
	tx pgx.Tx,
	streamID uuid.UUID,
	domainEvents []DomainEvent,
	expectedVersion int,
) error {
	if len(domainEvents) == 0 {
		return errors.New("no events to append")
	}

	q := db.New(tx)

	// Verify expected version matches actual version (optimistic locking)
	actualVersion, err := q.GetStreamVersion(ctx, streamID)
	if err != nil {
		return fmt.Errorf("failed to get current stream version: %w", err)
	}

	if int(actualVersion) != expectedVersion {
		return fmt.Errorf("optimistic lock failed: expected version %d, got %d", expectedVersion, actualVersion)
	}

	// Insert all events in batch
	for i, event := range domainEvents {
		eventData, err := Serialize(event)
		if err != nil {
			return fmt.Errorf("failed to serialize event: %w", err)
		}

		version := expectedVersion + i + 1

		err = q.InsertDateiEvent(ctx, db.InsertDateiEventParams{
			StreamID:      streamID,
			StreamVersion: int32(version),
			EventType:     event.EventType(),
			EventData:     eventData,
		})
		if err != nil {
			return fmt.Errorf("failed to insert event %s: %w", event.EventType(), err)
		}
	}

	return nil
}

// GetEvents loads all events for a stream starting from a specific version
func (es *PostgresEventStore) GetEvents(
	ctx context.Context,
	streamID uuid.UUID,
	fromVersion int,
) ([]DomainEvent, error) {
	q := db.New(es.db)

	rows, err := q.GetDateiEventsByStreamID(ctx, db.GetDateiEventsByStreamIDParams{
		StreamID:      streamID,
		StreamVersion: int32(fromVersion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	var events []DomainEvent
	for _, row := range rows {
		event, err := Deserialize(row.EventType, row.EventData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

package events

import (
	"context"
	"errors"
	"fmt"

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

	// Verify expected version matches actual version (optimistic locking)
	var actualVersion int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(stream_version), 0) FROM event_store WHERE stream_id = $1`,
		streamID,
	).Scan(&actualVersion)
	if err != nil {
		return fmt.Errorf("failed to get current stream version: %w", err)
	}

	if actualVersion != expectedVersion {
		return fmt.Errorf("optimistic lock failed: expected version %d, got %d", expectedVersion, actualVersion)
	}

	// Insert all events in batch
	for i, event := range domainEvents {
		eventData, err := Serialize(event)
		if err != nil {
			return fmt.Errorf("failed to serialize event: %w", err)
		}

		version := expectedVersion + i + 1

		_, err = tx.Exec(ctx,
			`INSERT INTO event_store (stream_id, stream_version, event_type, event_data, created_at)
			 VALUES ($1, $2, $3, $4, NOW())`,
			streamID,
			version,
			event.EventType(),
			eventData,
		)
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
	rows, err := es.db.Query(ctx,
		`SELECT event_type, event_data FROM event_store
		 WHERE stream_id = $1 AND stream_version >= $2
		 ORDER BY stream_version ASC`,
		streamID,
		fromVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []DomainEvent
	for rows.Next() {
		var eventType string
		var eventData []byte
		if err := rows.Scan(&eventType, &eventData); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		event, err := Deserialize(eventType, eventData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

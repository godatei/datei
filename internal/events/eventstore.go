package events

import (
	"context"
	"encoding/json"
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

	// GetEventsWithSnapshot loads events using snapshot optimization
	GetEventsWithSnapshot(ctx context.Context, streamID uuid.UUID) ([]DomainEvent, *Snapshot, error)

	// SaveSnapshot stores an aggregate snapshot
	SaveSnapshot(ctx context.Context, tx pgx.Tx, streamID uuid.UUID, version int, state interface{}) error

	// GetLatestSnapshot retrieves the most recent snapshot
	GetLatestSnapshot(ctx context.Context, streamID uuid.UUID) (*Snapshot, error)
}

// Snapshot represents a stored aggregate state
type Snapshot struct {
	StreamID uuid.UUID
	Version  int
	State    json.RawMessage
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
func (es *PostgresEventStore) AppendToStream(ctx context.Context, tx pgx.Tx, streamID uuid.UUID, domainEvents []DomainEvent, expectedVersion int) error {
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
func (es *PostgresEventStore) GetEvents(ctx context.Context, streamID uuid.UUID, fromVersion int) ([]DomainEvent, error) {
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

// GetEventsWithSnapshot loads events using snapshot optimization
func (es *PostgresEventStore) GetEventsWithSnapshot(ctx context.Context, streamID uuid.UUID) ([]DomainEvent, *Snapshot, error) {
	// Try to load latest snapshot
	snapshot, err := es.GetLatestSnapshot(ctx, streamID)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		// Log but continue - snapshots are optional
	}

	// Load events after snapshot version
	fromVersion := 1
	if snapshot != nil {
		fromVersion = snapshot.Version + 1
	}

	events, err := es.GetEvents(ctx, streamID, fromVersion)
	if err != nil {
		return nil, nil, err
	}

	return events, snapshot, nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a stream
func (es *PostgresEventStore) GetLatestSnapshot(ctx context.Context, streamID uuid.UUID) (*Snapshot, error) {
	var stream uuid.UUID
	var version int
	var state json.RawMessage

	err := es.db.QueryRow(ctx,
		`SELECT stream_id, stream_version, aggregate_state FROM event_snapshots
		 WHERE stream_id = $1
		 ORDER BY stream_version DESC
		 LIMIT 1`,
		streamID,
	).Scan(&stream, &version, &state)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}

	return &Snapshot{
		StreamID: stream,
		Version:  version,
		State:    state,
	}, nil
}

// SaveSnapshot stores an aggregate snapshot within a transaction
func (es *PostgresEventStore) SaveSnapshot(ctx context.Context, tx pgx.Tx, streamID uuid.UUID, version int, state interface{}) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot state: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO event_snapshots (stream_id, stream_version, aggregate_state, created_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (stream_id, stream_version) DO NOTHING`,
		streamID,
		version,
		stateJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	return nil
}

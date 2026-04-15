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

// EventRow holds the fields needed to deserialize a stored event.
type EventRow struct {
	EventType string
	EventData []byte
}

// EventStore defines the interface for event persistence
type EventStore interface {
	// AppendToStream persists events to a stream within a transaction
	AppendToStream(ctx context.Context, tx pgx.Tx, streamID uuid.UUID, events []DomainEvent, expectedVersion int) error

	// GetEvents loads all events for a stream starting from a specific version
	GetEvents(ctx context.Context, streamID uuid.UUID, fromVersion int) ([]DomainEvent, error)
}

// InsertParams groups the parameters for inserting a single event row.
type InsertParams struct {
	StreamID      uuid.UUID
	StreamVersion int32
	EventType     string
	EventData     []byte
}

// StoreQueries abstracts the sqlc query functions so a single store
// implementation can target different event tables.
type StoreQueries struct {
	GetVersion func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error)
	Insert     func(ctx context.Context, q *db.Queries, p InsertParams) error
	GetEvents  func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]EventRow, error)
}

// PostgresEventStore implements EventStore using PostgreSQL.
// It is parameterised with query callbacks so the same logic can serve
// any event table (datei_event, user_account_event, …).
type PostgresEventStore struct {
	db *pgxpool.Pool
	sq StoreQueries
}

// NewStore creates a PostgresEventStore with the given query callbacks.
// Domain packages call this to wire their specific sqlc queries.
func NewStore(pool *pgxpool.Pool, sq StoreQueries) *PostgresEventStore {
	return &PostgresEventStore{db: pool, sq: sq}
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

	actualVersion, err := es.sq.GetVersion(ctx, q, streamID)
	if err != nil {
		return fmt.Errorf("failed to get current stream version: %w", err)
	}

	if int(actualVersion) != expectedVersion {
		return fmt.Errorf("optimistic lock failed: expected version %d, got %d", expectedVersion, actualVersion)
	}

	for i, event := range domainEvents {
		eventData, err := Serialize(event)
		if err != nil {
			return fmt.Errorf("failed to serialize event: %w", err)
		}

		if err := es.sq.Insert(ctx, q, InsertParams{
			StreamID:      streamID,
			StreamVersion: int32(expectedVersion + i + 1),
			EventType:     event.EventType(),
			EventData:     eventData,
		}); err != nil {
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

	rows, err := es.sq.GetEvents(ctx, q, streamID, int32(fromVersion))
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	events := make([]DomainEvent, 0, len(rows))
	for _, row := range rows {
		event, err := Deserialize(row.EventType, row.EventData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

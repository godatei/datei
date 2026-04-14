//nolint:dupl // mirrors eventstore.go by design (event store pattern)
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

// UserAccountEventStore implements EventStore using the user_account_event table
type UserAccountEventStore struct {
	db *pgxpool.Pool
}

// NewUserAccountEventStore creates a new event store for user account events
func NewUserAccountEventStore(db *pgxpool.Pool) *UserAccountEventStore {
	return &UserAccountEventStore{db: db}
}

// AppendToStream persists user account events with optimistic locking
func (es *UserAccountEventStore) AppendToStream(
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

	actualVersion, err := q.GetUserAccountStreamVersion(ctx, streamID)
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

		version := expectedVersion + i + 1

		err = q.InsertUserAccountEvent(ctx, db.InsertUserAccountEventParams{
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

// GetEvents loads all user account events for a stream starting from a specific version
func (es *UserAccountEventStore) GetEvents(
	ctx context.Context,
	streamID uuid.UUID,
	fromVersion int,
) ([]DomainEvent, error) {
	q := db.New(es.db)

	rows, err := q.GetUserAccountEventsByStreamID(ctx, db.GetUserAccountEventsByStreamIDParams{
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

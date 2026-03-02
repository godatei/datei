package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/godatei/datei/internal/projections"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DateiRepository defines the interface for datei persistence
type DateiRepository interface {
	// LoadByID reconstructs a datei aggregate from the event store
	LoadByID(ctx context.Context, id uuid.UUID) (*DateiAggregate, error)

	// Save persists a datei aggregate (events + projections)
	Save(ctx context.Context, aggregate *DateiAggregate) error
}

// PostgresDateiRepository implements DateiRepository
type PostgresDateiRepository struct {
	db         *pgxpool.Pool
	eventStore events.EventStore
}

// NewPostgresDateiRepository creates a new repository
func NewPostgresDateiRepository(db *pgxpool.Pool, eventStore events.EventStore) *PostgresDateiRepository {
	return &PostgresDateiRepository{
		db:         db,
		eventStore: eventStore,
	}
}

// LoadByID reconstructs an aggregate from events (with snapshot optimization)
func (r *PostgresDateiRepository) LoadByID(ctx context.Context, id uuid.UUID) (*DateiAggregate, error) {
	// Load events with snapshot optimization
	eventList, err := r.eventStore.GetEvents(ctx, id, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	if len(eventList) == 0 {
		return nil, fmt.Errorf("datei not found: %w", context.Canceled)
	}

	// Create aggregate and apply events
	agg := &DateiAggregate{}

	// Replay all events to reconstruct current state
	if err := agg.ReplayEvents(eventList); err != nil {
		return nil, fmt.Errorf("failed to replay events: %w", err)
	}

	// Mark events as committed (they're from store, not uncommitted)
	agg.version = len(eventList)
	agg.uncommittedEvents = []events.DomainEvent{}

	return agg, nil
}

// Save persists an aggregate's uncommitted events and updates projections
func (r *PostgresDateiRepository) Save(ctx context.Context, agg *DateiAggregate) (returnErr error) {
	uncommittedEvents := agg.GetUncommittedEvents()
	if len(uncommittedEvents) == 0 {
		return nil // Nothing to save
	}

	// Begin transaction
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			returnErr = errors.Join(returnErr, err)
		}
	}()

	// Append events to event store (with optimistic locking)
	if err := r.eventStore.AppendToStream(
		ctx,
		tx,
		agg.ID,
		uncommittedEvents,
		agg.version-len(uncommittedEvents),
	); err != nil {
		return fmt.Errorf("failed to append events: %w", err)
	}

	// Update projections synchronously (same transaction)
	q := db.New(tx)
	for _, event := range uncommittedEvents {
		if err := r.updateProjection(ctx, q, event); err != nil {
			return fmt.Errorf("failed to update projection for %s: %w", event.EventType(), err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Mark events as committed
	agg.MarkEventsAsCommitted()

	return nil
}

// updateProjection updates read models based on domain events
func (r *PostgresDateiRepository) updateProjection(
	ctx context.Context,
	q *db.Queries,
	event events.DomainEvent,
) error {
	switch e := event.(type) {
	case events.DateiCreatedEvent:
		return projections.UpdateProjectionForDateiCreated(ctx, q, &e)

	case events.DateiRenamedEvent:
		return projections.UpdateProjectionForDateiRenamed(ctx, q, &e)

	case events.DateiVersionUploadedEvent:
		return projections.UpdateProjectionForDateiVersionUploaded(ctx, q, &e)

	case events.DateiMovedEvent:
		return projections.UpdateProjectionForDateiMoved(ctx, q, &e)

	case events.DateiTrashedEvent:
		return projections.UpdateProjectionForDateiTrashed(ctx, q, &e)

	case events.DateiRestoredEvent:
		return projections.UpdateProjectionForDateiRestored(ctx, q, &e)

	case events.DateiLinkedEvent:
		return projections.UpdateProjectionForDateiLinked(ctx, q, &e)

	case events.DateiUnlinkedEvent:
		return projections.UpdateProjectionForDateiUnlinked(ctx, q, &e)

	case events.DateiPermissionGrantedEvent:
		return projections.UpdateProjectionForDateiPermissionGranted(ctx, q, &e)

	case events.DateiPermissionRevokedEvent:
		return projections.UpdateProjectionForDateiPermissionRevoked(ctx, q, &e)

	default:
		return errors.New("unknown event type")
	}
}

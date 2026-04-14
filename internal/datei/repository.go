package datei

import (
	"context"
	"errors"
	"fmt"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for datei persistence
type Repository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*Aggregate, error)
	Save(ctx context.Context, aggregate *Aggregate) error
}

type postgresRepository struct {
	db         *pgxpool.Pool
	eventStore events.EventStore
}

// NewRepository creates a new datei repository
func NewRepository(db *pgxpool.Pool, eventStore events.EventStore) Repository {
	return &postgresRepository{
		db:         db,
		eventStore: eventStore,
	}
}

func (r *postgresRepository) LoadByID(ctx context.Context, id uuid.UUID) (*Aggregate, error) {
	eventList, err := r.eventStore.GetEvents(ctx, id, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	if len(eventList) == 0 {
		return nil, fmt.Errorf("datei not found: %w", dateierrors.ErrNotFound)
	}

	agg := &Aggregate{}
	if err := agg.ReplayEvents(eventList); err != nil {
		return nil, fmt.Errorf("failed to replay events: %w", err)
	}

	agg.version = len(eventList)
	agg.uncommittedEvents = []events.DomainEvent{}

	return agg, nil
}

func (r *postgresRepository) Save(ctx context.Context, agg *Aggregate) (returnErr error) {
	uncommittedEvents := agg.GetUncommittedEvents()
	if len(uncommittedEvents) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			returnErr = errors.Join(returnErr, err)
		}
	}()

	if err := r.eventStore.AppendToStream(
		ctx, tx, agg.ID, uncommittedEvents, agg.version-len(uncommittedEvents),
	); err != nil {
		return fmt.Errorf("failed to append events: %w", err)
	}

	q := db.New(tx)
	for _, event := range uncommittedEvents {
		if err := updateProjection(ctx, q, event); err != nil {
			return fmt.Errorf("failed to update projection for %s: %w", event.EventType(), err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	agg.MarkEventsAsCommitted()

	return nil
}

func updateProjection(ctx context.Context, q *db.Queries, event events.DomainEvent) error {
	switch e := event.(type) {
	case DateiCreatedEvent:
		return updateProjectionForDateiCreated(ctx, q, &e)
	case DateiRenamedEvent:
		return updateProjectionForDateiRenamed(ctx, q, &e)
	case DateiVersionUploadedEvent:
		return updateProjectionForDateiVersionUploaded(ctx, q, &e)
	case DateiMovedEvent:
		return updateProjectionForDateiMoved(ctx, q, &e)
	case DateiTrashedEvent:
		return updateProjectionForDateiTrashed(ctx, q, &e)
	case DateiRestoredEvent:
		return updateProjectionForDateiRestored(ctx, q, &e)
	case DateiLinkedEvent:
		return updateProjectionForDateiLinked(ctx, q, &e)
	case DateiUnlinkedEvent:
		return updateProjectionForDateiUnlinked(ctx, q, &e)
	case DateiPermissionGrantedEvent:
		return updateProjectionForDateiPermissionGranted(ctx, q, &e)
	case DateiPermissionRevokedEvent:
		return updateProjectionForDateiPermissionRevoked(ctx, q, &e)
	default:
		return errors.New("unknown event type")
	}
}

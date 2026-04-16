package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectionFunc updates a read-model projection for a single event.
type ProjectionFunc func(ctx context.Context, q *db.Queries, event DomainEvent) error

// GenericRepository handles the common event-sourcing persistence logic.
// Domain repositories compose this and add typed wrappers.
type GenericRepository struct {
	pool             *pgxpool.Pool
	eventStore       EventStore
	entityName       string
	updateProjection ProjectionFunc
}

// NewGenericRepository creates a GenericRepository.
func NewGenericRepository(
	pool *pgxpool.Pool,
	es EventStore,
	entityName string,
	up ProjectionFunc,
) GenericRepository {
	return GenericRepository{
		pool:             pool,
		eventStore:       es,
		entityName:       entityName,
		updateProjection: up,
	}
}

// LoadByID fetches all events for a stream and replays them onto the aggregate.
func (r *GenericRepository) LoadByID(ctx context.Context, id uuid.UUID, agg AggregateRoot) error {
	eventList, err := r.eventStore.GetEvents(ctx, id, 0)
	if err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	if len(eventList) == 0 {
		return fmt.Errorf("%s not found: %w", r.entityName, dateierrors.ErrNotFound)
	}

	agg.Replay(eventList)

	return nil
}

// Save persists uncommitted events and updates projections in a single transaction.
func (r *GenericRepository) Save(ctx context.Context, agg AggregateRoot) (returnErr error) {
	uncommittedEvents := agg.GetUncommittedEvents()
	if len(uncommittedEvents) == 0 {
		return nil
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			returnErr = errors.Join(returnErr, err)
		}
	}()

	if err := r.eventStore.AppendToStream(
		ctx, tx, agg.AggregateID(), uncommittedEvents, agg.Version()-len(uncommittedEvents),
	); err != nil {
		return fmt.Errorf("failed to append events: %w", err)
	}

	q := db.New(tx)
	for _, event := range uncommittedEvents {
		if err := r.updateProjection(ctx, q, event); err != nil {
			return fmt.Errorf("failed to update projection for %s: %w", event.EventType(), err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	agg.MarkEventsAsCommitted()

	return nil
}

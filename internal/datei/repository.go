package datei

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for datei persistence
type Repository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*Aggregate, error)
	Save(ctx context.Context, aggregate *Aggregate) error
}

type postgresRepository struct {
	base events.GenericRepository
}

// NewRepository creates a new datei repository
func NewRepository(pool *pgxpool.Pool, eventStore events.EventStore) Repository {
	return &postgresRepository{
		base: events.NewGenericRepository(pool, eventStore, "datei", updateProjection),
	}
}

func (r *postgresRepository) LoadByID(ctx context.Context, id uuid.UUID) (*Aggregate, error) {
	agg := &Aggregate{}
	if err := r.base.LoadByID(ctx, id, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

func (r *postgresRepository) Save(ctx context.Context, agg *Aggregate) error {
	return r.base.Save(ctx, agg)
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
		return fmt.Errorf("unknown datei event type: %s", event.EventType())
	}
}

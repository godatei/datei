package link

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for link persistence.
type Repository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*Aggregate, error)
	Save(ctx context.Context, aggregate *Aggregate) error
}

type postgresRepository struct {
	base events.GenericRepository
}

// NewRepository creates a new link repository.
func NewRepository(pool *pgxpool.Pool, eventStore events.EventStore) Repository {
	return &postgresRepository{
		base: events.NewGenericRepository(pool, eventStore, "link", updateProjection),
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
	case LinkCreatedEvent:
		return updateProjectionForLinkCreated(ctx, q, &e)
	case LinkUpdatedEvent:
		return updateProjectionForLinkUpdated(ctx, q, &e)
	case LinkKeyRotatedEvent:
		return updateProjectionForLinkKeyRotated(ctx, q, &e)
	case LinkDateiAddedEvent:
		return updateProjectionForLinkDateiAdded(ctx, q, &e)
	case LinkDateiRemovedEvent:
		return updateProjectionForLinkDateiRemoved(ctx, q, &e)
	case LinkRevokedEvent:
		return updateProjectionForLinkRevoked(ctx, q, &e)
	case LinkOpenedEvent:
		return updateProjectionForLinkOpened(ctx, q, &e)
	default:
		return fmt.Errorf("unknown link event type: %s", event.EventType())
	}
}

package file

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for file persistence
type Repository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*Aggregate, error)
	Save(ctx context.Context, aggregate *Aggregate) error
}

type postgresRepository struct {
	base events.GenericRepository
}

// NewRepository creates a new file repository
func NewRepository(pool *pgxpool.Pool, eventStore events.EventStore) Repository {
	return &postgresRepository{
		base: events.NewGenericRepository(pool, eventStore, "file", updateProjection),
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
	case FileCreatedEvent:
		return updateProjectionForFileCreated(ctx, q, &e)
	case FileRenamedEvent:
		return updateProjectionForFileRenamed(ctx, q, &e)
	case FileVersionUploadedEvent:
		return updateProjectionForFileVersionUploaded(ctx, q, &e)
	case FileMovedEvent:
		return updateProjectionForFileMoved(ctx, q, &e)
	case FileTrashedEvent:
		return updateProjectionForFileTrashed(ctx, q, &e)
	case FileRestoredEvent:
		return updateProjectionForFileRestored(ctx, q, &e)
	case FileLinkedEvent:
		return updateProjectionForFileLinked(ctx, q, &e)
	case FileUnlinkedEvent:
		return updateProjectionForFileUnlinked(ctx, q, &e)
	case FilePermissionGrantedEvent:
		return updateProjectionForFilePermissionGranted(ctx, q, &e)
	case FilePermissionRevokedEvent:
		return updateProjectionForFilePermissionRevoked(ctx, q, &e)
	default:
		return fmt.Errorf("unknown file event type: %s", event.EventType())
	}
}

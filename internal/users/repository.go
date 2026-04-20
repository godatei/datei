package users

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for user persistence
type Repository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*Aggregate, error)
	Save(ctx context.Context, aggregate *Aggregate) error
}

type postgresRepository struct {
	base events.GenericRepository
}

// NewRepository creates a new user repository
func NewRepository(pool *pgxpool.Pool, eventStore events.EventStore) Repository {
	return &postgresRepository{
		base: events.NewGenericRepository(pool, eventStore, "user", updateProjection),
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
	case UserRegisteredEvent:
		return updateProjectionForUserRegistered(ctx, q, &e)
	case UserNameChangedEvent:
		return updateProjectionForUserNameChanged(ctx, q, &e)
	case UserPasswordChangedEvent:
		return updateProjectionForUserPasswordChanged(ctx, q, &e)
	case UserEmailChangedEvent:
		return updateProjectionForUserEmailChanged(ctx, q, &e)
	case UserEmailVerifiedEvent:
		return updateProjectionForUserEmailVerified(ctx, q, &e)
	case UserEmailAddedEvent:
		return updateProjectionForUserEmailAdded(ctx, q, &e)
	case UserEmailRemovedEvent:
		return updateProjectionForUserEmailRemoved(ctx, q, &e)
	case UserEmailSetPrimaryEvent:
		return updateProjectionForUserEmailSetPrimary(ctx, q, &e)
	case UserMFASetupInitiatedEvent:
		return updateProjectionForUserMFASetupInitiated(ctx, q, &e)
	case UserMFAEnabledEvent:
		return updateProjectionForUserMFAEnabled(ctx, q, &e)
	case UserMFADisabledEvent:
		return updateProjectionForUserMFADisabled(ctx, q, &e)
	case UserMFARecoveryCodeUsedEvent:
		return updateProjectionForUserMFARecoveryCodeUsed(ctx, q, &e)
	case UserMFARecoveryCodesRegeneratedEvent:
		return updateProjectionForUserMFARecoveryCodesRegenerated(ctx, q, &e)
	case UserArchivedEvent:
		return updateProjectionForUserArchived(ctx, q, &e)
	case UserLoggedInEvent:
		return updateProjectionForUserLoggedIn(ctx, q, &e)
	default:
		return fmt.Errorf("unknown user event type: %s", event.EventType())
	}
}

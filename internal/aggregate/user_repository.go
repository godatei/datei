//nolint:dupl // mirrors repository.go by design (event-sourcing repository pattern)
package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/godatei/datei/internal/projections"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository defines the interface for user persistence
type UserRepository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*UserAggregate, error)
	Save(ctx context.Context, aggregate *UserAggregate) error
}

// PostgresUserRepository implements UserRepository
type PostgresUserRepository struct {
	db         *pgxpool.Pool
	eventStore events.EventStore
}

// NewPostgresUserRepository creates a new user repository
func NewPostgresUserRepository(db *pgxpool.Pool, eventStore events.EventStore) *PostgresUserRepository {
	return &PostgresUserRepository{
		db:         db,
		eventStore: eventStore,
	}
}

func (r *PostgresUserRepository) LoadByID(ctx context.Context, id uuid.UUID) (*UserAggregate, error) {
	eventList, err := r.eventStore.GetEvents(ctx, id, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	if len(eventList) == 0 {
		return nil, fmt.Errorf("user not found: %w", dateierrors.ErrNotFound)
	}

	agg := &UserAggregate{}
	if err := agg.ReplayEvents(eventList); err != nil {
		return nil, fmt.Errorf("failed to replay events: %w", err)
	}

	agg.version = len(eventList)
	agg.uncommittedEvents = []events.DomainEvent{}

	return agg, nil
}

func (r *PostgresUserRepository) Save(ctx context.Context, agg *UserAggregate) (returnErr error) {
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
		ctx,
		tx,
		agg.ID,
		uncommittedEvents,
		agg.version-len(uncommittedEvents),
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

func (r *PostgresUserRepository) updateProjection(
	ctx context.Context,
	q *db.Queries,
	event events.DomainEvent,
) error {
	switch e := event.(type) {
	case events.UserRegisteredEvent:
		return projections.UpdateProjectionForUserRegistered(ctx, q, &e)
	case events.UserNameChangedEvent:
		return projections.UpdateProjectionForUserNameChanged(ctx, q, &e)
	case events.UserPasswordChangedEvent:
		return projections.UpdateProjectionForUserPasswordChanged(ctx, q, &e)
	case events.UserEmailChangedEvent:
		return projections.UpdateProjectionForUserEmailChanged(ctx, q, &e)
	case events.UserEmailVerifiedEvent:
		return projections.UpdateProjectionForUserEmailVerified(ctx, q, &e)
	case events.UserEmailAddedEvent:
		return projections.UpdateProjectionForUserEmailAdded(ctx, q, &e)
	case events.UserEmailRemovedEvent:
		return projections.UpdateProjectionForUserEmailRemoved(ctx, q, &e)
	case events.UserEmailSetPrimaryEvent:
		return projections.UpdateProjectionForUserEmailSetPrimary(ctx, q, &e)
	case events.UserMFASetupInitiatedEvent:
		return projections.UpdateProjectionForUserMFASetupInitiated(ctx, q, &e)
	case events.UserMFAEnabledEvent:
		return projections.UpdateProjectionForUserMFAEnabled(ctx, q, &e)
	case events.UserMFADisabledEvent:
		return projections.UpdateProjectionForUserMFADisabled(ctx, q, &e)
	case events.UserMFARecoveryCodeUsedEvent:
		return projections.UpdateProjectionForUserMFARecoveryCodeUsed(ctx, q, &e)
	case events.UserMFARecoveryCodesRegeneratedEvent:
		return projections.UpdateProjectionForUserMFARecoveryCodesRegenerated(ctx, q, &e)
	case events.UserArchivedEvent:
		return projections.UpdateProjectionForUserArchived(ctx, q, &e)
	case events.UserLoggedInEvent:
		return projections.UpdateProjectionForUserLoggedIn(ctx, q, &e)
	default:
		return fmt.Errorf("unknown user event type: %s", event.EventType())
	}
}

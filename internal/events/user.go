package events

import (
	"context"
	"time"

	"github.com/godatei/datei/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//nolint:dupl // each domain wires its own sqlc queries into the generic store
func NewUserEventStore(pool *pgxpool.Pool) *PostgresEventStore {
	return newStore(pool, storeQueries{
		getVersion: func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error) {
			return q.GetUserAccountStreamVersion(ctx, id)
		},
		insert: func(ctx context.Context, q *db.Queries, p InsertParams) error {
			return q.InsertUserAccountEvent(ctx, db.InsertUserAccountEventParams{
				StreamID: p.StreamID, StreamVersion: p.StreamVersion,
				EventType: p.EventType, EventData: p.EventData,
			})
		},
		getEvents: func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]EventRow, error) {
			rows, err := q.GetUserAccountEventsByStreamID(ctx, db.GetUserAccountEventsByStreamIDParams{
				StreamID: id, StreamVersion: from,
			})
			if err != nil {
				return nil, err
			}
			out := make([]EventRow, len(rows))
			for i, r := range rows {
				out[i] = EventRow{EventType: r.EventType, EventData: r.EventData}
			}
			return out, nil
		},
	})
}

func init() {
	RegisterEvent("UserRegistered", func() DomainEvent { return &UserRegisteredEvent{} })
	RegisterEvent("UserNameChanged", func() DomainEvent { return &UserNameChangedEvent{} })
	RegisterEvent("UserPasswordChanged", func() DomainEvent { return &UserPasswordChangedEvent{} })
	RegisterEvent("UserEmailChanged", func() DomainEvent { return &UserEmailChangedEvent{} })
	RegisterEvent("UserEmailVerified", func() DomainEvent { return &UserEmailVerifiedEvent{} })
	RegisterEvent("UserEmailAdded", func() DomainEvent { return &UserEmailAddedEvent{} })
	RegisterEvent("UserEmailRemoved", func() DomainEvent { return &UserEmailRemovedEvent{} })
	RegisterEvent("UserEmailSetPrimary", func() DomainEvent { return &UserEmailSetPrimaryEvent{} })
	RegisterEvent("UserMFASetupInitiated", func() DomainEvent { return &UserMFASetupInitiatedEvent{} })
	RegisterEvent("UserMFAEnabled", func() DomainEvent { return &UserMFAEnabledEvent{} })
	RegisterEvent("UserMFADisabled", func() DomainEvent { return &UserMFADisabledEvent{} })
	RegisterEvent("UserMFARecoveryCodeUsed", func() DomainEvent { return &UserMFARecoveryCodeUsedEvent{} })
	RegisterEvent("UserMFARecoveryCodesRegenerated", func() DomainEvent { return &UserMFARecoveryCodesRegeneratedEvent{} })
	RegisterEvent("UserArchived", func() DomainEvent { return &UserArchivedEvent{} })
	RegisterEvent("UserLoggedIn", func() DomainEvent { return &UserLoggedInEvent{} })
}

// HashedRecoveryCode is stored in events for MFA recovery codes.
type HashedRecoveryCode struct {
	ID       uuid.UUID `json:"id"`
	CodeHash []byte    `json:"code_hash"`
	CodeSalt []byte    `json:"code_salt"`
}

type UserRegisteredEvent struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	EmailID      uuid.UUID `json:"email_id"`
	PasswordHash []byte    `json:"password_hash"`
	PasswordSalt []byte    `json:"password_salt"`
	CreatedAt    time.Time `json:"created_at"`
}

func (e UserRegisteredEvent) EventType() string   { return "UserRegistered" }
func (e UserRegisteredEvent) StreamID() uuid.UUID { return e.ID }

type UserNameChangedEvent struct {
	ID        uuid.UUID `json:"id"`
	NewName   string    `json:"new_name"`
	ChangedAt time.Time `json:"changed_at"`
}

func (e UserNameChangedEvent) EventType() string   { return "UserNameChanged" }
func (e UserNameChangedEvent) StreamID() uuid.UUID { return e.ID }

type UserPasswordChangedEvent struct {
	ID           uuid.UUID `json:"id"`
	PasswordHash []byte    `json:"password_hash"`
	PasswordSalt []byte    `json:"password_salt"`
	ChangedAt    time.Time `json:"changed_at"`
}

func (e UserPasswordChangedEvent) EventType() string   { return "UserPasswordChanged" }
func (e UserPasswordChangedEvent) StreamID() uuid.UUID { return e.ID }

type UserEmailChangedEvent struct {
	ID        uuid.UUID `json:"id"`
	OldEmail  string    `json:"old_email"`
	NewEmail  string    `json:"new_email"`
	ChangedAt time.Time `json:"changed_at"`
}

func (e UserEmailChangedEvent) EventType() string   { return "UserEmailChanged" }
func (e UserEmailChangedEvent) StreamID() uuid.UUID { return e.ID }

type UserEmailVerifiedEvent struct {
	ID         uuid.UUID `json:"id"`
	VerifiedAt time.Time `json:"verified_at"`
}

func (e UserEmailVerifiedEvent) EventType() string   { return "UserEmailVerified" }
func (e UserEmailVerifiedEvent) StreamID() uuid.UUID { return e.ID }

type UserEmailAddedEvent struct {
	ID      uuid.UUID `json:"id"`
	EmailID uuid.UUID `json:"email_id"`
	Email   string    `json:"email"`
	AddedAt time.Time `json:"added_at"`
}

func (e UserEmailAddedEvent) EventType() string   { return "UserEmailAdded" }
func (e UserEmailAddedEvent) StreamID() uuid.UUID { return e.ID }

type UserEmailRemovedEvent struct {
	ID        uuid.UUID `json:"id"`
	EmailID   uuid.UUID `json:"email_id"`
	RemovedAt time.Time `json:"removed_at"`
}

func (e UserEmailRemovedEvent) EventType() string   { return "UserEmailRemoved" }
func (e UserEmailRemovedEvent) StreamID() uuid.UUID { return e.ID }

type UserEmailSetPrimaryEvent struct {
	ID                uuid.UUID `json:"id"`
	OldPrimaryEmailID uuid.UUID `json:"old_primary_email_id"`
	NewPrimaryEmailID uuid.UUID `json:"new_primary_email_id"`
	ChangedAt         time.Time `json:"changed_at"`
}

func (e UserEmailSetPrimaryEvent) EventType() string   { return "UserEmailSetPrimary" }
func (e UserEmailSetPrimaryEvent) StreamID() uuid.UUID { return e.ID }

type UserMFASetupInitiatedEvent struct {
	ID          uuid.UUID `json:"id"`
	MFASecret   string    `json:"mfa_secret"`
	InitiatedAt time.Time `json:"initiated_at"`
}

func (e UserMFASetupInitiatedEvent) EventType() string   { return "UserMFASetupInitiated" }
func (e UserMFASetupInitiatedEvent) StreamID() uuid.UUID { return e.ID }

type UserMFAEnabledEvent struct {
	ID            uuid.UUID            `json:"id"`
	RecoveryCodes []HashedRecoveryCode `json:"recovery_codes"`
	EnabledAt     time.Time            `json:"enabled_at"`
}

func (e UserMFAEnabledEvent) EventType() string   { return "UserMFAEnabled" }
func (e UserMFAEnabledEvent) StreamID() uuid.UUID { return e.ID }

type UserMFADisabledEvent struct {
	ID         uuid.UUID `json:"id"`
	DisabledAt time.Time `json:"disabled_at"`
}

func (e UserMFADisabledEvent) EventType() string   { return "UserMFADisabled" }
func (e UserMFADisabledEvent) StreamID() uuid.UUID { return e.ID }

type UserMFARecoveryCodeUsedEvent struct {
	ID             uuid.UUID `json:"id"`
	RecoveryCodeID uuid.UUID `json:"recovery_code_id"`
	UsedAt         time.Time `json:"used_at"`
}

func (e UserMFARecoveryCodeUsedEvent) EventType() string   { return "UserMFARecoveryCodeUsed" }
func (e UserMFARecoveryCodeUsedEvent) StreamID() uuid.UUID { return e.ID }

type UserMFARecoveryCodesRegeneratedEvent struct {
	ID            uuid.UUID            `json:"id"`
	RecoveryCodes []HashedRecoveryCode `json:"recovery_codes"`
	RegeneratedAt time.Time            `json:"regenerated_at"`
}

func (e UserMFARecoveryCodesRegeneratedEvent) EventType() string {
	return "UserMFARecoveryCodesRegenerated"
}
func (e UserMFARecoveryCodesRegeneratedEvent) StreamID() uuid.UUID { return e.ID }

type UserArchivedEvent struct {
	ID         uuid.UUID `json:"id"`
	ArchivedAt time.Time `json:"archived_at"`
}

func (e UserArchivedEvent) EventType() string   { return "UserArchived" }
func (e UserArchivedEvent) StreamID() uuid.UUID { return e.ID }

type UserLoggedInEvent struct {
	ID         uuid.UUID `json:"id"`
	LoggedInAt time.Time `json:"logged_in_at"`
}

func (e UserLoggedInEvent) EventType() string   { return "UserLoggedIn" }
func (e UserLoggedInEvent) StreamID() uuid.UUID { return e.ID }

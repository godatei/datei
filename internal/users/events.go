package users

import (
	"context"
	"time"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserEvent extends DomainEvent with the ability to apply itself to an Aggregate.
type UserEvent interface {
	events.DomainEvent
	ApplyTo(a *Aggregate)
}

// NewEventStore creates an event store for the user_account_event table.
//
//nolint:dupl // each domain wires its own sqlc queries into the generic store
func NewEventStore(pool *pgxpool.Pool) *events.PostgresEventStore {
	return events.NewStore(pool, events.StoreQueries{
		GetVersion: func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error) {
			return q.GetUserAccountStreamVersion(ctx, id)
		},
		Insert: func(ctx context.Context, q *db.Queries, p events.InsertParams) error {
			return q.InsertUserAccountEvent(ctx, db.InsertUserAccountEventParams{
				StreamID: p.StreamID, StreamVersion: p.StreamVersion,
				EventType: p.EventType, EventData: p.EventData,
			})
		},
		GetEvents: func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]events.EventRow, error) {
			rows, err := q.GetUserAccountEventsByStreamID(ctx, db.GetUserAccountEventsByStreamIDParams{
				StreamID: id, StreamVersion: from,
			})
			if err != nil {
				return nil, err
			}
			out := make([]events.EventRow, len(rows))
			for i, r := range rows {
				out[i] = events.EventRow{EventType: r.EventType, EventData: r.EventData}
			}
			return out, nil
		},
	})
}

func init() {
	events.RegisterEvent("UserRegistered", func() events.DomainEvent { return &UserRegisteredEvent{} })
	events.RegisterEvent("UserNameChanged", func() events.DomainEvent { return &UserNameChangedEvent{} })
	events.RegisterEvent("UserPasswordChanged", func() events.DomainEvent { return &UserPasswordChangedEvent{} })
	events.RegisterEvent("UserEmailChanged", func() events.DomainEvent { return &UserEmailChangedEvent{} })
	events.RegisterEvent("UserEmailVerified", func() events.DomainEvent { return &UserEmailVerifiedEvent{} })
	events.RegisterEvent("UserEmailAdded", func() events.DomainEvent { return &UserEmailAddedEvent{} })
	events.RegisterEvent("UserEmailRemoved", func() events.DomainEvent { return &UserEmailRemovedEvent{} })
	events.RegisterEvent("UserEmailSetPrimary", func() events.DomainEvent { return &UserEmailSetPrimaryEvent{} })
	events.RegisterEvent("UserMFASetupInitiated", func() events.DomainEvent { return &UserMFASetupInitiatedEvent{} })
	events.RegisterEvent("UserMFAEnabled", func() events.DomainEvent { return &UserMFAEnabledEvent{} })
	events.RegisterEvent("UserMFADisabled", func() events.DomainEvent { return &UserMFADisabledEvent{} })
	events.RegisterEvent("UserMFARecoveryCodeUsed", func() events.DomainEvent { return &UserMFARecoveryCodeUsedEvent{} })
	events.RegisterEvent("UserMFARecoveryCodesRegenerated",
		func() events.DomainEvent { return &UserMFARecoveryCodesRegeneratedEvent{} })
	events.RegisterEvent("UserArchived", func() events.DomainEvent { return &UserArchivedEvent{} })
	events.RegisterEvent("UserLoggedIn", func() events.DomainEvent { return &UserLoggedInEvent{} })
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
func (e UserRegisteredEvent) ApplyTo(a *Aggregate) {
	a.ID = e.ID
	a.Name = e.Name
	a.Email = e.Email
	a.EmailID = e.EmailID
	a.PasswordHash = e.PasswordHash
	a.PasswordSalt = e.PasswordSalt
	a.CreatedAt = e.CreatedAt
	a.UpdatedAt = e.CreatedAt
}

type UserNameChangedEvent struct {
	ID        uuid.UUID `json:"id"`
	NewName   string    `json:"new_name"`
	ChangedAt time.Time `json:"changed_at"`
}

func (e UserNameChangedEvent) EventType() string   { return "UserNameChanged" }
func (e UserNameChangedEvent) StreamID() uuid.UUID { return e.ID }
func (e UserNameChangedEvent) ApplyTo(a *Aggregate) {
	a.Name = e.NewName
	a.UpdatedAt = e.ChangedAt
}

type UserPasswordChangedEvent struct {
	ID           uuid.UUID `json:"id"`
	PasswordHash []byte    `json:"password_hash"`
	PasswordSalt []byte    `json:"password_salt"`
	ChangedAt    time.Time `json:"changed_at"`
}

func (e UserPasswordChangedEvent) EventType() string   { return "UserPasswordChanged" }
func (e UserPasswordChangedEvent) StreamID() uuid.UUID { return e.ID }
func (e UserPasswordChangedEvent) ApplyTo(a *Aggregate) {
	a.PasswordHash = e.PasswordHash
	a.PasswordSalt = e.PasswordSalt
	a.UpdatedAt = e.ChangedAt
}

type UserEmailChangedEvent struct {
	ID        uuid.UUID `json:"id"`
	OldEmail  string    `json:"old_email"`
	NewEmail  string    `json:"new_email"`
	ChangedAt time.Time `json:"changed_at"`
}

func (e UserEmailChangedEvent) EventType() string   { return "UserEmailChanged" }
func (e UserEmailChangedEvent) StreamID() uuid.UUID { return e.ID }
func (e UserEmailChangedEvent) ApplyTo(a *Aggregate) {
	a.Email = e.NewEmail
	a.EmailVerified = false
	a.UpdatedAt = e.ChangedAt
}

type UserEmailVerifiedEvent struct {
	ID         uuid.UUID `json:"id"`
	VerifiedAt time.Time `json:"verified_at"`
}

func (e UserEmailVerifiedEvent) EventType() string   { return "UserEmailVerified" }
func (e UserEmailVerifiedEvent) StreamID() uuid.UUID { return e.ID }
func (e UserEmailVerifiedEvent) ApplyTo(a *Aggregate) {
	a.EmailVerified = true
	a.UpdatedAt = e.VerifiedAt
}

type UserEmailAddedEvent struct {
	ID      uuid.UUID `json:"id"`
	EmailID uuid.UUID `json:"email_id"`
	Email   string    `json:"email"`
	AddedAt time.Time `json:"added_at"`
}

func (e UserEmailAddedEvent) EventType() string    { return "UserEmailAdded" }
func (e UserEmailAddedEvent) StreamID() uuid.UUID  { return e.ID }
func (e UserEmailAddedEvent) ApplyTo(a *Aggregate) { a.UpdatedAt = e.AddedAt }

type UserEmailRemovedEvent struct {
	ID        uuid.UUID `json:"id"`
	EmailID   uuid.UUID `json:"email_id"`
	RemovedAt time.Time `json:"removed_at"`
}

func (e UserEmailRemovedEvent) EventType() string    { return "UserEmailRemoved" }
func (e UserEmailRemovedEvent) StreamID() uuid.UUID  { return e.ID }
func (e UserEmailRemovedEvent) ApplyTo(a *Aggregate) { a.UpdatedAt = e.RemovedAt }

type UserEmailSetPrimaryEvent struct {
	ID                uuid.UUID `json:"id"`
	OldPrimaryEmailID uuid.UUID `json:"old_primary_email_id"`
	NewPrimaryEmailID uuid.UUID `json:"new_primary_email_id"`
	ChangedAt         time.Time `json:"changed_at"`
}

func (e UserEmailSetPrimaryEvent) EventType() string   { return "UserEmailSetPrimary" }
func (e UserEmailSetPrimaryEvent) StreamID() uuid.UUID { return e.ID }
func (e UserEmailSetPrimaryEvent) ApplyTo(a *Aggregate) {
	a.EmailID = e.NewPrimaryEmailID
	a.UpdatedAt = e.ChangedAt
}

type UserMFASetupInitiatedEvent struct {
	ID          uuid.UUID `json:"id"`
	MFASecret   string    `json:"mfa_secret"`
	InitiatedAt time.Time `json:"initiated_at"`
}

func (e UserMFASetupInitiatedEvent) EventType() string   { return "UserMFASetupInitiated" }
func (e UserMFASetupInitiatedEvent) StreamID() uuid.UUID { return e.ID }
func (e UserMFASetupInitiatedEvent) ApplyTo(a *Aggregate) {
	a.MFASecret = &e.MFASecret
	a.UpdatedAt = e.InitiatedAt
}

type UserMFAEnabledEvent struct {
	ID            uuid.UUID            `json:"id"`
	RecoveryCodes []HashedRecoveryCode `json:"recovery_codes"`
	EnabledAt     time.Time            `json:"enabled_at"`
}

func (e UserMFAEnabledEvent) EventType() string   { return "UserMFAEnabled" }
func (e UserMFAEnabledEvent) StreamID() uuid.UUID { return e.ID }
func (e UserMFAEnabledEvent) ApplyTo(a *Aggregate) {
	a.MFAEnabled = true
	a.UpdatedAt = e.EnabledAt
}

type UserMFADisabledEvent struct {
	ID         uuid.UUID `json:"id"`
	DisabledAt time.Time `json:"disabled_at"`
}

func (e UserMFADisabledEvent) EventType() string   { return "UserMFADisabled" }
func (e UserMFADisabledEvent) StreamID() uuid.UUID { return e.ID }
func (e UserMFADisabledEvent) ApplyTo(a *Aggregate) {
	a.MFAEnabled = false
	a.MFASecret = nil
	a.UpdatedAt = e.DisabledAt
}

type UserMFARecoveryCodeUsedEvent struct {
	ID             uuid.UUID `json:"id"`
	RecoveryCodeID uuid.UUID `json:"recovery_code_id"`
	UsedAt         time.Time `json:"used_at"`
}

func (e UserMFARecoveryCodeUsedEvent) EventType() string    { return "UserMFARecoveryCodeUsed" }
func (e UserMFARecoveryCodeUsedEvent) StreamID() uuid.UUID  { return e.ID }
func (e UserMFARecoveryCodeUsedEvent) ApplyTo(a *Aggregate) { a.UpdatedAt = e.UsedAt }

type UserMFARecoveryCodesRegeneratedEvent struct {
	ID            uuid.UUID            `json:"id"`
	RecoveryCodes []HashedRecoveryCode `json:"recovery_codes"`
	RegeneratedAt time.Time            `json:"regenerated_at"`
}

func (e UserMFARecoveryCodesRegeneratedEvent) EventType() string {
	return "UserMFARecoveryCodesRegenerated"
}
func (e UserMFARecoveryCodesRegeneratedEvent) StreamID() uuid.UUID { return e.ID }
func (e UserMFARecoveryCodesRegeneratedEvent) ApplyTo(a *Aggregate) {
	a.UpdatedAt = e.RegeneratedAt
}

type UserArchivedEvent struct {
	ID         uuid.UUID `json:"id"`
	ArchivedAt time.Time `json:"archived_at"`
}

func (e UserArchivedEvent) EventType() string   { return "UserArchived" }
func (e UserArchivedEvent) StreamID() uuid.UUID { return e.ID }
func (e UserArchivedEvent) ApplyTo(a *Aggregate) {
	a.ArchivedAt = &e.ArchivedAt
	a.UpdatedAt = e.ArchivedAt
}

type UserLoggedInEvent struct {
	ID         uuid.UUID `json:"id"`
	LoggedInAt time.Time `json:"logged_in_at"`
}

func (e UserLoggedInEvent) EventType() string    { return "UserLoggedIn" }
func (e UserLoggedInEvent) StreamID() uuid.UUID  { return e.ID }
func (e UserLoggedInEvent) ApplyTo(a *Aggregate) { a.LastLoggedInAt = &e.LoggedInAt }

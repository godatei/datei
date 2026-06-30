package email

import (
	"context"
	"time"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MailAccountEvent extends DomainEvent with the ability to apply itself to a MailAccount.
type MailAccountEvent interface {
	events.DomainEvent
	ApplyTo(a *MailAccount)
}

// MailRuleEvent extends DomainEvent with the ability to apply itself to a MailRule.
type MailRuleEvent interface {
	events.DomainEvent
	ApplyTo(r *MailRule)
}

// NewAccountEventStore creates an event store for the mail_account_event table.
//
//nolint:dupl // each domain wires its own sqlc queries into the generic store
func NewAccountEventStore(pool *pgxpool.Pool) *events.PostgresEventStore {
	return events.NewStore(pool, events.StoreQueries{
		GetVersion: func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error) {
			return q.GetMailAccountStreamVersion(ctx, id)
		},
		Insert: func(ctx context.Context, q *db.Queries, p events.InsertParams) error {
			return q.InsertMailAccountEvent(ctx, db.InsertMailAccountEventParams{
				StreamID: p.StreamID, StreamVersion: p.StreamVersion,
				EventType: p.EventType, EventData: p.EventData,
			})
		},
		GetEvents: func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]events.EventRow, error) {
			rows, err := q.GetMailAccountEventsByStreamID(ctx, db.GetMailAccountEventsByStreamIDParams{
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

// NewRuleEventStore creates an event store for the mail_rule_event table.
//
//nolint:dupl // each domain wires its own sqlc queries into the generic store
func NewRuleEventStore(pool *pgxpool.Pool) *events.PostgresEventStore {
	return events.NewStore(pool, events.StoreQueries{
		GetVersion: func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error) {
			return q.GetMailRuleStreamVersion(ctx, id)
		},
		Insert: func(ctx context.Context, q *db.Queries, p events.InsertParams) error {
			return q.InsertMailRuleEvent(ctx, db.InsertMailRuleEventParams{
				StreamID: p.StreamID, StreamVersion: p.StreamVersion,
				EventType: p.EventType, EventData: p.EventData,
			})
		},
		GetEvents: func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]events.EventRow, error) {
			rows, err := q.GetMailRuleEventsByStreamID(ctx, db.GetMailRuleEventsByStreamIDParams{
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
	events.RegisterEvent("MailAccountCreated", func() events.DomainEvent { return &MailAccountCreatedEvent{} })
	events.RegisterEvent("MailAccountUpdated", func() events.DomainEvent { return &MailAccountUpdatedEvent{} })
	events.RegisterEvent("MailAccountDeleted", func() events.DomainEvent { return &MailAccountDeletedEvent{} })
	events.RegisterEvent("MailRuleCreated", func() events.DomainEvent { return &MailRuleCreatedEvent{} })
	events.RegisterEvent("MailRuleUpdated", func() events.DomainEvent { return &MailRuleUpdatedEvent{} })
	events.RegisterEvent("MailRuleDeleted", func() events.DomainEvent { return &MailRuleDeletedEvent{} })
}

// ============================================================================
// Mail Account Events
// ============================================================================

type MailAccountCreatedEvent struct {
	ID                uuid.UUID `json:"id"`
	OwnerID           uuid.UUID `json:"owner_id"`
	Name              string    `json:"name"`
	ImapHost          string    `json:"imap_host"`
	ImapPort          int       `json:"imap_port"`
	Username          string    `json:"username"`
	PasswordEncrypted []byte    `json:"password_encrypted"`
	Security          Security  `json:"security"`
	CreatedAt         time.Time `json:"created_at"`
}

func (e MailAccountCreatedEvent) EventType() string   { return "MailAccountCreated" }
func (e MailAccountCreatedEvent) StreamID() uuid.UUID { return e.ID }
func (e MailAccountCreatedEvent) ApplyTo(a *MailAccount) {
	a.ID = e.ID
	a.OwnerID = e.OwnerID
	a.Name = e.Name
	a.ImapHost = e.ImapHost
	a.ImapPort = e.ImapPort
	a.Username = e.Username
	a.PasswordEncrypted = e.PasswordEncrypted
	a.Security = e.Security
	a.CreatedAt = e.CreatedAt
	a.UpdatedAt = e.CreatedAt
}

type MailAccountUpdatedEvent struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	ImapHost          string    `json:"imap_host"`
	ImapPort          int       `json:"imap_port"`
	Username          string    `json:"username"`
	PasswordEncrypted []byte    `json:"password_encrypted"`
	Security          Security  `json:"security"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (e MailAccountUpdatedEvent) EventType() string   { return "MailAccountUpdated" }
func (e MailAccountUpdatedEvent) StreamID() uuid.UUID { return e.ID }
func (e MailAccountUpdatedEvent) ApplyTo(a *MailAccount) {
	a.Name = e.Name
	a.ImapHost = e.ImapHost
	a.ImapPort = e.ImapPort
	a.Username = e.Username
	a.PasswordEncrypted = e.PasswordEncrypted
	a.Security = e.Security
	a.UpdatedAt = e.UpdatedAt
}

type MailAccountDeletedEvent struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func (e MailAccountDeletedEvent) EventType() string   { return "MailAccountDeleted" }
func (e MailAccountDeletedEvent) StreamID() uuid.UUID { return e.ID }
func (e MailAccountDeletedEvent) ApplyTo(a *MailAccount) {
	a.DeletedAt = &e.DeletedAt
}

// ============================================================================
// Mail Rule Events
// ============================================================================

type MailRuleCreatedEvent struct {
	ID                uuid.UUID  `json:"id"`
	AccountID         uuid.UUID  `json:"account_id"`
	Name              string     `json:"name"`
	Order             int        `json:"order"`
	Enabled           bool       `json:"enabled"`
	Folder            string     `json:"folder"`
	FilterFrom        *string    `json:"filter_from,omitempty"`
	FilterSubject     *string    `json:"filter_subject,omitempty"`
	MaxAgeDays        int        `json:"max_age_days"`
	AttachmentPattern *string    `json:"attachment_pattern,omitempty"`
	Action            *Action    `json:"action,omitempty"`
	TargetDirectoryID *uuid.UUID `json:"target_directory_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (e MailRuleCreatedEvent) EventType() string   { return "MailRuleCreated" }
func (e MailRuleCreatedEvent) StreamID() uuid.UUID { return e.ID }
func (e MailRuleCreatedEvent) ApplyTo(r *MailRule) {
	r.ID = e.ID
	r.AccountID = e.AccountID
	r.Name = e.Name
	r.Order = e.Order
	r.Enabled = e.Enabled
	r.Folder = e.Folder
	r.FilterFrom = e.FilterFrom
	r.FilterSubject = e.FilterSubject
	r.MaxAgeDays = e.MaxAgeDays
	r.AttachmentPattern = e.AttachmentPattern
	r.Action = e.Action
	r.TargetDirectoryID = e.TargetDirectoryID
	r.CreatedAt = e.CreatedAt
	r.UpdatedAt = e.CreatedAt
}

type MailRuleUpdatedEvent struct {
	ID                uuid.UUID  `json:"id"`
	AccountID         uuid.UUID  `json:"account_id,omitempty"`
	Name              string     `json:"name"`
	Order             int        `json:"order"`
	Enabled           bool       `json:"enabled"`
	Folder            string     `json:"folder"`
	FilterFrom        *string    `json:"filter_from,omitempty"`
	FilterSubject     *string    `json:"filter_subject,omitempty"`
	MaxAgeDays        int        `json:"max_age_days"`
	AttachmentPattern *string    `json:"attachment_pattern,omitempty"`
	Action            *Action    `json:"action,omitempty"`
	TargetDirectoryID *uuid.UUID `json:"target_directory_id,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (e MailRuleUpdatedEvent) EventType() string   { return "MailRuleUpdated" }
func (e MailRuleUpdatedEvent) StreamID() uuid.UUID { return e.ID }
func (e MailRuleUpdatedEvent) ApplyTo(r *MailRule) {
	// AccountID was added later; events recorded before it carry uuid.Nil, so
	// only reassign when a concrete account is present to preserve replay state.
	if e.AccountID != uuid.Nil {
		r.AccountID = e.AccountID
	}
	r.Name = e.Name
	r.Order = e.Order
	r.Enabled = e.Enabled
	r.Folder = e.Folder
	r.FilterFrom = e.FilterFrom
	r.FilterSubject = e.FilterSubject
	r.MaxAgeDays = e.MaxAgeDays
	r.AttachmentPattern = e.AttachmentPattern
	r.Action = e.Action
	r.TargetDirectoryID = e.TargetDirectoryID
	r.UpdatedAt = e.UpdatedAt
}

type MailRuleDeletedEvent struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func (e MailRuleDeletedEvent) EventType() string   { return "MailRuleDeleted" }
func (e MailRuleDeletedEvent) StreamID() uuid.UUID { return e.ID }
func (e MailRuleDeletedEvent) ApplyTo(r *MailRule) {
	r.DeletedAt = &e.DeletedAt
}

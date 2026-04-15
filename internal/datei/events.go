package datei

import (
	"context"
	"time"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event extends DomainEvent with the ability to apply itself to an Aggregate.
type Event interface {
	events.DomainEvent
	ApplyTo(a *Aggregate)
}

// NewEventStore creates an event store for the datei_event table.
//
//nolint:dupl // each domain wires its own sqlc queries into the generic store
func NewEventStore(pool *pgxpool.Pool) *events.PostgresEventStore {
	return events.NewStore(pool, events.StoreQueries{
		GetVersion: func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error) {
			return q.GetStreamVersion(ctx, id)
		},
		Insert: func(ctx context.Context, q *db.Queries, p events.InsertParams) error {
			return q.InsertDateiEvent(ctx, db.InsertDateiEventParams{
				StreamID: p.StreamID, StreamVersion: p.StreamVersion,
				EventType: p.EventType, EventData: p.EventData,
			})
		},
		GetEvents: func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]events.EventRow, error) {
			rows, err := q.GetDateiEventsByStreamID(ctx, db.GetDateiEventsByStreamIDParams{
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
	events.RegisterEvent("DateiCreated", func() events.DomainEvent { return &DateiCreatedEvent{} })
	events.RegisterEvent("DateiRenamed", func() events.DomainEvent { return &DateiRenamedEvent{} })
	events.RegisterEvent("DateiVersionUploaded", func() events.DomainEvent { return &DateiVersionUploadedEvent{} })
	events.RegisterEvent("DateiMoved", func() events.DomainEvent { return &DateiMovedEvent{} })
	events.RegisterEvent("DateiTrashed", func() events.DomainEvent { return &DateiTrashedEvent{} })
	events.RegisterEvent("DateiRestored", func() events.DomainEvent { return &DateiRestoredEvent{} })
	events.RegisterEvent("DateiLinked", func() events.DomainEvent { return &DateiLinkedEvent{} })
	events.RegisterEvent("DateiUnlinked", func() events.DomainEvent { return &DateiUnlinkedEvent{} })
	events.RegisterEvent("DateiPermissionGranted", func() events.DomainEvent { return &DateiPermissionGrantedEvent{} })
	events.RegisterEvent("DateiPermissionRevoked", func() events.DomainEvent { return &DateiPermissionRevokedEvent{} })
}

// ============================================================================
// Datei Events
// ============================================================================

type DateiCreatedEvent struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	IsDirectory bool       `json:"is_directory"`
	Name        string     `json:"name"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (e DateiCreatedEvent) EventType() string   { return "DateiCreated" }
func (e DateiCreatedEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiCreatedEvent) ApplyTo(a *Aggregate) {
	a.ID = e.ID
	a.ParentID = e.ParentID
	a.IsDirectory = e.IsDirectory
	a.Name = e.Name
	a.CreatedBy = e.CreatedBy
	a.CreatedAt = e.CreatedAt
	a.UpdatedAt = e.CreatedAt
	a.UpdatedBy = e.CreatedBy
}

type DateiRenamedEvent struct {
	ID        uuid.UUID `json:"id"`
	OldName   string    `json:"old_name"`
	NewName   string    `json:"new_name"`
	RenamedBy uuid.UUID `json:"renamed_by"`
	RenamedAt time.Time `json:"renamed_at"`
}

func (e DateiRenamedEvent) EventType() string   { return "DateiRenamed" }
func (e DateiRenamedEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiRenamedEvent) ApplyTo(a *Aggregate) {
	a.Name = e.NewName
	a.UpdatedAt = e.RenamedAt
	a.UpdatedBy = e.RenamedBy
}

type DateiVersionUploadedEvent struct {
	ID         uuid.UUID `json:"id"`
	S3Key      string    `json:"s3_key"`
	FileSize   int64     `json:"file_size"`
	Checksum   string    `json:"checksum"`
	MimeType   string    `json:"mime_type"`
	ContentMD  *string   `json:"content_md,omitempty"`
	UploadedBy uuid.UUID `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func (e DateiVersionUploadedEvent) EventType() string   { return "DateiVersionUploaded" }
func (e DateiVersionUploadedEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiVersionUploadedEvent) ApplyTo(a *Aggregate) {
	a.S3Key = &e.S3Key
	a.Size = &e.FileSize
	a.Checksum = &e.Checksum
	a.MimeType = &e.MimeType
	a.ContentMD = e.ContentMD
	a.UpdatedAt = e.UploadedAt
	a.UpdatedBy = e.UploadedBy
}

type DateiMovedEvent struct {
	ID          uuid.UUID  `json:"id"`
	OldParentID *uuid.UUID `json:"old_parent_id,omitempty"`
	NewParentID *uuid.UUID `json:"new_parent_id,omitempty"`
	MovedBy     uuid.UUID  `json:"moved_by"`
	MovedAt     time.Time  `json:"moved_at"`
}

func (e DateiMovedEvent) EventType() string   { return "DateiMoved" }
func (e DateiMovedEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiMovedEvent) ApplyTo(a *Aggregate) {
	a.ParentID = e.NewParentID
	a.UpdatedAt = e.MovedAt
	a.UpdatedBy = e.MovedBy
}

type DateiTrashedEvent struct {
	ID        uuid.UUID `json:"id"`
	TrashedBy uuid.UUID `json:"trashed_by"`
	TrashedAt time.Time `json:"trashed_at"`
}

func (e DateiTrashedEvent) EventType() string   { return "DateiTrashed" }
func (e DateiTrashedEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiTrashedEvent) ApplyTo(a *Aggregate) {
	a.TrashedAt = &e.TrashedAt
	a.TrashedBy = &e.TrashedBy
	a.UpdatedAt = e.TrashedAt
	a.UpdatedBy = e.TrashedBy
}

type DateiRestoredEvent struct {
	ID         uuid.UUID `json:"id"`
	RestoredBy uuid.UUID `json:"restored_by"`
	RestoredAt time.Time `json:"restored_at"`
}

func (e DateiRestoredEvent) EventType() string   { return "DateiRestored" }
func (e DateiRestoredEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiRestoredEvent) ApplyTo(a *Aggregate) {
	a.TrashedAt = nil
	a.TrashedBy = nil
	a.UpdatedAt = e.RestoredAt
	a.UpdatedBy = e.RestoredBy
}

type DateiLinkedEvent struct {
	ID            uuid.UUID `json:"id"`
	LinkedDateiID uuid.UUID `json:"linked_datei_id"`
	LinkedBy      uuid.UUID `json:"linked_by"`
	LinkedAt      time.Time `json:"linked_at"`
}

func (e DateiLinkedEvent) EventType() string   { return "DateiLinked" }
func (e DateiLinkedEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiLinkedEvent) ApplyTo(a *Aggregate) {
	a.LinkedDateiID = &e.LinkedDateiID
	a.UpdatedAt = e.LinkedAt
	a.UpdatedBy = e.LinkedBy
}

type DateiUnlinkedEvent struct {
	ID         uuid.UUID `json:"id"`
	UnlinkedBy uuid.UUID `json:"unlinked_by"`
	UnlinkedAt time.Time `json:"unlinked_at"`
}

func (e DateiUnlinkedEvent) EventType() string   { return "DateiUnlinked" }
func (e DateiUnlinkedEvent) StreamID() uuid.UUID { return e.ID }
func (e DateiUnlinkedEvent) ApplyTo(a *Aggregate) {
	a.LinkedDateiID = nil
	a.UpdatedAt = e.UnlinkedAt
	a.UpdatedBy = e.UnlinkedBy
}

// ============================================================================
// Permission Events
// ============================================================================

type DateiPermissionGrantedEvent struct {
	ID             uuid.UUID  `json:"id"`
	DateiID        uuid.UUID  `json:"datei_id"`
	UserAccountID  *uuid.UUID `json:"user_account_id,omitempty"`
	UserGroupID    *uuid.UUID `json:"user_group_id,omitempty"`
	PermissionType string     `json:"permission_type"`
	GrantedBy      uuid.UUID  `json:"granted_by"`
	GrantedAt      time.Time  `json:"granted_at"`
}

func (e DateiPermissionGrantedEvent) EventType() string    { return "DateiPermissionGranted" }
func (e DateiPermissionGrantedEvent) StreamID() uuid.UUID  { return e.DateiID }
func (e DateiPermissionGrantedEvent) ApplyTo(_ *Aggregate) {}

type DateiPermissionRevokedEvent struct {
	ID            uuid.UUID  `json:"id"`
	DateiID       uuid.UUID  `json:"datei_id"`
	UserAccountID *uuid.UUID `json:"user_account_id,omitempty"`
	UserGroupID   *uuid.UUID `json:"user_group_id,omitempty"`
	RevokedBy     uuid.UUID  `json:"revoked_by"`
	RevokedAt     time.Time  `json:"revoked_at"`
}

func (e DateiPermissionRevokedEvent) EventType() string    { return "DateiPermissionRevoked" }
func (e DateiPermissionRevokedEvent) StreamID() uuid.UUID  { return e.DateiID }
func (e DateiPermissionRevokedEvent) ApplyTo(_ *Aggregate) {}

package file

import (
	"context"
	"time"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FileEvent extends DomainEvent with the ability to apply itself to an Aggregate.
type FileEvent interface {
	events.DomainEvent
	ApplyTo(a *Aggregate)
}

// NewEventStore creates an event store for the file_event table.
//
//nolint:dupl // each domain wires its own sqlc queries into the generic store
func NewEventStore(pool *pgxpool.Pool) *events.PostgresEventStore {
	return events.NewStore(pool, events.StoreQueries{
		GetVersion: func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error) {
			return q.GetStreamVersion(ctx, id)
		},
		Insert: func(ctx context.Context, q *db.Queries, p events.InsertParams) error {
			return q.InsertFileEvent(ctx, db.InsertFileEventParams{
				StreamID: p.StreamID, StreamVersion: p.StreamVersion,
				EventType: p.EventType, EventData: p.EventData,
			})
		},
		GetEvents: func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]events.EventRow, error) {
			rows, err := q.GetFileEventsByStreamID(ctx, db.GetFileEventsByStreamIDParams{
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
	events.RegisterEvent("FileCreated", func() events.DomainEvent { return &FileCreatedEvent{} })
	events.RegisterEvent("FileRenamed", func() events.DomainEvent { return &FileRenamedEvent{} })
	events.RegisterEvent("FileVersionUploaded", func() events.DomainEvent { return &FileVersionUploadedEvent{} })
	events.RegisterEvent("FileMoved", func() events.DomainEvent { return &FileMovedEvent{} })
	events.RegisterEvent("FileTrashed", func() events.DomainEvent { return &FileTrashedEvent{} })
	events.RegisterEvent("FileRestored", func() events.DomainEvent { return &FileRestoredEvent{} })
	events.RegisterEvent("FileLinked", func() events.DomainEvent { return &FileLinkedEvent{} })
	events.RegisterEvent("FileUnlinked", func() events.DomainEvent { return &FileUnlinkedEvent{} })
	events.RegisterEvent("FilePermissionGranted", func() events.DomainEvent { return &FilePermissionGrantedEvent{} })
	events.RegisterEvent("FilePermissionRevoked", func() events.DomainEvent { return &FilePermissionRevokedEvent{} })
}

// ============================================================================
// File Events
// ============================================================================

type FileCreatedEvent struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	IsDirectory bool       `json:"is_directory"`
	Name        string     `json:"name"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (e FileCreatedEvent) EventType() string   { return "FileCreated" }
func (e FileCreatedEvent) StreamID() uuid.UUID { return e.ID }
func (e FileCreatedEvent) ApplyTo(a *Aggregate) {
	a.ID = e.ID
	a.ParentID = e.ParentID
	a.IsDirectory = e.IsDirectory
	a.Name = e.Name
	a.CreatedBy = e.CreatedBy
	a.CreatedAt = e.CreatedAt
	a.UpdatedAt = e.CreatedAt
	a.UpdatedBy = e.CreatedBy
}

type FileRenamedEvent struct {
	ID        uuid.UUID `json:"id"`
	OldName   string    `json:"old_name"`
	NewName   string    `json:"new_name"`
	RenamedBy uuid.UUID `json:"renamed_by"`
	RenamedAt time.Time `json:"renamed_at"`
}

func (e FileRenamedEvent) EventType() string   { return "FileRenamed" }
func (e FileRenamedEvent) StreamID() uuid.UUID { return e.ID }
func (e FileRenamedEvent) ApplyTo(a *Aggregate) {
	a.Name = e.NewName
	a.UpdatedAt = e.RenamedAt
	a.UpdatedBy = e.RenamedBy
}

type FileVersionUploadedEvent struct {
	ID         uuid.UUID `json:"id"`
	S3Key      string    `json:"s3_key"`
	FileSize   int64     `json:"file_size"`
	Checksum   string    `json:"checksum"`
	MimeType   string    `json:"mime_type"`
	ContentMD  *string   `json:"content_md,omitempty"`
	UploadedBy uuid.UUID `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func (e FileVersionUploadedEvent) EventType() string   { return "FileVersionUploaded" }
func (e FileVersionUploadedEvent) StreamID() uuid.UUID { return e.ID }
func (e FileVersionUploadedEvent) ApplyTo(a *Aggregate) {
	a.S3Key = &e.S3Key
	a.Size = &e.FileSize
	a.Checksum = &e.Checksum
	a.MimeType = &e.MimeType
	a.ContentMD = e.ContentMD
	a.UpdatedAt = e.UploadedAt
	a.UpdatedBy = e.UploadedBy
}

type FileMovedEvent struct {
	ID          uuid.UUID  `json:"id"`
	OldParentID *uuid.UUID `json:"old_parent_id,omitempty"`
	NewParentID *uuid.UUID `json:"new_parent_id,omitempty"`
	MovedBy     uuid.UUID  `json:"moved_by"`
	MovedAt     time.Time  `json:"moved_at"`
}

func (e FileMovedEvent) EventType() string   { return "FileMoved" }
func (e FileMovedEvent) StreamID() uuid.UUID { return e.ID }
func (e FileMovedEvent) ApplyTo(a *Aggregate) {
	a.ParentID = e.NewParentID
	a.UpdatedAt = e.MovedAt
	a.UpdatedBy = e.MovedBy
}

type FileTrashedEvent struct {
	ID        uuid.UUID `json:"id"`
	TrashedBy uuid.UUID `json:"trashed_by"`
	TrashedAt time.Time `json:"trashed_at"`
}

func (e FileTrashedEvent) EventType() string   { return "FileTrashed" }
func (e FileTrashedEvent) StreamID() uuid.UUID { return e.ID }
func (e FileTrashedEvent) ApplyTo(a *Aggregate) {
	a.TrashedAt = &e.TrashedAt
	a.TrashedBy = &e.TrashedBy
	a.UpdatedAt = e.TrashedAt
	a.UpdatedBy = e.TrashedBy
}

type FileRestoredEvent struct {
	ID         uuid.UUID `json:"id"`
	RestoredBy uuid.UUID `json:"restored_by"`
	RestoredAt time.Time `json:"restored_at"`
}

func (e FileRestoredEvent) EventType() string   { return "FileRestored" }
func (e FileRestoredEvent) StreamID() uuid.UUID { return e.ID }
func (e FileRestoredEvent) ApplyTo(a *Aggregate) {
	a.TrashedAt = nil
	a.TrashedBy = nil
	a.UpdatedAt = e.RestoredAt
	a.UpdatedBy = e.RestoredBy
}

type FileLinkedEvent struct {
	ID           uuid.UUID `json:"id"`
	LinkedFileID uuid.UUID `json:"linked_file_id"`
	LinkedBy     uuid.UUID `json:"linked_by"`
	LinkedAt     time.Time `json:"linked_at"`
}

func (e FileLinkedEvent) EventType() string   { return "FileLinked" }
func (e FileLinkedEvent) StreamID() uuid.UUID { return e.ID }
func (e FileLinkedEvent) ApplyTo(a *Aggregate) {
	a.LinkedFileID = &e.LinkedFileID
	a.UpdatedAt = e.LinkedAt
	a.UpdatedBy = e.LinkedBy
}

type FileUnlinkedEvent struct {
	ID         uuid.UUID `json:"id"`
	UnlinkedBy uuid.UUID `json:"unlinked_by"`
	UnlinkedAt time.Time `json:"unlinked_at"`
}

func (e FileUnlinkedEvent) EventType() string   { return "FileUnlinked" }
func (e FileUnlinkedEvent) StreamID() uuid.UUID { return e.ID }
func (e FileUnlinkedEvent) ApplyTo(a *Aggregate) {
	a.LinkedFileID = nil
	a.UpdatedAt = e.UnlinkedAt
	a.UpdatedBy = e.UnlinkedBy
}

// ============================================================================
// Permission Events
// ============================================================================

type FilePermissionGrantedEvent struct {
	ID             uuid.UUID  `json:"id"`
	FileID         uuid.UUID  `json:"file_id"`
	UserAccountID  *uuid.UUID `json:"user_account_id,omitempty"`
	UserGroupID    *uuid.UUID `json:"user_group_id,omitempty"`
	PermissionType string     `json:"permission_type"`
	GrantedBy      uuid.UUID  `json:"granted_by"`
	GrantedAt      time.Time  `json:"granted_at"`
}

func (e FilePermissionGrantedEvent) EventType() string    { return "FilePermissionGranted" }
func (e FilePermissionGrantedEvent) StreamID() uuid.UUID  { return e.FileID }
func (e FilePermissionGrantedEvent) ApplyTo(_ *Aggregate) {}

type FilePermissionRevokedEvent struct {
	ID            uuid.UUID  `json:"id"`
	FileID        uuid.UUID  `json:"file_id"`
	UserAccountID *uuid.UUID `json:"user_account_id,omitempty"`
	UserGroupID   *uuid.UUID `json:"user_group_id,omitempty"`
	RevokedBy     uuid.UUID  `json:"revoked_by"`
	RevokedAt     time.Time  `json:"revoked_at"`
}

func (e FilePermissionRevokedEvent) EventType() string    { return "FilePermissionRevoked" }
func (e FilePermissionRevokedEvent) StreamID() uuid.UUID  { return e.FileID }
func (e FilePermissionRevokedEvent) ApplyTo(_ *Aggregate) {}

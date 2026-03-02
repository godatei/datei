package aggregate

import (
	"errors"
	"time"

	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
)

// DateiAggregate represents the domain entity for a file or directory
type DateiAggregate struct {
	// Identity
	ID            uuid.UUID
	ParentID      *uuid.UUID
	IsDirectory   bool
	LinkedDateiID *uuid.UUID

	// Current state (derived from events)
	Name      string
	S3Key     *string
	Size      *int64
	Checksum  *string
	MimeType  *string
	ContentMD *string
	CreatedBy uuid.UUID
	CreatedAt time.Time
	TrashedAt *time.Time
	TrashedBy *uuid.UUID
	UpdatedAt time.Time

	// Event tracking
	uncommittedEvents []events.DomainEvent
	version           int
}

// GetUncommittedEvents returns events that haven't been persisted yet
func (a *DateiAggregate) GetUncommittedEvents() []events.DomainEvent {
	return a.uncommittedEvents
}

// MarkEventsAsCommitted clears uncommitted events after successful persistence
func (a *DateiAggregate) MarkEventsAsCommitted() {
	a.uncommittedEvents = []events.DomainEvent{}
}

func (a *DateiAggregate) recordEvent(event events.DomainEvent) {
	a.uncommittedEvents = append(a.uncommittedEvents, event)
	a.version++
	a.ApplyEvent(event)
}

// ============================================================================
// Commands (operations that produce events)
// ============================================================================

// Create initializes a new datei
func (a *DateiAggregate) Create(
	id uuid.UUID,
	parentID *uuid.UUID,
	isDirectory bool,
	name string,
	createdBy uuid.UUID,
	now time.Time,
) error {
	// Validation
	if id == uuid.Nil {
		return errors.New("invalid datei id")
	}
	if createdBy == uuid.Nil {
		return errors.New("created_by cannot be nil")
	}
	if name == "" {
		return errors.New("name cannot be empty")
	}

	event := events.DateiCreatedEvent{
		ID:          id,
		ParentID:    parentID,
		IsDirectory: isDirectory,
		Name:        name,
		CreatedBy:   createdBy,
		CreatedAt:   now,
	}

	a.recordEvent(event)
	return nil
}

// Rename changes the name of the datei
func (a *DateiAggregate) Rename(newName string, renamedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot rename: datei not created")
	}
	if newName == "" {
		return errors.New("name cannot be empty")
	}
	if newName == a.Name {
		return errors.New("new name is same as current name")
	}

	event := events.DateiRenamedEvent{
		ID:        a.ID,
		OldName:   a.Name,
		NewName:   newName,
		RenamedBy: renamedBy,
		RenamedAt: now,
	}

	a.recordEvent(event)
	return nil
}

// UploadVersion creates a new file version
func (a *DateiAggregate) UploadVersion(
	s3Key string,
	fileSize int64,
	checksum string,
	mimeType string,
	contentMD *string,
	uploadedBy uuid.UUID,
	now time.Time,
) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot upload: datei not created")
	}
	if a.IsDirectory {
		return errors.New("cannot upload file to directory")
	}
	if s3Key == "" {
		return errors.New("s3_key cannot be empty")
	}

	event := events.DateiVersionUploadedEvent{
		ID:         a.ID,
		S3Key:      s3Key,
		FileSize:   fileSize,
		Checksum:   checksum,
		MimeType:   mimeType,
		ContentMD:  contentMD,
		UploadedBy: uploadedBy,
		UploadedAt: now,
	}

	a.recordEvent(event)
	return nil
}

// Move changes the parent directory
func (a *DateiAggregate) Move(newParentID *uuid.UUID, movedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot move: datei not created")
	}

	// Prevent circular references (can't be own ancestor)
	if newParentID != nil && *newParentID == a.ID {
		return errors.New("cannot move datei to itself")
	}

	event := events.DateiMovedEvent{
		ID:          a.ID,
		OldParentID: a.ParentID,
		NewParentID: newParentID,
		MovedBy:     movedBy,
		MovedAt:     now,
	}

	a.recordEvent(event)
	return nil
}

// Trash moves to trash bin (soft delete)
func (a *DateiAggregate) Trash(trashedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot trash: datei not created")
	}
	if a.TrashedAt != nil {
		return errors.New("datei already trashed")
	}

	event := events.DateiTrashedEvent{
		ID:        a.ID,
		TrashedBy: trashedBy,
		TrashedAt: now,
	}

	a.recordEvent(event)
	return nil
}

// Restore recovers from trash
func (a *DateiAggregate) Restore(restoredBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot restore: datei not created")
	}
	if a.TrashedAt == nil {
		return errors.New("datei not trashed")
	}

	event := events.DateiRestoredEvent{
		ID:         a.ID,
		RestoredBy: restoredBy,
		RestoredAt: now,
	}

	a.recordEvent(event)
	return nil
}

// Link creates a symlink
func (a *DateiAggregate) Link(linkedDateiID uuid.UUID, linkedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot link: datei not created")
	}
	if linkedDateiID == uuid.Nil {
		return errors.New("invalid linked datei id")
	}
	if a.LinkedDateiID != nil {
		return errors.New("datei already linked")
	}

	event := events.DateiLinkedEvent{
		ID:            a.ID,
		LinkedDateiID: linkedDateiID,
		LinkedBy:      linkedBy,
		LinkedAt:      now,
	}

	a.recordEvent(event)
	return nil
}

// Unlink removes a symlink
func (a *DateiAggregate) Unlink(unlinkedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot unlink: datei not created")
	}
	if a.LinkedDateiID == nil {
		return errors.New("datei not linked")
	}

	event := events.DateiUnlinkedEvent{
		ID:         a.ID,
		UnlinkedBy: unlinkedBy,
		UnlinkedAt: now,
	}

	a.recordEvent(event)
	return nil
}

// ============================================================================
// Event Application (state reconstruction)
// ============================================================================

// ApplyEvent updates the aggregate state based on the event
func (a *DateiAggregate) ApplyEvent(event events.DomainEvent) {
	switch e := event.(type) {
	case events.DateiCreatedEvent:
		a.ID = e.ID
		a.ParentID = e.ParentID
		a.IsDirectory = e.IsDirectory
		a.Name = e.Name
		a.CreatedBy = e.CreatedBy
		a.CreatedAt = e.CreatedAt
		a.UpdatedAt = e.CreatedAt

	case events.DateiRenamedEvent:
		a.Name = e.NewName
		a.UpdatedAt = e.RenamedAt

	case events.DateiVersionUploadedEvent:
		a.S3Key = &e.S3Key
		a.Size = &e.FileSize
		a.Checksum = &e.Checksum
		a.MimeType = &e.MimeType
		a.ContentMD = e.ContentMD
		a.UpdatedAt = e.UploadedAt

	case events.DateiMovedEvent:
		a.ParentID = e.NewParentID
		a.UpdatedAt = e.MovedAt

	case events.DateiTrashedEvent:
		a.TrashedAt = &e.TrashedAt
		a.TrashedBy = &e.TrashedBy
		a.UpdatedAt = e.TrashedAt

	case events.DateiRestoredEvent:
		a.TrashedAt = nil
		a.TrashedBy = nil
		a.UpdatedAt = e.RestoredAt

	case events.DateiLinkedEvent:
		a.LinkedDateiID = &e.LinkedDateiID
		a.UpdatedAt = e.LinkedAt

	case events.DateiUnlinkedEvent:
		a.LinkedDateiID = nil
		a.UpdatedAt = e.UnlinkedAt
	}
}

// ReplayEvents reconstructs aggregate state from event history
func (a *DateiAggregate) ReplayEvents(domainEvents []events.DomainEvent) error {
	for _, event := range domainEvents {
		a.ApplyEvent(event)
	}
	return nil
}

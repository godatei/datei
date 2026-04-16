package datei

import (
	"errors"
	"time"

	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
)

// Aggregate represents the domain entity for a file or directory
type Aggregate struct {
	events.Base[Aggregate, DateiEvent]

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
	UpdatedBy uuid.UUID
}

func (a *Aggregate) AggregateID() uuid.UUID { return a.ID }

func (a *Aggregate) recordEvent(event DateiEvent) {
	a.RecordEvent(a, event)
}

// Replay reconstructs aggregate state from event history.
func (a *Aggregate) Replay(domainEvents []events.DomainEvent) {
	a.ReplayEvents(a, domainEvents)
}

// ============================================================================
// Commands
// ============================================================================

func (a *Aggregate) Create(
	id uuid.UUID,
	parentID *uuid.UUID,
	isDirectory bool,
	name string,
	createdBy uuid.UUID,
	now time.Time,
) error {
	if id == uuid.Nil {
		return errors.New("invalid datei id")
	}
	if name == "" {
		return errors.New("name cannot be empty")
	}

	a.recordEvent(DateiCreatedEvent{
		ID:          id,
		ParentID:    parentID,
		IsDirectory: isDirectory,
		Name:        name,
		CreatedBy:   createdBy,
		CreatedAt:   now,
	})
	return nil
}

func (a *Aggregate) Rename(newName string, renamedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot rename: datei not created")
	}
	if newName == "" {
		return errors.New("name cannot be empty")
	}
	if newName == a.Name {
		return errors.New("new name is same as current name")
	}

	a.recordEvent(DateiRenamedEvent{
		ID:        a.ID,
		OldName:   a.Name,
		NewName:   newName,
		RenamedBy: renamedBy,
		RenamedAt: now,
	})
	return nil
}

func (a *Aggregate) UploadVersion(
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

	a.recordEvent(DateiVersionUploadedEvent{
		ID:         a.ID,
		S3Key:      s3Key,
		FileSize:   fileSize,
		Checksum:   checksum,
		MimeType:   mimeType,
		ContentMD:  contentMD,
		UploadedBy: uploadedBy,
		UploadedAt: now,
	})
	return nil
}

func (a *Aggregate) Move(newParentID *uuid.UUID, movedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot move: datei not created")
	}
	if newParentID != nil && *newParentID == a.ID {
		return errors.New("cannot move datei to itself")
	}

	a.recordEvent(DateiMovedEvent{
		ID:          a.ID,
		OldParentID: a.ParentID,
		NewParentID: newParentID,
		MovedBy:     movedBy,
		MovedAt:     now,
	})
	return nil
}

func (a *Aggregate) Trash(trashedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot trash: datei not created")
	}
	if a.TrashedAt != nil {
		return errors.New("datei already trashed")
	}

	a.recordEvent(DateiTrashedEvent{
		ID:        a.ID,
		TrashedBy: trashedBy,
		TrashedAt: now,
	})
	return nil
}

func (a *Aggregate) Restore(restoredBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot restore: datei not created")
	}
	if a.TrashedAt == nil {
		return errors.New("datei not trashed")
	}

	a.recordEvent(DateiRestoredEvent{
		ID:         a.ID,
		RestoredBy: restoredBy,
		RestoredAt: now,
	})
	return nil
}

func (a *Aggregate) Link(linkedDateiID uuid.UUID, linkedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot link: datei not created")
	}
	if linkedDateiID == uuid.Nil {
		return errors.New("invalid linked datei id")
	}
	if a.LinkedDateiID != nil {
		return errors.New("datei already linked")
	}

	a.recordEvent(DateiLinkedEvent{
		ID:            a.ID,
		LinkedDateiID: linkedDateiID,
		LinkedBy:      linkedBy,
		LinkedAt:      now,
	})
	return nil
}

func (a *Aggregate) Unlink(unlinkedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot unlink: datei not created")
	}
	if a.LinkedDateiID == nil {
		return errors.New("datei not linked")
	}

	a.recordEvent(DateiUnlinkedEvent{
		ID:         a.ID,
		UnlinkedBy: unlinkedBy,
		UnlinkedAt: now,
	})
	return nil
}

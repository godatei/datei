package file

import (
	"errors"
	"time"

	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
)

// Aggregate represents the domain entity for a file or directory
type Aggregate struct {
	events.Base[Aggregate, FileEvent]

	// Identity
	ID           uuid.UUID
	ParentID     *uuid.UUID
	IsDirectory  bool
	LinkedFileID *uuid.UUID

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

func (a *Aggregate) recordEvent(event FileEvent) {
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
		return errors.New("invalid file id")
	}
	if name == "" {
		return errors.New("name cannot be empty")
	}

	a.recordEvent(FileCreatedEvent{
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
		return errors.New("cannot rename: file not created")
	}
	if newName == "" {
		return errors.New("name cannot be empty")
	}
	if newName == a.Name {
		return errors.New("new name is same as current name")
	}

	a.recordEvent(FileRenamedEvent{
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
		return errors.New("cannot upload: file not created")
	}
	if a.IsDirectory {
		return errors.New("cannot upload file to directory")
	}
	if s3Key == "" {
		return errors.New("s3_key cannot be empty")
	}

	a.recordEvent(FileVersionUploadedEvent{
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
		return errors.New("cannot move: file not created")
	}
	if newParentID != nil && *newParentID == a.ID {
		return errors.New("cannot move file to itself")
	}

	a.recordEvent(FileMovedEvent{
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
		return errors.New("cannot trash: file not created")
	}
	if a.TrashedAt != nil {
		return errors.New("file already trashed")
	}

	a.recordEvent(FileTrashedEvent{
		ID:        a.ID,
		TrashedBy: trashedBy,
		TrashedAt: now,
	})
	return nil
}

func (a *Aggregate) Restore(restoredBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot restore: file not created")
	}
	if a.TrashedAt == nil {
		return errors.New("file not trashed")
	}

	a.recordEvent(FileRestoredEvent{
		ID:         a.ID,
		RestoredBy: restoredBy,
		RestoredAt: now,
	})
	return nil
}

func (a *Aggregate) Link(linkedFileID uuid.UUID, linkedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot link: file not created")
	}
	if linkedFileID == uuid.Nil {
		return errors.New("invalid linked file id")
	}
	if a.LinkedFileID != nil {
		return errors.New("file already linked")
	}

	a.recordEvent(FileLinkedEvent{
		ID:           a.ID,
		LinkedFileID: linkedFileID,
		LinkedBy:     linkedBy,
		LinkedAt:     now,
	})
	return nil
}

func (a *Aggregate) Unlink(unlinkedBy uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot unlink: file not created")
	}
	if a.LinkedFileID == nil {
		return errors.New("file not linked")
	}

	a.recordEvent(FileUnlinkedEvent{
		ID:         a.ID,
		UnlinkedBy: unlinkedBy,
		UnlinkedAt: now,
	})
	return nil
}

package link

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
)

const linkNameMaxLen = 255

// Aggregate represents the domain entity for a public-share link.
type Aggregate struct {
	events.Base[Aggregate, LinkEvent]

	ID        uuid.UUID
	OwnerID   uuid.UUID
	Name      string
	Key       string
	Code      *string
	ExpiresAt *time.Time
	RevokedAt *time.Time
	fileIDs   map[uuid.UUID]struct{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (a *Aggregate) AggregateID() uuid.UUID { return a.ID }

func (a *Aggregate) recordEvent(event LinkEvent) {
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
	ownerID uuid.UUID,
	name string,
	key string,
	code *string,
	expiresAt *time.Time,
	fileIDs []uuid.UUID,
	now time.Time,
) error {
	if id == uuid.Nil {
		return errors.New("invalid link id")
	}
	if ownerID == uuid.Nil {
		return errors.New("invalid owner id")
	}
	if err := validateName(name); err != nil {
		return err
	}
	if err := validateExpiresAt(expiresAt, now); err != nil {
		return err
	}
	if key == "" {
		return errors.New("key cannot be empty")
	}

	a.recordEvent(LinkCreatedEvent{
		ID:        id,
		OwnerID:   ownerID,
		Name:      name,
		Key:       key,
		Code:      code,
		ExpiresAt: expiresAt,
		FileIDs:   fileIDs,
		CreatedAt: now,
	})
	return nil
}

// Update records a single LinkUpdatedEvent for batched edits to name, code,
// and expiration (all driven by the same edit modal save). The caller is
// expected to pass the desired absolute state for every field. The event is
// only recorded if at least one of the three differs from the current state.
func (a *Aggregate) Update(name string, code *string, expiresAt *time.Time, now time.Time) error {
	if err := a.checkActive("update"); err != nil {
		return err
	}
	if err := validateName(name); err != nil {
		return err
	}

	nameSame := name == a.Name
	codeSame := (a.Code == nil && code == nil) ||
		(a.Code != nil && code != nil && *a.Code == *code)
	expirySame := (a.ExpiresAt == nil && expiresAt == nil) ||
		(a.ExpiresAt != nil && expiresAt != nil && a.ExpiresAt.Equal(*expiresAt))
	if nameSame && codeSame && expirySame {
		return nil
	}

	// Only enforce the future-expiry rule when the field is actually changing,
	// otherwise renaming an already-expired link would be impossible.
	if !expirySame {
		if err := validateExpiresAt(expiresAt, now); err != nil {
			return err
		}
	}

	a.recordEvent(LinkUpdatedEvent{
		ID:        a.ID,
		Name:      name,
		Code:      code,
		ExpiresAt: expiresAt,
		UpdatedAt: now,
	})
	return nil
}

func (a *Aggregate) RotateKey(newKey string, now time.Time) error {
	if err := a.checkActive("rotate key"); err != nil {
		return err
	}
	if newKey == "" {
		return errors.New("key cannot be empty")
	}
	if newKey == a.Key {
		return errors.New("new key is same as current key")
	}

	a.recordEvent(LinkKeyRotatedEvent{
		ID:        a.ID,
		OldKey:    a.Key,
		NewKey:    newKey,
		RotatedAt: now,
	})
	return nil
}

func (a *Aggregate) AddFile(fileID uuid.UUID, now time.Time) error {
	if err := a.checkActive("add file"); err != nil {
		return err
	}
	if fileID == uuid.Nil {
		return errors.New("invalid file id")
	}
	if _, exists := a.fileIDs[fileID]; exists {
		return fmt.Errorf("file already added to link: %w", apperrors.ErrLinkFileAlreadyAdded)
	}

	a.recordEvent(LinkFileAddedEvent{
		ID:      a.ID,
		FileID:  fileID,
		AddedAt: now,
	})
	return nil
}

func (a *Aggregate) RemoveFile(fileID uuid.UUID, now time.Time) error {
	if err := a.checkActive("remove file"); err != nil {
		return err
	}
	if _, exists := a.fileIDs[fileID]; !exists {
		return fmt.Errorf("file not part of link: %w", apperrors.ErrLinkFileNotShared)
	}

	a.recordEvent(LinkFileRemovedEvent{
		ID:        a.ID,
		FileID:    fileID,
		RemovedAt: now,
	})
	return nil
}

// RecordOpen is invoked by the unlock endpoint after the key + code check
// succeeds. The event is the source of truth for the open counter; the
// projection handler increments link_projection.open_count in the same
// transaction.
func (a *Aggregate) RecordOpen(now time.Time) error {
	if err := a.checkActive("record open"); err != nil {
		return err
	}
	a.recordEvent(LinkOpenedEvent{
		ID:       a.ID,
		OpenedAt: now,
	})
	return nil
}

func (a *Aggregate) Revoke(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot revoke: link not created")
	}
	if a.RevokedAt != nil {
		return fmt.Errorf("link already revoked: %w", apperrors.ErrLinkRevoked)
	}

	a.recordEvent(LinkRevokedEvent{
		ID:        a.ID,
		RevokedAt: now,
	})
	return nil
}

func (a *Aggregate) checkActive(action string) error {
	if a.ID == uuid.Nil {
		return errors.New("cannot " + action + ": link not created")
	}
	if a.RevokedAt != nil {
		return fmt.Errorf("cannot %s: %w", action, apperrors.ErrLinkRevoked)
	}
	return nil
}

// validateName enforces the same rules as the rename dialog's form-level
// validators (required, non-whitespace-only, max length) at the domain layer.
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name cannot be empty: %w", apperrors.ErrInvalidInput)
	}
	if len(name) > linkNameMaxLen {
		return fmt.Errorf("name exceeds %d chars: %w", linkNameMaxLen, apperrors.ErrInvalidInput)
	}
	return nil
}

// validateExpiresAt rejects expirations at or before now. A nil expiresAt
// (meaning "never expires") is always valid.
func validateExpiresAt(expiresAt *time.Time, now time.Time) error {
	if expiresAt == nil {
		return nil
	}
	if !expiresAt.After(now) {
		return fmt.Errorf("expiration must be in the future: %w", apperrors.ErrInvalidInput)
	}
	return nil
}

package link

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/godatei/datei/internal/dateierrors"
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
	dateiIDs  map[uuid.UUID]struct{}
	OpenCount int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HasDatei reports whether the given datei is part of the link's shared set.
// The set is private so callers can't mutate aggregate state outside of
// commands.
func (a *Aggregate) HasDatei(id uuid.UUID) bool {
	_, ok := a.dateiIDs[id]
	return ok
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
	dateiIDs []uuid.UUID,
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
		DateiIDs:  dateiIDs,
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

func (a *Aggregate) AddDatei(dateiID uuid.UUID, now time.Time) error {
	if err := a.checkActive("add datei"); err != nil {
		return err
	}
	if dateiID == uuid.Nil {
		return errors.New("invalid datei id")
	}
	if _, exists := a.dateiIDs[dateiID]; exists {
		return fmt.Errorf("datei already added to link: %w", dateierrors.ErrLinkDateiAlreadyAdded)
	}

	a.recordEvent(LinkDateiAddedEvent{
		ID:      a.ID,
		DateiID: dateiID,
		AddedAt: now,
	})
	return nil
}

func (a *Aggregate) RemoveDatei(dateiID uuid.UUID, now time.Time) error {
	if err := a.checkActive("remove datei"); err != nil {
		return err
	}
	if _, exists := a.dateiIDs[dateiID]; !exists {
		return fmt.Errorf("datei not part of link: %w", dateierrors.ErrLinkDateiNotShared)
	}

	a.recordEvent(LinkDateiRemovedEvent{
		ID:        a.ID,
		DateiID:   dateiID,
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
		return fmt.Errorf("link already revoked: %w", dateierrors.ErrLinkRevoked)
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
		return fmt.Errorf("cannot %s: %w", action, dateierrors.ErrLinkRevoked)
	}
	return nil
}

// validateName enforces the same rules as the rename dialog's form-level
// validators (required, non-whitespace-only, max length) at the domain layer.
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name cannot be empty: %w", dateierrors.ErrInvalidInput)
	}
	if len(name) > linkNameMaxLen {
		return fmt.Errorf("name exceeds %d chars: %w", linkNameMaxLen, dateierrors.ErrInvalidInput)
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
		return fmt.Errorf("expiration must be in the future: %w", dateierrors.ErrInvalidInput)
	}
	return nil
}

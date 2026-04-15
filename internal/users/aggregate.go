package users

import (
	"errors"
	"time"

	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
)

// Aggregate represents a user account domain entity
type Aggregate struct {
	ID   uuid.UUID
	Name string

	Email         string
	EmailID       uuid.UUID
	EmailVerified bool
	PasswordHash  []byte
	PasswordSalt  []byte

	MFASecret  *string
	MFAEnabled bool

	ArchivedAt     *time.Time
	LastLoggedInAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time

	uncommittedEvents []events.DomainEvent
	version           int
}

func (a *Aggregate) GetUncommittedEvents() []events.DomainEvent {
	return a.uncommittedEvents
}

func (a *Aggregate) MarkEventsAsCommitted() {
	a.uncommittedEvents = []events.DomainEvent{}
}

func (a *Aggregate) Version() int {
	return a.version
}

func (a *Aggregate) recordEvent(event events.DomainEvent) {
	a.uncommittedEvents = append(a.uncommittedEvents, event)
	a.version++
	a.ApplyEvent(event)
}

// ApplyEvent updates aggregate state by delegating to the event's ApplyTo method.
func (a *Aggregate) ApplyEvent(event events.DomainEvent) {
	event.(Event).ApplyTo(a)
}

// ReplayEvents reconstructs aggregate state from event history
func (a *Aggregate) ReplayEvents(domainEvents []events.DomainEvent) error {
	for _, event := range domainEvents {
		a.ApplyEvent(event)
	}
	return nil
}

// ============================================================================
// Commands
// ============================================================================

func (a *Aggregate) Register(
	id uuid.UUID, name, email string, emailID uuid.UUID,
	passwordHash, passwordSalt []byte, now time.Time,
) error {
	if id == uuid.Nil {
		return errors.New("invalid user id")
	}
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if email == "" {
		return errors.New("email cannot be empty")
	}

	a.recordEvent(UserRegisteredEvent{
		ID:           id,
		Name:         name,
		Email:        email,
		EmailID:      emailID,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		CreatedAt:    now,
	})
	return nil
}

func (a *Aggregate) ChangeName(newName string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if newName == "" {
		return errors.New("name cannot be empty")
	}
	if newName == a.Name {
		return nil
	}

	a.recordEvent(UserNameChangedEvent{
		ID:        a.ID,
		NewName:   newName,
		ChangedAt: now,
	})
	return nil
}

func (a *Aggregate) ChangePassword(passwordHash, passwordSalt []byte, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(UserPasswordChangedEvent{
		ID:           a.ID,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		ChangedAt:    now,
	})
	return nil
}

func (a *Aggregate) ChangeEmail(oldEmail, newEmail string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if newEmail == "" {
		return errors.New("email cannot be empty")
	}

	a.recordEvent(UserEmailChangedEvent{
		ID:        a.ID,
		OldEmail:  oldEmail,
		NewEmail:  newEmail,
		ChangedAt: now,
	})
	return nil
}

func (a *Aggregate) VerifyEmail(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.EmailVerified {
		return errors.New("email already verified")
	}

	a.recordEvent(UserEmailVerifiedEvent{
		ID:         a.ID,
		VerifiedAt: now,
	})
	return nil
}

func (a *Aggregate) AddEmail(emailID uuid.UUID, email string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if email == "" {
		return errors.New("email cannot be empty")
	}

	a.recordEvent(UserEmailAddedEvent{
		ID:      a.ID,
		EmailID: emailID,
		Email:   email,
		AddedAt: now,
	})
	return nil
}

func (a *Aggregate) RemoveEmail(emailID uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(UserEmailRemovedEvent{
		ID:        a.ID,
		EmailID:   emailID,
		RemovedAt: now,
	})
	return nil
}

func (a *Aggregate) SetPrimaryEmail(
	oldPrimaryEmailID, newPrimaryEmailID uuid.UUID, now time.Time,
) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(UserEmailSetPrimaryEvent{
		ID:                a.ID,
		OldPrimaryEmailID: oldPrimaryEmailID,
		NewPrimaryEmailID: newPrimaryEmailID,
		ChangedAt:         now,
	})
	return nil
}

func (a *Aggregate) InitiateMFASetup(secret string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.MFAEnabled {
		return errors.New("MFA is already enabled")
	}

	a.recordEvent(UserMFASetupInitiatedEvent{
		ID:          a.ID,
		MFASecret:   secret,
		InitiatedAt: now,
	})
	return nil
}

func (a *Aggregate) EnableMFA(codes []HashedRecoveryCode, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.MFAEnabled {
		return errors.New("MFA is already enabled")
	}
	if a.MFASecret == nil {
		return errors.New("MFA not set up")
	}

	a.recordEvent(UserMFAEnabledEvent{
		ID:            a.ID,
		RecoveryCodes: codes,
		EnabledAt:     now,
	})
	return nil
}

func (a *Aggregate) DisableMFA(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if !a.MFAEnabled {
		return errors.New("MFA is not enabled")
	}

	a.recordEvent(UserMFADisabledEvent{
		ID:         a.ID,
		DisabledAt: now,
	})
	return nil
}

func (a *Aggregate) UseRecoveryCode(codeID uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(UserMFARecoveryCodeUsedEvent{
		ID:             a.ID,
		RecoveryCodeID: codeID,
		UsedAt:         now,
	})
	return nil
}

func (a *Aggregate) RegenerateRecoveryCodes(codes []HashedRecoveryCode, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if !a.MFAEnabled {
		return errors.New("MFA is not enabled")
	}

	a.recordEvent(UserMFARecoveryCodesRegeneratedEvent{
		ID:            a.ID,
		RecoveryCodes: codes,
		RegeneratedAt: now,
	})
	return nil
}

func (a *Aggregate) Archive(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.ArchivedAt != nil {
		return errors.New("user already archived")
	}

	a.recordEvent(UserArchivedEvent{
		ID:         a.ID,
		ArchivedAt: now,
	})
	return nil
}

func (a *Aggregate) RecordLogin(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(UserLoggedInEvent{
		ID:         a.ID,
		LoggedInAt: now,
	})
	return nil
}

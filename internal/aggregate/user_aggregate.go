package aggregate

import (
	"errors"
	"time"

	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
)

// UserAggregate represents a user account domain entity
type UserAggregate struct {
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

func (a *UserAggregate) GetUncommittedEvents() []events.DomainEvent {
	return a.uncommittedEvents
}

func (a *UserAggregate) MarkEventsAsCommitted() {
	a.uncommittedEvents = []events.DomainEvent{}
}

func (a *UserAggregate) Version() int {
	return a.version
}

func (a *UserAggregate) recordEvent(event events.DomainEvent) {
	a.uncommittedEvents = append(a.uncommittedEvents, event)
	a.version++
	a.ApplyEvent(event)
}

// ============================================================================
// Commands
// ============================================================================

func (a *UserAggregate) Register(
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

	a.recordEvent(events.UserRegisteredEvent{
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

func (a *UserAggregate) ChangeName(newName string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if newName == "" {
		return errors.New("name cannot be empty")
	}
	if newName == a.Name {
		return nil
	}

	a.recordEvent(events.UserNameChangedEvent{
		ID:        a.ID,
		NewName:   newName,
		ChangedAt: now,
	})
	return nil
}

func (a *UserAggregate) ChangePassword(passwordHash, passwordSalt []byte, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(events.UserPasswordChangedEvent{
		ID:           a.ID,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		ChangedAt:    now,
	})
	return nil
}

func (a *UserAggregate) ChangeEmail(oldEmail, newEmail string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if newEmail == "" {
		return errors.New("email cannot be empty")
	}

	a.recordEvent(events.UserEmailChangedEvent{
		ID:        a.ID,
		OldEmail:  oldEmail,
		NewEmail:  newEmail,
		ChangedAt: now,
	})
	return nil
}

func (a *UserAggregate) VerifyEmail(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.EmailVerified {
		return errors.New("email already verified")
	}

	a.recordEvent(events.UserEmailVerifiedEvent{
		ID:         a.ID,
		VerifiedAt: now,
	})
	return nil
}

func (a *UserAggregate) AddEmail(emailID uuid.UUID, email string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if email == "" {
		return errors.New("email cannot be empty")
	}

	a.recordEvent(events.UserEmailAddedEvent{
		ID:      a.ID,
		EmailID: emailID,
		Email:   email,
		AddedAt: now,
	})
	return nil
}

func (a *UserAggregate) RemoveEmail(emailID uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(events.UserEmailRemovedEvent{
		ID:        a.ID,
		EmailID:   emailID,
		RemovedAt: now,
	})
	return nil
}

func (a *UserAggregate) SetPrimaryEmail(
	oldPrimaryEmailID, newPrimaryEmailID uuid.UUID, now time.Time,
) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(events.UserEmailSetPrimaryEvent{
		ID:                a.ID,
		OldPrimaryEmailID: oldPrimaryEmailID,
		NewPrimaryEmailID: newPrimaryEmailID,
		ChangedAt:         now,
	})
	return nil
}

func (a *UserAggregate) InitiateMFASetup(secret string, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.MFAEnabled {
		return errors.New("MFA is already enabled")
	}

	a.recordEvent(events.UserMFASetupInitiatedEvent{
		ID:          a.ID,
		MFASecret:   secret,
		InitiatedAt: now,
	})
	return nil
}

func (a *UserAggregate) EnableMFA(codes []events.HashedRecoveryCode, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.MFAEnabled {
		return errors.New("MFA is already enabled")
	}
	if a.MFASecret == nil {
		return errors.New("MFA not set up")
	}

	a.recordEvent(events.UserMFAEnabledEvent{
		ID:            a.ID,
		RecoveryCodes: codes,
		EnabledAt:     now,
	})
	return nil
}

func (a *UserAggregate) DisableMFA(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if !a.MFAEnabled {
		return errors.New("MFA is not enabled")
	}

	a.recordEvent(events.UserMFADisabledEvent{
		ID:         a.ID,
		DisabledAt: now,
	})
	return nil
}

func (a *UserAggregate) UseRecoveryCode(codeID uuid.UUID, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(events.UserMFARecoveryCodeUsedEvent{
		ID:             a.ID,
		RecoveryCodeID: codeID,
		UsedAt:         now,
	})
	return nil
}

func (a *UserAggregate) RegenerateRecoveryCodes(codes []events.HashedRecoveryCode, now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if !a.MFAEnabled {
		return errors.New("MFA is not enabled")
	}

	a.recordEvent(events.UserMFARecoveryCodesRegeneratedEvent{
		ID:            a.ID,
		RecoveryCodes: codes,
		RegeneratedAt: now,
	})
	return nil
}

func (a *UserAggregate) Archive(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}
	if a.ArchivedAt != nil {
		return errors.New("user already archived")
	}

	a.recordEvent(events.UserArchivedEvent{
		ID:         a.ID,
		ArchivedAt: now,
	})
	return nil
}

func (a *UserAggregate) RecordLogin(now time.Time) error {
	if a.ID == uuid.Nil {
		return errors.New("user not created")
	}

	a.recordEvent(events.UserLoggedInEvent{
		ID:         a.ID,
		LoggedInAt: now,
	})
	return nil
}

// ============================================================================
// Event Application
// ============================================================================

func (a *UserAggregate) ApplyEvent(event events.DomainEvent) {
	switch e := event.(type) {
	case events.UserRegisteredEvent:
		a.ID = e.ID
		a.Name = e.Name
		a.Email = e.Email
		a.EmailID = e.EmailID
		a.PasswordHash = e.PasswordHash
		a.PasswordSalt = e.PasswordSalt
		a.CreatedAt = e.CreatedAt
		a.UpdatedAt = e.CreatedAt

	case events.UserNameChangedEvent:
		a.Name = e.NewName
		a.UpdatedAt = e.ChangedAt

	case events.UserPasswordChangedEvent:
		a.PasswordHash = e.PasswordHash
		a.PasswordSalt = e.PasswordSalt
		a.UpdatedAt = e.ChangedAt

	case events.UserEmailChangedEvent:
		a.Email = e.NewEmail
		a.EmailVerified = false
		a.UpdatedAt = e.ChangedAt

	case events.UserEmailVerifiedEvent:
		a.EmailVerified = true
		a.UpdatedAt = e.VerifiedAt

	case events.UserEmailAddedEvent:
		a.UpdatedAt = e.AddedAt

	case events.UserEmailRemovedEvent:
		a.UpdatedAt = e.RemovedAt

	case events.UserEmailSetPrimaryEvent:
		a.EmailID = e.NewPrimaryEmailID
		a.UpdatedAt = e.ChangedAt

	case events.UserMFASetupInitiatedEvent:
		a.MFASecret = &e.MFASecret
		a.UpdatedAt = e.InitiatedAt

	case events.UserMFAEnabledEvent:
		a.MFAEnabled = true
		a.UpdatedAt = e.EnabledAt

	case events.UserMFADisabledEvent:
		a.MFAEnabled = false
		a.MFASecret = nil
		a.UpdatedAt = e.DisabledAt

	case events.UserMFARecoveryCodeUsedEvent:
		a.UpdatedAt = e.UsedAt

	case events.UserMFARecoveryCodesRegeneratedEvent:
		a.UpdatedAt = e.RegeneratedAt

	case events.UserArchivedEvent:
		a.ArchivedAt = &e.ArchivedAt
		a.UpdatedAt = e.ArchivedAt

	case events.UserLoggedInEvent:
		a.LastLoggedInAt = &e.LoggedInAt
	}
}

func (a *UserAggregate) ReplayEvents(domainEvents []events.DomainEvent) error {
	for _, event := range domainEvents {
		a.ApplyEvent(event)
	}
	return nil
}

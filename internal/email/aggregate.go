package email

import (
	"errors"
	"time"

	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
)

// Security is the transport security used for a mail account's IMAP connection.
type Security string

const (
	SecuritySSL      Security = "ssl"
	SecuritySTARTTLS Security = "starttls"
	SecurityNone     Security = "none"
)

func (s Security) valid() bool {
	switch s {
	case SecuritySSL, SecuritySTARTTLS, SecurityNone:
		return true
	default:
		return false
	}
}

// Action is performed on a mail after its attachments are consumed.
type Action string

const ActionMarkAsRead Action = "mark_as_read"

func (a Action) valid() bool {
	return a == ActionMarkAsRead
}

// ============================================================================
// Mail Account Aggregate
// ============================================================================

// MailAccount is the event-sourced configuration for one IMAP mailbox.
type MailAccount struct {
	events.Base[MailAccount, MailAccountEvent]

	ID                uuid.UUID
	OwnerID           uuid.UUID
	Name              string
	ImapHost          string
	ImapPort          int
	Username          string
	PasswordEncrypted []byte
	Security          Security
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

func (a *MailAccount) AggregateID() uuid.UUID { return a.ID }

func (a *MailAccount) recordEvent(event MailAccountEvent) {
	a.RecordEvent(a, event)
}

// Replay reconstructs aggregate state from event history.
func (a *MailAccount) Replay(domainEvents []events.DomainEvent) {
	a.ReplayEvents(a, domainEvents)
}

func (a *MailAccount) Create(
	id, ownerID uuid.UUID,
	name, imapHost string,
	imapPort int,
	username string,
	passwordEncrypted []byte,
	security Security,
	now time.Time,
) error {
	if id == uuid.Nil {
		return errors.New("invalid mail account id")
	}
	if ownerID == uuid.Nil {
		return errors.New("invalid owner id")
	}
	if name == "" || imapHost == "" || username == "" {
		return errors.New("name, imap host and username are required")
	}
	if imapPort < 1 || imapPort > 65535 {
		return errors.New("imap port out of range")
	}
	if len(passwordEncrypted) == 0 {
		return errors.New("password is required")
	}
	if !security.valid() {
		return errors.New("invalid security")
	}

	a.recordEvent(MailAccountCreatedEvent{
		ID:                id,
		OwnerID:           ownerID,
		Name:              name,
		ImapHost:          imapHost,
		ImapPort:          imapPort,
		Username:          username,
		PasswordEncrypted: passwordEncrypted,
		Security:          security,
		CreatedAt:         now,
	})
	return nil
}

func (a *MailAccount) Update(
	name, imapHost string,
	imapPort int,
	username string,
	passwordEncrypted []byte,
	security Security,
	now time.Time,
) error {
	if a.ID == uuid.Nil || a.DeletedAt != nil {
		return errors.New("cannot update: account does not exist")
	}
	if name == "" || imapHost == "" || username == "" {
		return errors.New("name, imap host and username are required")
	}
	if imapPort < 1 || imapPort > 65535 {
		return errors.New("imap port out of range")
	}
	if len(passwordEncrypted) == 0 {
		return errors.New("password is required")
	}
	if !security.valid() {
		return errors.New("invalid security")
	}

	a.recordEvent(MailAccountUpdatedEvent{
		ID:                a.ID,
		Name:              name,
		ImapHost:          imapHost,
		ImapPort:          imapPort,
		Username:          username,
		PasswordEncrypted: passwordEncrypted,
		Security:          security,
		UpdatedAt:         now,
	})
	return nil
}

func (a *MailAccount) Delete(now time.Time) error {
	if a.ID == uuid.Nil || a.DeletedAt != nil {
		return errors.New("cannot delete: account does not exist")
	}
	a.recordEvent(MailAccountDeletedEvent{ID: a.ID, DeletedAt: now})
	return nil
}

// ============================================================================
// Mail Rule Aggregate
// ============================================================================

// MailRule is the event-sourced definition of how a mailbox's messages are
// matched and what is done with their attachments.
type MailRule struct {
	events.Base[MailRule, MailRuleEvent]

	ID                uuid.UUID
	AccountID         uuid.UUID
	Name              string
	Order             int
	Enabled           bool
	Folder            string
	FilterFrom        *string
	FilterSubject     *string
	MaxAgeDays        int
	AttachmentPattern *string
	Action            Action
	TargetDirectoryID *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

func (r *MailRule) AggregateID() uuid.UUID { return r.ID }

func (r *MailRule) recordEvent(event MailRuleEvent) {
	r.RecordEvent(r, event)
}

// Replay reconstructs aggregate state from event history.
func (r *MailRule) Replay(domainEvents []events.DomainEvent) {
	r.ReplayEvents(r, domainEvents)
}

// RuleSpec holds the mutable fields shared by create and update.
type RuleSpec struct {
	Name              string
	Order             int
	Enabled           bool
	Folder            string
	FilterFrom        *string
	FilterSubject     *string
	MaxAgeDays        int
	AttachmentPattern *string
	Action            Action
	TargetDirectoryID *uuid.UUID
}

func (s RuleSpec) validate() error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.Folder == "" {
		return errors.New("folder is required")
	}
	if s.MaxAgeDays < 1 {
		return errors.New("max age days must be positive")
	}
	if !s.Action.valid() {
		return errors.New("invalid action")
	}
	return nil
}

func (r *MailRule) Create(id, accountID uuid.UUID, spec RuleSpec, now time.Time) error {
	if id == uuid.Nil {
		return errors.New("invalid mail rule id")
	}
	if accountID == uuid.Nil {
		return errors.New("invalid account id")
	}
	if err := spec.validate(); err != nil {
		return err
	}

	r.recordEvent(MailRuleCreatedEvent{
		ID:                id,
		AccountID:         accountID,
		Name:              spec.Name,
		Order:             spec.Order,
		Enabled:           spec.Enabled,
		Folder:            spec.Folder,
		FilterFrom:        spec.FilterFrom,
		FilterSubject:     spec.FilterSubject,
		MaxAgeDays:        spec.MaxAgeDays,
		AttachmentPattern: spec.AttachmentPattern,
		Action:            spec.Action,
		TargetDirectoryID: spec.TargetDirectoryID,
		CreatedAt:         now,
	})
	return nil
}

func (r *MailRule) Update(accountID uuid.UUID, spec RuleSpec, now time.Time) error {
	if r.ID == uuid.Nil || r.DeletedAt != nil {
		return errors.New("cannot update: rule does not exist")
	}
	if accountID == uuid.Nil {
		return errors.New("invalid account id")
	}
	if err := spec.validate(); err != nil {
		return err
	}

	r.recordEvent(MailRuleUpdatedEvent{
		ID:                r.ID,
		AccountID:         accountID,
		Name:              spec.Name,
		Order:             spec.Order,
		Enabled:           spec.Enabled,
		Folder:            spec.Folder,
		FilterFrom:        spec.FilterFrom,
		FilterSubject:     spec.FilterSubject,
		MaxAgeDays:        spec.MaxAgeDays,
		AttachmentPattern: spec.AttachmentPattern,
		Action:            spec.Action,
		TargetDirectoryID: spec.TargetDirectoryID,
		UpdatedAt:         now,
	})
	return nil
}

func (r *MailRule) Delete(now time.Time) error {
	if r.ID == uuid.Nil || r.DeletedAt != nil {
		return errors.New("cannot delete: rule does not exist")
	}
	r.recordEvent(MailRuleDeletedEvent{ID: r.ID, DeletedAt: now})
	return nil
}

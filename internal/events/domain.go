package events

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the base interface for all domain events
type DomainEvent interface {
	EventType() string
	StreamID() uuid.UUID
}

// ============================================================================
// Datei Events
// ============================================================================

// DateiCreatedEvent fired when a new datei (file or directory) is created
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

// DateiRenamedEvent fired when a datei name is changed
type DateiRenamedEvent struct {
	ID        uuid.UUID `json:"id"`
	OldName   string    `json:"old_name"`
	NewName   string    `json:"new_name"`
	RenamedBy uuid.UUID `json:"renamed_by"`
	RenamedAt time.Time `json:"renamed_at"`
}

func (e DateiRenamedEvent) EventType() string   { return "DateiRenamed" }
func (e DateiRenamedEvent) StreamID() uuid.UUID { return e.ID }

// DateiVersionUploadedEvent fired when a new file version is uploaded
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

// DateiMovedEvent fired when a datei is moved to a different parent directory
type DateiMovedEvent struct {
	ID          uuid.UUID  `json:"id"`
	OldParentID *uuid.UUID `json:"old_parent_id,omitempty"`
	NewParentID *uuid.UUID `json:"new_parent_id,omitempty"`
	MovedBy     uuid.UUID  `json:"moved_by"`
	MovedAt     time.Time  `json:"moved_at"`
}

func (e DateiMovedEvent) EventType() string   { return "DateiMoved" }
func (e DateiMovedEvent) StreamID() uuid.UUID { return e.ID }

// DateiTrashedEvent fired when a datei is moved to trash
type DateiTrashedEvent struct {
	ID        uuid.UUID `json:"id"`
	TrashedBy uuid.UUID `json:"trashed_by"`
	TrashedAt time.Time `json:"trashed_at"`
}

func (e DateiTrashedEvent) EventType() string   { return "DateiTrashed" }
func (e DateiTrashedEvent) StreamID() uuid.UUID { return e.ID }

// DateiRestoredEvent fired when a datei is restored from trash
type DateiRestoredEvent struct {
	ID         uuid.UUID `json:"id"`
	RestoredBy uuid.UUID `json:"restored_by"`
	RestoredAt time.Time `json:"restored_at"`
}

func (e DateiRestoredEvent) EventType() string   { return "DateiRestored" }
func (e DateiRestoredEvent) StreamID() uuid.UUID { return e.ID }

// DateiLinkedEvent fired when a datei is linked as a symlink
type DateiLinkedEvent struct {
	ID            uuid.UUID `json:"id"`
	LinkedDateiID uuid.UUID `json:"linked_datei_id"`
	LinkedBy      uuid.UUID `json:"linked_by"`
	LinkedAt      time.Time `json:"linked_at"`
}

func (e DateiLinkedEvent) EventType() string   { return "DateiLinked" }
func (e DateiLinkedEvent) StreamID() uuid.UUID { return e.ID }

// DateiUnlinkedEvent fired when a datei is unlinked from a target
type DateiUnlinkedEvent struct {
	ID         uuid.UUID `json:"id"`
	UnlinkedBy uuid.UUID `json:"unlinked_by"`
	UnlinkedAt time.Time `json:"unlinked_at"`
}

func (e DateiUnlinkedEvent) EventType() string   { return "DateiUnlinked" }
func (e DateiUnlinkedEvent) StreamID() uuid.UUID { return e.ID }

// ============================================================================
// Permission Events
// ============================================================================

// DateiPermissionGrantedEvent fired when access is granted
type DateiPermissionGrantedEvent struct {
	ID             uuid.UUID  `json:"id"`
	DateiID        uuid.UUID  `json:"datei_id"`
	UserAccountID  *uuid.UUID `json:"user_account_id,omitempty"`
	UserGroupID    *uuid.UUID `json:"user_group_id,omitempty"`
	PermissionType string     `json:"permission_type"`
	GrantedBy      uuid.UUID  `json:"granted_by"`
	GrantedAt      time.Time  `json:"granted_at"`
}

func (e DateiPermissionGrantedEvent) EventType() string   { return "DateiPermissionGranted" }
func (e DateiPermissionGrantedEvent) StreamID() uuid.UUID { return e.DateiID }

// DateiPermissionRevokedEvent fired when access is revoked
type DateiPermissionRevokedEvent struct {
	ID            uuid.UUID  `json:"id"`
	DateiID       uuid.UUID  `json:"datei_id"`
	UserAccountID *uuid.UUID `json:"user_account_id,omitempty"`
	UserGroupID   *uuid.UUID `json:"user_group_id,omitempty"`
	RevokedBy     uuid.UUID  `json:"revoked_by"`
	RevokedAt     time.Time  `json:"revoked_at"`
}

func (e DateiPermissionRevokedEvent) EventType() string   { return "DateiPermissionRevoked" }
func (e DateiPermissionRevokedEvent) StreamID() uuid.UUID { return e.DateiID }

// ============================================================================
// User Account Events
// ============================================================================

// HashedRecoveryCode is stored in events for MFA recovery codes.
type HashedRecoveryCode struct {
	ID       uuid.UUID `json:"id"`
	CodeHash []byte    `json:"code_hash"`
	CodeSalt []byte    `json:"code_salt"`
}

type UserRegisteredEvent struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	EmailID      uuid.UUID `json:"email_id"`
	PasswordHash []byte    `json:"password_hash"`
	PasswordSalt []byte    `json:"password_salt"`
	CreatedAt    time.Time `json:"created_at"`
}

func (e UserRegisteredEvent) EventType() string   { return "UserRegistered" }
func (e UserRegisteredEvent) StreamID() uuid.UUID { return e.ID }

type UserNameChangedEvent struct {
	ID        uuid.UUID `json:"id"`
	NewName   string    `json:"new_name"`
	ChangedAt time.Time `json:"changed_at"`
}

func (e UserNameChangedEvent) EventType() string   { return "UserNameChanged" }
func (e UserNameChangedEvent) StreamID() uuid.UUID { return e.ID }

type UserPasswordChangedEvent struct {
	ID           uuid.UUID `json:"id"`
	PasswordHash []byte    `json:"password_hash"`
	PasswordSalt []byte    `json:"password_salt"`
	ChangedAt    time.Time `json:"changed_at"`
}

func (e UserPasswordChangedEvent) EventType() string   { return "UserPasswordChanged" }
func (e UserPasswordChangedEvent) StreamID() uuid.UUID { return e.ID }

type UserEmailChangedEvent struct {
	ID        uuid.UUID `json:"id"`
	OldEmail  string    `json:"old_email"`
	NewEmail  string    `json:"new_email"`
	ChangedAt time.Time `json:"changed_at"`
}

func (e UserEmailChangedEvent) EventType() string   { return "UserEmailChanged" }
func (e UserEmailChangedEvent) StreamID() uuid.UUID { return e.ID }

type UserEmailVerifiedEvent struct {
	ID         uuid.UUID `json:"id"`
	VerifiedAt time.Time `json:"verified_at"`
}

func (e UserEmailVerifiedEvent) EventType() string   { return "UserEmailVerified" }
func (e UserEmailVerifiedEvent) StreamID() uuid.UUID { return e.ID }

type UserMFASetupInitiatedEvent struct {
	ID          uuid.UUID `json:"id"`
	MFASecret   string    `json:"mfa_secret"`
	InitiatedAt time.Time `json:"initiated_at"`
}

func (e UserMFASetupInitiatedEvent) EventType() string   { return "UserMFASetupInitiated" }
func (e UserMFASetupInitiatedEvent) StreamID() uuid.UUID { return e.ID }

type UserMFAEnabledEvent struct {
	ID            uuid.UUID            `json:"id"`
	RecoveryCodes []HashedRecoveryCode `json:"recovery_codes"`
	EnabledAt     time.Time            `json:"enabled_at"`
}

func (e UserMFAEnabledEvent) EventType() string   { return "UserMFAEnabled" }
func (e UserMFAEnabledEvent) StreamID() uuid.UUID { return e.ID }

type UserMFADisabledEvent struct {
	ID         uuid.UUID `json:"id"`
	DisabledAt time.Time `json:"disabled_at"`
}

func (e UserMFADisabledEvent) EventType() string   { return "UserMFADisabled" }
func (e UserMFADisabledEvent) StreamID() uuid.UUID { return e.ID }

type UserMFARecoveryCodeUsedEvent struct {
	ID             uuid.UUID `json:"id"`
	RecoveryCodeID uuid.UUID `json:"recovery_code_id"`
	UsedAt         time.Time `json:"used_at"`
}

func (e UserMFARecoveryCodeUsedEvent) EventType() string   { return "UserMFARecoveryCodeUsed" }
func (e UserMFARecoveryCodeUsedEvent) StreamID() uuid.UUID { return e.ID }

type UserMFARecoveryCodesRegeneratedEvent struct {
	ID            uuid.UUID            `json:"id"`
	RecoveryCodes []HashedRecoveryCode `json:"recovery_codes"`
	RegeneratedAt time.Time            `json:"regenerated_at"`
}

func (e UserMFARecoveryCodesRegeneratedEvent) EventType() string {
	return "UserMFARecoveryCodesRegenerated"
}
func (e UserMFARecoveryCodesRegeneratedEvent) StreamID() uuid.UUID { return e.ID }

type UserArchivedEvent struct {
	ID         uuid.UUID `json:"id"`
	ArchivedAt time.Time `json:"archived_at"`
}

func (e UserArchivedEvent) EventType() string   { return "UserArchived" }
func (e UserArchivedEvent) StreamID() uuid.UUID { return e.ID }

type UserLoggedInEvent struct {
	ID         uuid.UUID `json:"id"`
	LoggedInAt time.Time `json:"logged_in_at"`
}

func (e UserLoggedInEvent) EventType() string   { return "UserLoggedIn" }
func (e UserLoggedInEvent) StreamID() uuid.UUID { return e.ID }

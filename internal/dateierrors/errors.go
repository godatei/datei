package dateierrors

import "errors"

var (
	// Generic input-validation error, shared across all domains.
	ErrInvalidInput = errors.New("invalid input")

	ErrIsDirectory          = errors.New("cannot download directory")
	ErrNotFound             = errors.New("datei not found")
	ErrNoContent            = errors.New("datei has no content")
	ErrUnsupportedMediaType = errors.New("thumbnail not supported for this file type")
	ErrNotModified          = errors.New("not modified")
	ErrParentNotFound       = errors.New("parent directory not found")
	ErrParentNotDirectory   = errors.New("parent is not a directory")
	ErrParentTrashed        = errors.New("parent directory is trashed")
	ErrParentNotTrashed     = errors.New("parent directory is not trashed")
	ErrCycleDetected        = errors.New("cannot move directory into its own subtree")
	ErrNotInTrash           = errors.New("datei is not in trash")

	// Link / public-share errors
	ErrLinkNotFound          = errors.New("link not found")
	ErrLinkExpired           = errors.New("link expired")
	ErrLinkRevoked           = errors.New("link revoked")
	ErrLinkCodeRequired      = errors.New("link code required or invalid")
	ErrLinkDateiNotShared    = errors.New("datei not in link scope")
	ErrLinkDateiAlreadyAdded = errors.New("datei already added to link")
	ErrLinkForbidden         = errors.New("link operation forbidden")
	ErrLinkUnauthorized      = errors.New("link session token missing, invalid, or expired")

	// User / auth errors
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrEmailAlreadyInUse       = errors.New("email already in use")
	ErrRegistrationDisabled    = errors.New("registration is disabled")
	ErrCurrentPasswordRequired = errors.New("current password required")
	ErrEmailMismatch           = errors.New("email does not match")
	ErrMFARequired             = errors.New("MFA verification required")
	ErrMFAAlreadyEnabled       = errors.New("MFA is already enabled")
	ErrMFANotEnabled           = errors.New("MFA is not enabled")
	ErrMFANotSetUp             = errors.New("MFA has not been set up")
	ErrMFAInvalidCode          = errors.New("invalid MFA code")
)

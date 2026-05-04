package dateierrors

import "errors"

var (
	ErrIsDirectory          = errors.New("cannot download directory")
	ErrNotFound             = errors.New("datei not found")
	ErrNoContent            = errors.New("datei has no content")
	ErrUnsupportedMediaType = errors.New("thumbnail not supported for this file type")
	ErrNotModified          = errors.New("not modified")
	ErrParentNotFound       = errors.New("parent directory not found")
	ErrParentNotDirectory   = errors.New("parent is not a directory")
	ErrParentTrashed        = errors.New("parent directory is trashed")
	ErrCycleDetected        = errors.New("cannot move directory into its own subtree")

	// Link / public-share errors
	ErrLinkNotFound          = errors.New("link not found")
	ErrLinkExpired           = errors.New("link expired")
	ErrLinkRevoked           = errors.New("link revoked")
	ErrLinkCodeRequired      = errors.New("link code required")
	ErrLinkCodeInvalid       = errors.New("link code invalid")
	ErrLinkDateiNotShared    = errors.New("datei not in link scope")
	ErrLinkDateiAlreadyAdded = errors.New("datei already added to link")
	ErrLinkForbidden         = errors.New("link operation forbidden")

	// User / auth errors
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidInput            = errors.New("invalid input")
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

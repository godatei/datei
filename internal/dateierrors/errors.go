package dateierrors

import "errors"

var (
	ErrIsDirectory = errors.New("cannot download directory")
	ErrNotFound    = errors.New("datei not found")
	ErrNoContent   = errors.New("datei has no content")

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

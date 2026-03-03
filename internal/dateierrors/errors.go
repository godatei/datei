package dateierrors

import "errors"

var (
	ErrIsDirectory = errors.New("cannot download directory")
	ErrNotFound    = errors.New("datei not found")
	ErrNoContent   = errors.New("datei has no content")
)

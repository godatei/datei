// Package httpauth provides helpers for parsing HTTP Authorization headers.
package httpauth

import "strings"

// ParseBearer extracts the token from an "Authorization: Bearer <token>" header
// value. The scheme match is case-insensitive (RFC 7235) and surrounding
// whitespace is tolerated. ok is false when the header is empty, not a Bearer
// credential, or malformed.
func ParseBearer(header string) (token string, ok bool) {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", false
	}
	return fields[1], true
}

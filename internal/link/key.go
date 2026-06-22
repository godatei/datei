package link

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
)

var keyEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// generateKey returns a 12-byte random key encoded as base64-url (16 ASCII
// characters), suitable for use as a URL slug. 96 bits of entropy keeps keys
// unguessable while keeping share URLs reasonably short.
func generateKey() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return keyEncoding.EncodeToString(b), nil
}

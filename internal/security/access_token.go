package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// AccessTokenPrefix is prepended to every personal access token presented to
// the client. It is not part of the hashed secret; it exists purely to make
// tokens recognizable (e.g. for secret scanners) and to distinguish them from
// JWTs on the Bearer auth path.
const AccessTokenPrefix = "datei-"

// accessTokenBytes is the number of random bytes in a personal access token.
const accessTokenBytes = 16

// GenerateAccessToken returns a new personal access token. The plaintext is the
// value shown to the user once: AccessTokenPrefix followed by the base64url
// encoding of accessTokenBytes random bytes. The hash is the SHA-256 of the
// raw (decoded) secret bytes and is all that gets persisted; the plaintext is
// unrecoverable from the hash.
func GenerateAccessToken() (plaintext string, hash []byte, err error) {
	b := make([]byte, accessTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return AccessTokenPrefix + base64.RawURLEncoding.EncodeToString(b), hashAccessTokenSecret(b), nil
}

// HashPresentedAccessToken strips the prefix from a presented token, decodes
// the secret, and returns the SHA-256 hash of the raw bytes, suitable for a
// lookup against the stored hash. ok is false if the value is not a
// well-formed personal access token.
func HashPresentedAccessToken(presented string) (hash []byte, ok bool) {
	presented = strings.TrimSpace(presented)
	secret, found := strings.CutPrefix(presented, AccessTokenPrefix)
	if !found || secret == "" {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil || len(raw) != accessTokenBytes {
		return nil, false
	}
	return hashAccessTokenSecret(raw), true
}

func hashAccessTokenSecret(secret []byte) []byte {
	sum := sha256.Sum256(secret)
	return sum[:]
}

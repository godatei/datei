package link

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/config"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// publicLinkTokenKind identifies a session token issued by the public-link
// unlock endpoint, distinguishing it from regular auth tokens that share the
// same signing secret.
const publicLinkTokenKind = "public_link"

// linkFingerprintClaim binds the issued JWT to the link's key+code at the
// moment of unlock. Any change to either (key rotation, code set/change/clear)
// flips the fingerprint and invalidates stale sessions before `exp`.
const linkFingerprintClaim = "link_fp"

// secret is loaded once from the same auth JWT secret used for owner login;
// the `kind` claim distinguishes the two token populations.
var secret = sync.OnceValue(config.AuthJWTSecret)

// LinkFingerprint returns the SHA-256 fingerprint of a link's secret material
// (key + code), encoded as base64-url. The fingerprint is what's embedded in
// the JWT and compared on every authenticated public-link call. Code is nil
// when the link has no code.
func LinkFingerprint(key string, code *string) string {
	codeBytes := ""
	if code != nil {
		codeBytes = *code
	}
	h := sha256.Sum256([]byte(key + "\x00" + codeBytes))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// signSessionToken builds and signs a public-link session JWT whose subject
// is the link UUID and whose fingerprint claim binds the token to the link's
// current key+code.
func signSessionToken(linkID uuid.UUID, fingerprint string, iat, exp time.Time) (string, error) {
	token, err := jwt.NewBuilder().
		IssuedAt(iat).
		NotBefore(iat).
		Expiration(exp).
		Subject(linkID.String()).
		Claim(authjwt.KindKey, publicLinkTokenKind).
		Claim(linkFingerprintClaim, fingerprint).
		Build()
	if err != nil {
		return "", err
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), secret()))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

// SessionClaims is the verified payload of a public-link JWT.
type SessionClaims struct {
	LinkID      uuid.UUID
	Fingerprint string
}

// ParseSessionToken verifies the signature, expiration, `kind` claim, and
// pulls out the link UUID and the fingerprint the token was issued for.
func ParseSessionToken(tokenString string) (SessionClaims, error) {
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithKey(jwa.HS256(), secret()),
		jwt.WithValidate(true),
	)
	if err != nil {
		return SessionClaims{}, apperrors.ErrLinkUnauthorized
	}
	var kind string
	if err := token.Get(authjwt.KindKey, &kind); err != nil || kind != publicLinkTokenKind {
		return SessionClaims{}, apperrors.ErrLinkUnauthorized
	}
	sub, ok := token.Subject()
	if !ok {
		return SessionClaims{}, apperrors.ErrLinkUnauthorized
	}
	id, err := uuid.Parse(sub)
	if err != nil {
		return SessionClaims{}, fmt.Errorf("invalid link id in token: %w", errors.Join(err, apperrors.ErrLinkUnauthorized))
	}
	var fingerprint string
	if err := token.Get(linkFingerprintClaim, &fingerprint); err != nil || fingerprint == "" {
		return SessionClaims{}, apperrors.ErrLinkUnauthorized
	}
	return SessionClaims{LinkID: id, Fingerprint: fingerprint}, nil
}

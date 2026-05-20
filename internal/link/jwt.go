package link

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// publicLinkTokenKind identifies a session token issued by the public-link
// unlock endpoint, distinguishing it from regular auth tokens that share the
// same signing secret.
const publicLinkTokenKind = "public_link"

// secret is loaded once from the same auth JWT secret used for owner login;
// the `kind` claim distinguishes the two token populations.
var secret = sync.OnceValue(config.AuthJWTSecret)

// signSessionToken builds and signs a public-link session JWT whose subject
// is the link UUID.
func signSessionToken(linkID uuid.UUID, iat, exp time.Time) (string, error) {
	token, err := jwt.NewBuilder().
		IssuedAt(iat).
		NotBefore(iat).
		Expiration(exp).
		Subject(linkID.String()).
		Claim(authjwt.KindKey, publicLinkTokenKind).
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

// ParseSessionToken verifies the signature, expiration, and `kind` claim, and
// returns the link UUID encoded in the subject. Use this from the middleware
// that gates the public list/download endpoints.
func ParseSessionToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithKey(jwa.HS256(), secret()),
		jwt.WithValidate(true),
	)
	if err != nil {
		return uuid.Nil, dateierrors.ErrLinkUnauthorized
	}
	var kind string
	if err := token.Get(authjwt.KindKey, &kind); err != nil || kind != publicLinkTokenKind {
		return uuid.Nil, dateierrors.ErrLinkUnauthorized
	}
	sub, ok := token.Subject()
	if !ok {
		return uuid.Nil, dateierrors.ErrLinkUnauthorized
	}
	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid link id in token: %w", errors.Join(err, dateierrors.ErrLinkUnauthorized))
	}
	return id, nil
}

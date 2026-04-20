package authjwt

import (
	"sync"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/godatei/datei/internal/config"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const (
	UserNameKey          = "name"
	UserEmailKey         = "email"
	UserEmailVerifiedKey = "email_verified"
	PasswordResetKey     = "password_reset"
)

var JWTAuth = sync.OnceValue(func() *jwtauth.JWTAuth {
	return jwtauth.New("HS256", config.AuthJWTSecret(), nil)
})

// GenerateDefaultToken creates a standard login token.
func GenerateDefaultToken(userID uuid.UUID, name, email string, emailVerified bool) (jwt.Token, string, error) {
	now := time.Now()
	claims := map[string]any{
		jwt.IssuedAtKey:      now,
		jwt.NotBeforeKey:     now,
		jwt.ExpirationKey:    now.Add(config.AuthTokenExpiration()),
		jwt.SubjectKey:       userID.String(),
		UserNameKey:          name,
		UserEmailKey:         email,
		UserEmailVerifiedKey: emailVerified,
	}
	return JWTAuth().Encode(claims)
}

// GenerateResetToken creates a short-lived password-reset token.
func GenerateResetToken(userID uuid.UUID, email string) (jwt.Token, string, error) {
	now := time.Now()
	claims := map[string]any{
		jwt.IssuedAtKey:      now,
		jwt.NotBeforeKey:     now,
		jwt.ExpirationKey:    now.Add(config.AuthResetTokenDuration()),
		jwt.SubjectKey:       userID.String(),
		UserEmailKey:         email,
		UserEmailVerifiedKey: true,
		PasswordResetKey:     true,
	}
	return JWTAuth().Encode(claims)
}

// GenerateVerificationToken creates an email-verification token.
func GenerateVerificationToken(userID uuid.UUID, email string) (jwt.Token, string, error) {
	now := time.Now()
	claims := map[string]any{
		jwt.IssuedAtKey:      now,
		jwt.NotBeforeKey:     now,
		jwt.ExpirationKey:    now.Add(config.AuthResetTokenDuration()),
		jwt.SubjectKey:       userID.String(),
		UserEmailKey:         email,
		UserEmailVerifiedKey: true,
	}
	return JWTAuth().Encode(claims)
}

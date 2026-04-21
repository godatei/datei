package authjwt

import (
	"sync"
	"time"

	"github.com/godatei/datei/internal/config"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const (
	UserNameKey          = "name"
	UserEmailKey         = "email"
	UserEmailVerifiedKey = "email_verified"
	ActionKey            = "action"
)

var secret = sync.OnceValue(config.AuthJWTSecret)

func GenerateDefaultToken(userID uuid.UUID, name, email string, emailVerified bool) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		IssuedAt(now).
		NotBefore(now).
		Expiration(now.Add(config.AuthTokenExpiration())).
		Subject(userID.String()).
		Claim(UserNameKey, name).
		Claim(UserEmailKey, email).
		Claim(UserEmailVerifiedKey, emailVerified).
		Build()
	if err != nil {
		return "", err
	}
	return sign(token)
}

func GenerateResetToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		IssuedAt(now).
		NotBefore(now).
		Expiration(now.Add(config.AuthResetTokenDuration())).
		Subject(userID.String()).
		Claim(UserEmailKey, email).
		Claim(UserEmailVerifiedKey, true).
		Claim(ActionKey, string(ActionResetPassword)).
		Build()
	if err != nil {
		return "", err
	}
	return sign(token)
}

func GenerateVerificationToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		IssuedAt(now).
		NotBefore(now).
		Expiration(now.Add(config.AuthResetTokenDuration())).
		Subject(userID.String()).
		Claim(UserEmailKey, email).
		Claim(UserEmailVerifiedKey, true).
		Claim(ActionKey, string(ActionVerifyEmail)).
		Build()
	if err != nil {
		return "", err
	}
	return sign(token)
}

func ParseToken(tokenString string) (jwt.Token, error) {
	return jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.HS256(), secret()), jwt.WithValidate(true))
}

func sign(token jwt.Token) (string, error) {
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), secret()))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

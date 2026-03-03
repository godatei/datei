package authn

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/jwtauth/v5"
	"github.com/godatei/datei/internal/authjwt"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type contextKey struct{}

var ErrNoAuthentication = errors.New("no authentication")

// AuthInfo holds the authenticated user's info extracted from JWT.
type AuthInfo struct {
	UserID        uuid.UUID
	Name          string
	Email         string
	EmailVerified bool
	PasswordReset bool
}

// Middleware validates the JWT Bearer token and injects AuthInfo into context.
func Middleware(next http.Handler) http.Handler {
	verifier := jwtauth.Verifier(authjwt.JWTAuth())
	return verifier(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := jwtauth.FromContext(r.Context())
		if err != nil || token == nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if jwt.Validate(token) != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		info, err := extractAuthInfo(token)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
}

// FromContext retrieves AuthInfo from the request context.
func FromContext(ctx context.Context) (AuthInfo, error) {
	if auth, ok := ctx.Value(contextKey{}).(AuthInfo); ok {
		return auth, nil
	}
	return AuthInfo{}, ErrNoAuthentication
}

// RequireContext panics if no auth info is present (use after Middleware).
func RequireContext(ctx context.Context) AuthInfo {
	auth, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return auth
}

func extractAuthInfo(token jwt.Token) (AuthInfo, error) {
	sub, ok := token.Subject()
	if !ok {
		return AuthInfo{}, errors.New("missing subject claim")
	}
	userID, err := uuid.Parse(sub)
	if err != nil {
		return AuthInfo{}, err
	}

	info := AuthInfo{UserID: userID}

	var name string
	if err := token.Get(authjwt.UserNameKey, &name); err == nil {
		info.Name = name
	}
	var email string
	if err := token.Get(authjwt.UserEmailKey, &email); err == nil {
		info.Email = email
	}
	var verified bool
	if err := token.Get(authjwt.UserEmailVerifiedKey, &verified); err == nil {
		info.EmailVerified = verified
	}
	var reset bool
	if err := token.Get(authjwt.PasswordResetKey, &reset); err == nil {
		info.PasswordReset = reset
	}

	return info, nil
}

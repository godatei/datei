package authn

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
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

// OpenAPIAuthFunc returns an openapi3filter.AuthenticationFunc that validates
// Bearer JWTs and injects AuthInfo into the request context.
// The OapiRequestValidator only calls this for routes with a security requirement;
// routes with `security: []` in the spec are skipped automatically.
func OpenAPIAuthFunc() openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		if input.SecurityScheme.Type != "http" || input.SecurityScheme.Scheme != "Bearer" {
			return fmt.Errorf("unsupported security scheme: %s/%s",
				input.SecurityScheme.Type, input.SecurityScheme.Scheme)
		}

		r := input.RequestValidationInput.Request
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return errors.New("missing Authorization header")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return errors.New("invalid Authorization header format")
		}

		token, err := jwtauth.VerifyToken(authjwt.JWTAuth(), tokenString)
		if err != nil {
			slog.Debug("auth: token verification failed", "path", r.URL.Path, "error", err)
			return fmt.Errorf("invalid token: %w", err)
		}

		if err := jwt.Validate(token); err != nil {
			slog.Debug("auth: token validation failed", "path", r.URL.Path, "error", err)
			return fmt.Errorf("token validation failed: %w", err)
		}

		info, err := extractAuthInfo(token)
		if err != nil {
			slog.Debug("auth: failed to extract claims", "path", r.URL.Path, "error", err)
			return fmt.Errorf("failed to extract claims: %w", err)
		}

		newCtx := context.WithValue(r.Context(), contextKey{}, info)
		*r = *r.WithContext(newCtx) //nolint:contextcheck // must use r.Context(), not func ctx

		return nil
	}
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

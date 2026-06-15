package authn

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/users"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type emailContextKey struct{}

type userContextKey struct{}

var ErrNoAuthentication = errors.New("no authentication")

// EmailIdentity holds session identity details that are not part of the database
// user record — currently just the email tied to the credential (the JWT email
// claim, or the primary email for Basic Auth). The authoritative user record is
// stored separately under userContextKey (see CurrentUser/GetCurrentUser).
type EmailIdentity struct {
	Email string
}

// claims holds the values extracted from a validated identity JWT. It is
// internal to the auth flow; consumers read Identity and the user projection.
type claims struct {
	userID uuid.UUID
	email  string
	action authjwt.Action
}

// OpenAPIAuthFunc returns an openapi3filter.AuthenticationFunc that validates
// Bearer JWTs and injects the Identity and user projection into the request context.
// The OapiRequestValidator only calls this for routes with a security requirement;
// routes with `security: []` in the spec are skipped automatically.
func OpenAPIAuthFunc(userSvc *users.UserService) openapi3filter.AuthenticationFunc {
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

		token, err := authjwt.ParseToken(tokenString)
		if err != nil {
			slog.Debug("auth: token verification failed", "path", r.URL.Path, "error", err)
			return fmt.Errorf("invalid token: %w", err)
		}

		claims, err := extractClaims(token)
		if err != nil {
			slog.Debug("auth: failed to extract claims", "path", r.URL.Path, "error", err)
			return fmt.Errorf("failed to extract claims: %w", err)
		}

		if ext, ok := input.RequestValidationInput.Route.Operation.Extensions["x-required-action"]; ok {
			extStr, ok := ext.(string)
			if !ok {
				return fmt.Errorf("x-required-action extension must be a string")
			}
			required, err := authjwt.ParseAction(extStr)
			if err != nil {
				return fmt.Errorf("invalid x-required-action extension: %w", err)
			}
			if claims.action != required {
				return fmt.Errorf("token action %q not allowed for this endpoint", claims.action)
			}
		} else if claims.action != "" {
			return fmt.Errorf("token action %q not allowed for this endpoint", claims.action)
		}

		//nolint:contextcheck // use r.Context() so the lookup is cancelled on client disconnect
		account, err := userSvc.GetUser(r.Context(), claims.userID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				slog.Debug("auth: user not found", "path", r.URL.Path, "user_id", claims.userID)
				return fmt.Errorf("user not found: %w", err)
			}
			slog.Error("auth: failed to load user", "path", r.URL.Path, "user_id", claims.userID, "error", err)
			return fmt.Errorf("failed to load user: %w", err)
		}
		if account.ArchivedAt != nil {
			slog.Debug("auth: user is archived", "path", r.URL.Path, "user_id", claims.userID)
			return errors.New("user is archived")
		}

		identity := EmailIdentity{Email: claims.email}
		//nolint:contextcheck // must use r.Context(), not func ctx
		*r = *r.WithContext(PopulateContext(r.Context(), identity, account))

		return nil
	}
}

// GetEmailIdentity retrieves the session Identity from the request context.
func GetEmailIdentity(ctx context.Context) (EmailIdentity, error) {
	if id, ok := ctx.Value(emailContextKey{}).(EmailIdentity); ok {
		return id, nil
	}
	return EmailIdentity{}, ErrNoAuthentication
}

// RequireEmailIdentity panics if no Identity is present (use after Middleware).
func RequireEmailIdentity(ctx context.Context) EmailIdentity {
	id, err := GetEmailIdentity(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// GetCurrentUser retrieves the authenticated user's projection from ctx.
func GetCurrentUser(ctx context.Context) (db.UserAccountProjection, error) {
	if user, ok := ctx.Value(userContextKey{}).(db.UserAccountProjection); ok {
		return user, nil
	}
	return db.UserAccountProjection{}, ErrNoAuthentication
}

// RequireCurrentUser panics if no user projection is present (use after Middleware).
func RequireCurrentUser(ctx context.Context) db.UserAccountProjection {
	user, err := GetCurrentUser(ctx)
	if err != nil {
		panic(err)
	}
	return user
}

// PopulateContext injects the EmailIdentity and database-backed user projection into ctx.
func PopulateContext(ctx context.Context, identity EmailIdentity, user db.UserAccountProjection) context.Context {
	ctx = context.WithValue(ctx, emailContextKey{}, identity)
	ctx = context.WithValue(ctx, userContextKey{}, user)
	return ctx
}

func extractClaims(token jwt.Token) (claims, error) {
	// Require the `kind` claim to be present and equal to KindUser. This is
	// what stops a public-link session token (signed with the same secret) from
	// being accepted here as an owner-auth token.
	var kind string
	if err := token.Get(authjwt.KindKey, &kind); err != nil {
		return claims{}, errors.New("missing kind claim")
	}
	if kind != authjwt.KindUser {
		return claims{}, fmt.Errorf("token kind %q not allowed", kind)
	}

	var c claims
	_ = token.Get(authjwt.UserEmailKey, &c.email)

	if sub, ok := token.Subject(); !ok {
		return claims{}, errors.New("missing subject claim")
	} else if userID, err := uuid.Parse(sub); err != nil {
		return claims{}, err
	} else {
		c.userID = userID
	}

	var actionStr string
	if err := token.Get(authjwt.ActionKey, &actionStr); err == nil {
		if action, err := authjwt.ParseAction(actionStr); err != nil {
			return claims{}, err
		} else {
			c.action = action
		}
	}

	return c, nil
}

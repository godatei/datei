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
	"github.com/godatei/datei/internal/httpauth"
	"github.com/godatei/datei/internal/security"
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
	// ctx is the request's own context: the validator threads r.Context() through
	// unchanged, so it carries client-disconnect cancellation.
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

		tokenString, ok := httpauth.ParseBearer(authHeader)
		if !ok {
			return errors.New("invalid Authorization header format")
		}

		action, err := requiredAction(input)
		if err != nil {
			return err
		}

		var (
			account  *users.UserAccount
			identity *EmailIdentity
		)
		// Personal access tokens are prefixed and carry no JWT claims; route them
		// to the token auth path instead of JWT verification.
		if strings.HasPrefix(tokenString, security.AccessTokenPrefix) {
			account, identity, err = authenticateAccessToken(ctx, userSvc, tokenString, action)
		} else {
			account, identity, err = authenticateJWT(ctx, userSvc, tokenString, action)
		}
		if err != nil {
			return err
		}

		*r = *r.WithContext(PopulateContext(ctx, *identity, *account))
		return nil
	}
}

// requiredAction reads the x-required-action OpenAPI extension for the route,
// returning the empty action when the endpoint is not action-scoped.
func requiredAction(input *openapi3filter.AuthenticationInput) (authjwt.Action, error) {
	ext, ok := input.RequestValidationInput.Route.Operation.Extensions["x-required-action"]
	if !ok {
		return "", nil
	}
	extStr, ok := ext.(string)
	if !ok {
		return "", errors.New("x-required-action extension must be a string")
	}
	action, err := authjwt.ParseAction(extStr)
	if err != nil {
		return "", fmt.Errorf("invalid x-required-action extension: %w", err)
	}
	return action, nil
}

// authenticateJWT validates a Bearer identity JWT against the required action
// and resolves the owning account and identity.
func authenticateJWT(
	ctx context.Context,
	userSvc *users.UserService,
	tokenString string,
	action authjwt.Action,
) (*users.UserAccount, *EmailIdentity, error) {
	token, err := authjwt.ParseToken(tokenString)
	if err != nil {
		slog.Debug("auth: token verification failed", "error", err)
		return nil, nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, err := extractClaims(token)
	if err != nil {
		slog.Debug("auth: failed to extract claims", "error", err)
		return nil, nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	if claims.action != action {
		return nil, nil, fmt.Errorf("token action %q not allowed for this endpoint", claims.action)
	}

	account, err := userSvc.GetUser(ctx, claims.userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			slog.Debug("auth: user not found", "user_id", claims.userID)
			return nil, nil, fmt.Errorf("user not found: %w", err)
		}
		slog.Error("auth: failed to load user", "user_id", claims.userID, "error", err)
		return nil, nil, fmt.Errorf("failed to load user: %w", err)
	}
	if account.Archived {
		slog.Debug("auth: user is archived", "user_id", claims.userID)
		return nil, nil, errors.New("user is archived")
	}

	return &account, &EmailIdentity{Email: claims.email}, nil
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

// authenticateAccessToken validates a personal access token presented as a
// Bearer credential and resolves the owning account and identity. Access tokens
// grant full account access but cannot satisfy action-scoped endpoints (e.g.
// email verification), which require a purpose-built JWT.
func authenticateAccessToken(
	ctx context.Context, userSvc *users.UserService, tokenString string, action authjwt.Action,
) (*users.UserAccount, *EmailIdentity, error) {
	if action != "" {
		return nil, nil, errors.New("access tokens cannot be used for action-scoped endpoints")
	}

	result, err := userSvc.AuthenticateAccessToken(ctx, tokenString)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) {
			slog.Debug("auth: invalid access token")
			return nil, nil, fmt.Errorf("invalid access token: %w", err)
		}
		slog.Error("auth: failed to authenticate access token", "error", err)
		return nil, nil, fmt.Errorf("failed to authenticate access token: %w", err)
	}

	return &result.UserAccount, &EmailIdentity{Email: result.Email}, nil
}

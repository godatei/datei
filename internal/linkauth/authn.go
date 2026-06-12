package linkauth

import (
	"context"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/link"
)

// SecuritySchemeName is the OpenAPI security scheme name used to gate the
// public list/download endpoints. The dispatcher in serve.go routes this
// scheme to the public-link JWT verifier instead of the owner-auth one.
const SecuritySchemeName = "publicLinkBearerAuthentication"

type publicLinkContextKey struct{}

// PublicLinkSessionFromContext returns the public-link session claims attached
// to this request by the auth middleware.
func PublicLinkSessionFromContext(ctx context.Context) (link.SessionClaims, bool) {
	s, ok := ctx.Value(publicLinkContextKey{}).(link.SessionClaims)
	return s, ok
}

// RequirePublicLinkSessionFromContext panics if no public-link session is
// present in ctx. Use this after the public-link auth middleware has run.
func RequirePublicLinkSessionFromContext(ctx context.Context) link.SessionClaims {
	s, ok := PublicLinkSessionFromContext(ctx)
	if !ok {
		panic("no public link session in context")
	}
	return s
}

// OpenAPIAuthFunc returns an openapi3filter.AuthenticationFunc that validates
// the public-link session JWT and attaches the parsed claims to the request
// context. The dispatcher in serve.go routes the SecuritySchemeName above to
// this function.
func OpenAPIAuthFunc() openapi3filter.AuthenticationFunc {
	return func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
		r := input.RequestValidationInput.Request
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return fmt.Errorf("missing Authorization header: %w", dateierrors.ErrLinkUnauthorized)
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return fmt.Errorf("invalid Authorization header format: %w", dateierrors.ErrLinkUnauthorized)
		}
		claims, err := link.ParseSessionToken(tokenString)
		if err != nil {
			return err
		}
		ctx := context.WithValue(r.Context(), publicLinkContextKey{}, claims)
		*r = *r.WithContext(ctx) //nolint:contextcheck // must use r.Context(), not func ctx
		return nil
	}
}

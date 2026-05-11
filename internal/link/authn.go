package link

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/google/uuid"
)

// SecuritySchemeName is the OpenAPI security scheme name used to gate the
// public list/download endpoints. The dispatcher in serve.go routes this
// scheme to the public-link JWT verifier instead of the owner-auth one.
const SecuritySchemeName = "publicLinkBearerAuthentication"

type publicLinkContextKey struct{}

// LinkIDFromContext returns the link UUID bound to the public-link session
// token attached to this request.
func LinkIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(publicLinkContextKey{}).(uuid.UUID)
	return id, ok
}

// RequireLinkIDFromContext panics if no link session is present in ctx. Use
// this after the public-link auth middleware has run.
func RequireLinkIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := LinkIDFromContext(ctx)
	if !ok {
		panic("no public link session in context")
	}
	return id
}

// OpenAPIAuthFunc returns an openapi3filter.AuthenticationFunc that validates
// the public-link session JWT and attaches the link UUID to the request
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
		linkID, err := ParseSessionToken(tokenString)
		if err != nil {
			return errors.Join(err, dateierrors.ErrLinkUnauthorized)
		}
		ctx := context.WithValue(r.Context(), publicLinkContextKey{}, linkID)
		*r = *r.WithContext(ctx) //nolint:contextcheck // must use r.Context(), not func ctx
		return nil
	}
}

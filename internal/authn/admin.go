package authn

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
)

// RequireAdmin returns the current AuthInfo if the caller is authenticated AND
// has is_admin=true in the projection. The flag is verified against the
// database (not the JWT) so demotion takes effect on the next call.
// Returns dateierrors.ErrForbidden if the caller is not an admin.
func RequireAdmin(ctx context.Context, q *db.Queries) (AuthInfo, error) {
	info, err := FromContext(ctx)
	if err != nil {
		return AuthInfo{}, err
	}
	user, err := q.GetUserAccountByID(ctx, info.UserID)
	if err != nil {
		return AuthInfo{}, fmt.Errorf("failed to load user for admin check: %w", err)
	}
	if !user.IsAdmin {
		return AuthInfo{}, dateierrors.ErrForbidden
	}
	info.IsAdmin = true
	return info, nil
}

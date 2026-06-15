package authn

import (
	"context"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/db"
)

// RequireAdmin returns the current user record if the caller is authenticated
// AND is_admin=true. Returns apperrors.ErrForbidden if not.
//
// The admin flag is read from the database-backed account loaded by the auth
// middleware, so demotion takes effect on the next request.
func RequireAdmin(ctx context.Context) (db.UserAccountProjection, error) {
	if user, err := GetCurrentUser(ctx); err != nil {
		return db.UserAccountProjection{}, err
	} else if !user.IsAdmin {
		return db.UserAccountProjection{}, apperrors.ErrForbidden
	} else {
		return user, nil
	}
}

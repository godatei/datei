package authn

import (
	"context"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/users"
)

// RequireAdmin returns the current user record if the caller is authenticated
// AND is_admin=true. Returns apperrors.ErrForbidden if not.
//
// The admin flag is read from the database-backed account loaded by the auth
// middleware, so demotion takes effect on the next request.
func RequireAdmin(ctx context.Context) (users.UserAccount, error) {
	if user, err := GetCurrentUser(ctx); err != nil {
		return users.UserAccount{}, err
	} else if !user.IsAdmin {
		return users.UserAccount{}, apperrors.ErrForbidden
	} else {
		return user, nil
	}
}

package authn

import (
	"context"

	"github.com/godatei/datei/internal/dateierrors"
)

// RequireAdmin returns the current AuthInfo if the caller is authenticated AND
// has is_admin=true on the JWT. Returns dateierrors.ErrForbidden if not.
//
// The check trusts the JWT claim; demotion takes effect on the user's next
// token refresh. If we ever need real-time revocation, route this through a
// DB lookup instead.
func RequireAdmin(ctx context.Context) (AuthInfo, error) {
	info, err := FromContext(ctx)
	if err != nil {
		return AuthInfo{}, err
	}
	if !info.IsAdmin {
		return AuthInfo{}, dateierrors.ErrForbidden
	}
	return info, nil
}

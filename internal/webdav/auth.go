package webdav

import (
	"net/http"

	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/users"
)

// BasicAuthMiddleware validates HTTP Basic Auth credentials against the user
// service and injects the resulting AuthInfo into the request context.
// Accounts with MFA enabled cannot use WebDAV.
func BasicAuthMiddleware(userSvc *users.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Datei WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			out, err := userSvc.ValidateCredentials(r.Context(), email, password)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="Datei WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if out.RequiresMFA {
				http.Error(w, "MFA-protected accounts cannot use WebDAV", http.StatusForbidden)
				return
			}

			info := authn.AuthInfo{
				UserID:        out.UserID,
				Name:          out.Name,
				Email:         out.Email,
				EmailVerified: out.EmailVerified,
			}
			r = r.WithContext(authn.NewContext(r.Context(), info))
			next.ServeHTTP(w, r)
		})
	}
}

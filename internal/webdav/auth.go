package webdav

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/users"
)

// BasicAuthMiddleware authenticates WebDAV requests with a personal access
// token and injects the resulting Identity and user projection into the request
// context. WebDAV only supports access tokens (never passwords): the token may
// be supplied either as a Bearer credential or as the password in HTTP Basic
// auth (the Basic username is ignored, since proxies may log it). Failed
// attempts are rate-limited per IP (20 failures/minute).
func BasicAuthMiddleware(userSvc *users.UserService) func(http.Handler) http.Handler {
	failLimiter := httprate.NewRateLimiter(20, 1*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractToken(r)
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Datei WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ip := middleware.GetClientIP(r.Context())
			if allowed, _, _ := failLimiter.Status(ip); !allowed {
				if limited := failLimiter.RespondOnLimit(w, r, ip); limited {
					return
				}
			}

			out, err := userSvc.AuthenticateAccessToken(r.Context(), token)
			if err != nil {
				switch {
				case errors.Is(err, apperrors.ErrInvalidCredentials):
					if limited := failLimiter.RespondOnLimit(w, r, ip); !limited {
						w.Header().Set("WWW-Authenticate", `Basic realm="Datei WebDAV"`)
						http.Error(w, "Unauthorized", http.StatusUnauthorized)
					}
				default:
					slog.ErrorContext(r.Context(), "failed to validate webdav access token", "error", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}

			identity := authn.EmailIdentity{Email: out.Email}
			r = r.WithContext(authn.PopulateContext(r.Context(), identity, out.UserAccount))
			next.ServeHTTP(w, r)
		})
	}
}

// extractToken pulls a personal access token from the request, preferring a
// Bearer credential and falling back to the password field of HTTP Basic auth.
func extractToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if token, found := strings.CutPrefix(authHeader, "Bearer "); found && token != "" {
		return token, true
	}
	if _, password, ok := r.BasicAuth(); ok && password != "" {
		return password, true
	}
	return "", false
}

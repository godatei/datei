package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
)

// AuthRoutes returns a chi.Router for /api/v1/auth/* (public, rate-limited)
func AuthRoutes(pool *pgxpool.Pool, userRepo aggregate.UserRepository, m mailer.Mailer) chi.Router {
	r := chi.NewRouter()
	r.Use(httprate.Limit(
		10, 1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByRealIP, httprate.KeyByEndpoint),
	))
	r.Post("/login", loginHandler(pool, userRepo))
	r.Get("/login/config", loginConfigHandler())
	r.Post("/register", registerHandler(pool, userRepo, m))
	r.Post("/reset", resetPasswordHandler(pool, m))
	return r
}

func loginHandler(pool *pgxpool.Pool, userRepo aggregate.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		req, err := jsonBody[LoginRequest](w, r)
		if err != nil {
			return
		}

		q := db.New(pool)
		user, err := q.GetUserAccountByEmail(ctx, req.Email)
		if err != nil {
			http.Error(w, "invalid email or password", http.StatusBadRequest)
			return
		}

		if err := security.VerifyPassword(req.Password, user.PasswordHash, user.PasswordSalt); err != nil {
			http.Error(w, "invalid email or password", http.StatusBadRequest)
			return
		}

		if user.MfaEnabled {
			if req.MFACode == nil {
				respondJSON(w, LoginResponse{RequiresMFA: true})
				return
			}

			if user.MfaSecret == nil {
				http.Error(w, "MFA configuration error", http.StatusInternalServerError)
				return
			}

			valid := totp.Validate(*req.MFACode, *user.MfaSecret)

			if !valid {
				normalized := security.NormalizeRecoveryCode(*req.MFACode)
				codes, err := q.GetUnusedMFARecoveryCodes(ctx, user.ID)
				if err != nil {
					slog.Error("failed to get recovery codes", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				var matchedCodeID *uuid.UUID
				for _, code := range codes {
					if security.VerifyRecoveryCode(normalized, code.CodeHash, code.CodeSalt) {
						matchedCodeID = &code.ID
						break
					}
				}

				if matchedCodeID == nil {
					http.Error(w, "invalid MFA code or recovery code", http.StatusUnauthorized)
					return
				}

				agg, err := userRepo.LoadByID(ctx, user.ID)
				if err != nil {
					slog.Error("failed to load user aggregate", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				if err := agg.UseRecoveryCode(*matchedCodeID, time.Now()); err != nil {
					slog.Error("failed to use recovery code", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				if err := userRepo.Save(ctx, agg); err != nil {
					slog.Error("failed to save user aggregate", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
			}
		}

		primaryEmail, err := q.GetPrimaryEmailForUser(ctx, user.ID)
		if err != nil {
			slog.Error("failed to get primary email", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		emailVerified := !config.AuthEmailVerificationRequired() || primaryEmail.VerifiedAt != nil

		_, tokenString, err := authjwt.GenerateDefaultToken(user.ID, user.Name, primaryEmail.Email, emailVerified)
		if err != nil {
			slog.Error("failed to generate token", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		agg, err := userRepo.LoadByID(ctx, user.ID)
		if err != nil {
			slog.Error("failed to load user for login tracking", "error", err)
		} else {
			if err := agg.RecordLogin(time.Now()); err == nil {
				if err := userRepo.Save(ctx, agg); err != nil {
					slog.Error("failed to save login event", "error", err)
				}
			}
		}

		respondJSON(w, LoginResponse{Token: tokenString})
	}
}

func loginConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, LoginConfigResponse{
			RegistrationEnabled: config.AuthRegistrationEnabled(),
		})
	}
}

func registerHandler(pool *pgxpool.Pool, userRepo aggregate.UserRepository, m mailer.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !config.AuthRegistrationEnabled() {
			http.Error(w, "registration is disabled", http.StatusForbidden)
			return
		}

		req, err := jsonBody[RegisterRequest](w, r)
		if err != nil {
			return
		}

		if req.Email == "" || req.Password == "" || req.Name == "" {
			http.Error(w, "email, name, and password are required", http.StatusBadRequest)
			return
		}
		if len(req.Password) < 8 {
			http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		q := db.New(pool)
		if _, err := q.GetUserAccountByEmail(ctx, req.Email); err == nil {
			http.Error(w, "email already registered", http.StatusBadRequest)
			return
		}

		passwordHash, passwordSalt, err := security.HashPassword(req.Password)
		if err != nil {
			slog.Error("failed to hash password", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		userID := uuid.New()
		emailID := uuid.New()
		agg := &aggregate.UserAggregate{}
		if err := agg.Register(userID, req.Name, req.Email, emailID, passwordHash, passwordSalt, time.Now()); err != nil {
			slog.Error("failed to create user aggregate", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to save user", "error", err)
			http.Error(w, "registration failed", http.StatusInternalServerError)
			return
		}

		if config.AuthEmailVerificationRequired() {
			_, token, err := authjwt.GenerateVerificationToken(userID, req.Email)
			if err != nil {
				slog.Error("failed to generate verification token", "error", err)
			} else {
				verifyURL := fmt.Sprintf("%s/verify?jwt=%s", config.ServerHost(), token)
				if err := m.Send(ctx, mailer.EmailVerificationEmail(req.Email, verifyURL)); err != nil {
					slog.Warn("failed to send verification email", "error", err)
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func resetPasswordHandler(pool *pgxpool.Pool, m mailer.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		req, err := jsonBody[ResetPasswordRequest](w, r)
		if err != nil {
			return
		}

		q := db.New(pool)
		user, err := q.GetUserAccountByEmail(ctx, req.Email)
		if err != nil {
			// Don't reveal whether email exists
			w.WriteHeader(http.StatusNoContent)
			return
		}

		primaryEmail, err := q.GetPrimaryEmailForUser(ctx, user.ID)
		if err != nil {
			slog.Error("failed to get primary email", "error", err)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		_, token, err := authjwt.GenerateResetToken(user.ID, primaryEmail.Email)
		if err != nil {
			slog.Error("failed to generate reset token", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		resetURL := fmt.Sprintf("%s/reset?jwt=%s", config.ServerHost(), token)
		if err := m.Send(ctx, mailer.PasswordResetEmail(primaryEmail.Email, resetURL)); err != nil {
			slog.Warn("failed to send reset email", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

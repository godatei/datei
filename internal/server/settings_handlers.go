package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// SettingsRoutes returns a chi.Router for /api/v1/settings/* (auth required)
func SettingsRoutes(pool *pgxpool.Pool, userRepo aggregate.UserRepository, m mailer.Mailer) chi.Router {
	r := chi.NewRouter()
	r.Use(authn.Middleware)

	r.Post("/user", updateUserHandler(userRepo))
	r.Patch("/user/email", updateUserEmailHandler(userRepo, m))
	r.Post("/verify/request", verifyRequestHandler(pool, m))
	r.Post("/verify/confirm", verifyConfirmHandler(userRepo))
	r.Post("/mfa/setup", mfaSetupHandler(pool, userRepo))
	r.Post("/mfa/enable", mfaEnableHandler(pool, userRepo))
	r.Post("/mfa/disable", mfaDisableHandler(pool, userRepo))
	r.Post("/mfa/recovery-codes/regenerate", mfaRegenerateHandler(pool, userRepo))
	r.Get("/mfa/recovery-codes/status", mfaRecoveryStatusHandler(pool))

	return r
}

func updateUserHandler(userRepo aggregate.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		req, err := jsonBody[UpdateUserRequest](w, r)
		if err != nil {
			return
		}

		agg, err := userRepo.LoadByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to load user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		now := time.Now()
		if req.Name != nil {
			if err := agg.ChangeName(*req.Name, now); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if req.Password != nil {
			if len(*req.Password) < 8 {
				http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
				return
			}
			hash, salt, err := security.HashPassword(*req.Password)
			if err != nil {
				slog.Error("failed to hash password", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if err := agg.ChangePassword(hash, salt, now); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to save user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func updateUserEmailHandler(userRepo aggregate.UserRepository, m mailer.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		req, err := jsonBody[UpdateUserEmailRequest](w, r)
		if err != nil {
			return
		}

		agg, err := userRepo.LoadByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to load user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := agg.ChangeEmail(agg.Email, req.Email, time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to save user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if config.AuthEmailVerificationRequired() {
			sendVerificationEmail(ctx, m, authInfo.UserID, req.Email)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func verifyRequestHandler(pool *pgxpool.Pool, m mailer.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		q := db.New(pool)
		email, err := q.GetPrimaryEmailForUser(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to get email", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		sendVerificationEmail(ctx, m, authInfo.UserID, email.Email)
		w.WriteHeader(http.StatusNoContent)
	}
}

func verifyConfirmHandler(userRepo aggregate.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		agg, err := userRepo.LoadByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to load user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := agg.VerifyEmail(time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to save user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func mfaSetupHandler(pool *pgxpool.Pool, userRepo aggregate.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		q := db.New(pool)
		user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to get user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if user.MfaEnabled {
			http.Error(w, "MFA is already enabled", http.StatusBadRequest)
			return
		}

		email, err := q.GetPrimaryEmailForUser(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to get email", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Datei",
			AccountName: email.Email,
			Algorithm:   otp.AlgorithmSHA1,
			Digits:      otp.DigitsSix,
			Period:      30,
		})
		if err != nil {
			slog.Error("failed to generate TOTP key", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		agg, err := userRepo.LoadByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to load user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := agg.InitiateMFASetup(key.Secret(), time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to save MFA secret", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		img, err := key.Image(200, 200)
		if err != nil {
			slog.Error("failed to generate QR code", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			slog.Error("failed to encode QR code", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
		respondJSON(w, SetupMFAResponse{
			Secret:    key.Secret(),
			QRCodeURL: qrCode,
		})
	}
}

func mfaEnableHandler(pool *pgxpool.Pool, userRepo aggregate.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		req, err := jsonBody[EnableMFARequest](w, r)
		if err != nil {
			return
		}

		q := db.New(pool)
		user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to get user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if user.MfaEnabled {
			http.Error(w, "MFA is already enabled", http.StatusBadRequest)
			return
		}
		if user.MfaSecret == nil {
			http.Error(w, "MFA not set up", http.StatusBadRequest)
			return
		}
		if !totp.Validate(req.Code, *user.MfaSecret) {
			http.Error(w, "invalid code", http.StatusBadRequest)
			return
		}

		codes, err := security.GenerateRecoveryCodes()
		if err != nil {
			slog.Error("failed to generate recovery codes", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		hashedCodes := make([]events.HashedRecoveryCode, len(codes))
		for i, code := range codes {
			hash, salt, err := security.HashRecoveryCode(code)
			if err != nil {
				slog.Error("failed to hash recovery code", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			hashedCodes[i] = events.HashedRecoveryCode{
				ID:       uuid.New(),
				CodeHash: hash,
				CodeSalt: salt,
			}
		}

		agg, err := userRepo.LoadByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to load user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := agg.EnableMFA(hashedCodes, time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to enable MFA", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		formattedCodes := make([]string, len(codes))
		for i, code := range codes {
			formattedCodes[i] = security.FormatRecoveryCode(code)
		}
		respondJSON(w, EnableMFAResponse{RecoveryCodes: formattedCodes})
	}
}

func mfaDisableHandler(pool *pgxpool.Pool, userRepo aggregate.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		req, err := jsonBody[DisableMFARequest](w, r)
		if err != nil {
			return
		}

		q := db.New(pool)
		user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to get user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !user.MfaEnabled {
			http.Error(w, "MFA is not enabled", http.StatusBadRequest)
			return
		}
		if err := security.VerifyPassword(req.Password, user.PasswordHash, user.PasswordSalt); err != nil {
			http.Error(w, "invalid password", http.StatusUnauthorized)
			return
		}

		agg, err := userRepo.LoadByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to load user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := agg.DisableMFA(time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to disable MFA", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func mfaRegenerateHandler(pool *pgxpool.Pool, userRepo aggregate.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		req, err := jsonBody[RegenerateMFARecoveryCodesRequest](w, r)
		if err != nil {
			return
		}

		q := db.New(pool)
		user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to get user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !user.MfaEnabled {
			http.Error(w, "MFA is not enabled", http.StatusBadRequest)
			return
		}
		if err := security.VerifyPassword(req.Password, user.PasswordHash, user.PasswordSalt); err != nil {
			http.Error(w, "invalid password", http.StatusUnauthorized)
			return
		}

		codes, err := security.GenerateRecoveryCodes()
		if err != nil {
			slog.Error("failed to generate recovery codes", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		hashedCodes := make([]events.HashedRecoveryCode, len(codes))
		for i, code := range codes {
			hash, salt, err := security.HashRecoveryCode(code)
			if err != nil {
				slog.Error("failed to hash recovery code", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			hashedCodes[i] = events.HashedRecoveryCode{
				ID:       uuid.New(),
				CodeHash: hash,
				CodeSalt: salt,
			}
		}

		agg, err := userRepo.LoadByID(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to load user", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := agg.RegenerateRecoveryCodes(hashedCodes, time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := userRepo.Save(ctx, agg); err != nil {
			slog.Error("failed to regenerate recovery codes", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		formattedCodes := make([]string, len(codes))
		for i, code := range codes {
			formattedCodes[i] = security.FormatRecoveryCode(code)
		}
		respondJSON(w, RegenerateMFARecoveryCodesResponse{RecoveryCodes: formattedCodes})
	}
}

func mfaRecoveryStatusHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authInfo := authn.RequireContext(ctx)

		q := db.New(pool)
		count, err := q.CountUnusedMFARecoveryCodes(ctx, authInfo.UserID)
		if err != nil {
			slog.Error("failed to count recovery codes", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		respondJSON(w, MFARecoveryCodesStatusResponse{RemainingCodes: count})
	}
}

func sendVerificationEmail(ctx context.Context, m mailer.Mailer, userID uuid.UUID, email string) {
	_, token, err := authjwt.GenerateVerificationToken(userID, email)
	if err != nil {
		slog.Error("failed to generate verification token", "error", err)
		return
	}
	verifyURL := fmt.Sprintf("%s/verify?jwt=%s", config.ServerHost(), token)
	if err := m.Send(ctx, mailer.EmailVerificationEmail(email, verifyURL)); err != nil {
		slog.Warn("failed to send verification email", "error", err)
	}
}

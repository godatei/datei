package users

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mailer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	db         *pgxpool.Pool
	repository Repository
	mailer     mailer.Mailer
}

func NewUserService(
	db *pgxpool.Pool,
	repository Repository,
	m mailer.Mailer,
) *UserService {
	return &UserService{
		db:         db,
		repository: repository,
		mailer:     m,
	}
}

func (s *UserService) queries() *db.Queries {
	return db.New(s.db)
}

// GetUser returns the current user profile from the projection.
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (db.UserAccountProjection, error) {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, userID)
	if err != nil {
		return db.UserAccountProjection{}, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *UserService) sendVerificationEmail(ctx context.Context, userID uuid.UUID, email string) {
	token, err := authjwt.GenerateVerificationToken(userID, email)
	if err != nil {
		slog.Error("failed to generate verification token", "error", err)
		return
	}
	verifyURL := fmt.Sprintf("%s/verify?jwt=%s", config.ServerHost(), token)
	if err := s.mailer.Send(ctx, mailer.EmailVerificationEmail(email, verifyURL)); err != nil {
		slog.Warn("failed to send verification email", "error", err)
	}
}

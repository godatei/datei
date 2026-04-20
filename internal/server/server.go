package server

import (
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/ocr"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

const fileFormField = "file"

type server struct {
	dateiService *datei.Service
	userService  *users.UserService
}

func NewServer(
	pool *pgxpool.Pool,
	store storage.Store,
	dateiRepo datei.Repository,
	userRepo users.Repository,
	m mailer.Mailer,
	ocrClient *ocr.Client,
) *server {
	return &server{
		dateiService: datei.NewService(pool, store, dateiRepo, ocrClient),
		userService:  users.NewUserService(pool, userRepo, m),
	}
}

var _ StrictServerInterface = (*server)(nil)

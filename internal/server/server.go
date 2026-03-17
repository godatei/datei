package server

import (
	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

const fileFormField = "file"

type server struct {
	dateiService *datei.DateiService
	userService  *users.UserService
}

func NewServer(
	pool *pgxpool.Pool,
	store storage.Store,
	dateiRepo aggregate.DateiRepository,
	userRepo aggregate.UserRepository,
	m mailer.Mailer,
) *server {
	return &server{
		dateiService: datei.NewDateiService(pool, store, dateiRepo),
		userService:  users.NewUserService(pool, userRepo, m),
	}
}

var _ StrictServerInterface = (*server)(nil)

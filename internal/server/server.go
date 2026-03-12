package server

import (
	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const fileFormField = "file"

type server struct {
	pool         *pgxpool.Pool
	dateiService *datei.DateiService
	userRepo     aggregate.UserRepository
	mailer       mailer.Mailer
}

func NewServer(
	pool *pgxpool.Pool,
	store storage.Store,
	dateiRepo aggregate.DateiRepository,
	userRepo aggregate.UserRepository,
	m mailer.Mailer,
) *server {
	return &server{
		pool:         pool,
		dateiService: datei.NewDateiService(pool, store, dateiRepo),
		userRepo:     userRepo,
		mailer:       m,
	}
}

var _ StrictServerInterface = (*server)(nil)

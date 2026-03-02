package server

import (
	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const fileFormField = "file"

type server struct {
	dateiService *datei.DateiService
}

func NewServer(db *pgxpool.Pool, store storage.Store, repository aggregate.DateiRepository) *server {
	return &server{
		dateiService: datei.NewDateiService(db, store, repository),
	}
}

var _ StrictServerInterface = (*server)(nil)

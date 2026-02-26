package server

import (
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const fileFormField = "file"

type server struct {
	dateiService *datei.DateiService
}

func NewServer(db *pgxpool.Pool, store storage.Store) *server {
	return &server{
		dateiService: datei.NewDateiService(db, store),
	}
}

var _ StrictServerInterface = (*server)(nil)

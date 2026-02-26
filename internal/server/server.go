package server

import (
	"github.com/godatei/datei/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const fileFormField = "file"

type server struct {
	db    *pgxpool.Pool
	store storage.Store
}

func NewServer(db *pgxpool.Pool, store storage.Store) *server {
	return &server{db: db, store: store}
}

var _ StrictServerInterface = (*server)(nil)

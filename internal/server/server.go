package server

import (
	"fmt"
	"io"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/ocr"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	fileFormField           = "file"
	nameFormField           = "name"
	parentIdFormField       = "parentId"
	updateParentIdFormField = "updateParentId"
)

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

// readLimited reads at most maxBytes from r. If the input exceeds maxBytes, an error is returned.
func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	buf, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > maxBytes {
		return nil, fmt.Errorf("input exceeds %d bytes", maxBytes)
	}
	return buf, nil
}

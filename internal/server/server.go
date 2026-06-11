package server

import (
	"fmt"
	"io"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/link"
	"github.com/godatei/datei/internal/users"
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
	linkService  *link.Service
}

func NewServer(
	dateiSvc *datei.Service,
	userSvc *users.UserService,
	linkSvc *link.Service,
) *server {
	return &server{
		dateiService: dateiSvc,
		userService:  userSvc,
		linkService:  linkSvc,
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

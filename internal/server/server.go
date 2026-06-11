package server

import (
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
	dateiServer
	trashServer
	authServer
	settingsServer
	emailsServer
	adminUsersServer
	linkServer
	publicLinkServer
}

func NewServer(
	dateiSvc *datei.Service,
	userSvc *users.UserService,
	linkSvc *link.Service,
	publicLinkSvc *link.PublicService,
) *server {
	return &server{
		dateiServer:      dateiServer{svc: dateiSvc},
		trashServer:      trashServer{svc: dateiSvc},
		authServer:       authServer{svc: userSvc},
		settingsServer:   settingsServer{svc: userSvc},
		emailsServer:     emailsServer{svc: userSvc},
		adminUsersServer: adminUsersServer{svc: userSvc},
		linkServer:       linkServer{svc: linkSvc},
		publicLinkServer: publicLinkServer{svc: publicLinkSvc},
	}
}

var _ StrictServerInterface = (*server)(nil)

package server

import (
	"github.com/godatei/datei/internal/file"
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
	fileServer
	trashServer
	authServer
	settingsServer
	emailsServer
	adminUsersServer
	linkServer
	publicLinkServer
}

func NewServer(
	fileSvc *file.Service,
	userSvc *users.UserService,
	linkSvc *link.Service,
	publicLinkSvc *link.PublicService,
) *server {
	return &server{
		fileServer:       fileServer{svc: fileSvc},
		trashServer:      trashServer{svc: fileSvc},
		authServer:       authServer{svc: userSvc},
		settingsServer:   settingsServer{svc: userSvc},
		emailsServer:     emailsServer{svc: userSvc},
		adminUsersServer: adminUsersServer{svc: userSvc},
		linkServer:       linkServer{svc: linkSvc},
		publicLinkServer: publicLinkServer{svc: publicLinkSvc},
	}
}

var _ StrictServerInterface = (*server)(nil)

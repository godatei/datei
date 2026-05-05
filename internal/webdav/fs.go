package webdav

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	xdav "golang.org/x/net/webdav"
)

type dateiFS struct {
	service *datei.Service
	queries *db.Queries
}

// NewHandler returns a webdav.Handler that serves the Datei file system.
// It must be mounted at /dav (or with the same prefix passed here).
func NewHandler(pool *pgxpool.Pool, service *datei.Service) *xdav.Handler {
	return &xdav.Handler{
		FileSystem: &dateiFS{service: service, queries: db.New(pool)},
		LockSystem: xdav.NewMemLS(),
	}
}

// resolve returns the DateiProjection for the given WebDAV path.
// Returns nil, nil for the virtual root ("/").
func (fs *dateiFS) resolve(ctx context.Context, name string) (*db.DateiProjection, error) {
	name = strings.Trim(name, "/")
	if name == "" {
		return nil, nil
	}
	segments := strings.Split(name, "/")
	var parentID *uuid.UUID
	var proj *db.DateiProjection
	for i, seg := range segments {
		child, err := fs.findChild(ctx, parentID, seg)
		if err != nil {
			return nil, err
		}
		proj = child
		if i < len(segments)-1 {
			if !proj.IsDirectory {
				return nil, os.ErrNotExist
			}
			parentID = &proj.ID
		}
	}
	return proj, nil
}

// findChild returns the non-trashed child of parentID with the given name.
func (fs *dateiFS) findChild(ctx context.Context, parentID *uuid.UUID, name string) (*db.DateiProjection, error) {
	var children []db.DateiProjection
	var err error
	if parentID == nil {
		children, err = fs.queries.ListRootDateiProjections(ctx)
	} else {
		children, err = fs.queries.ListDateiProjectionsByParent(ctx, parentID)
	}
	if err != nil {
		return nil, err
	}
	for i := range children {
		if children[i].Name == name {
			return &children[i], nil
		}
	}
	return nil, os.ErrNotExist
}

// resolveParent returns the parent projection and the base filename for a path.
// parent is nil when the item lives directly under the virtual root.
func (fs *dateiFS) resolveParent(ctx context.Context, name string) (*db.DateiProjection, string, error) {
	dir, base := path.Split(strings.TrimRight(name, "/"))
	dir = strings.TrimRight(dir, "/")
	if dir == "" {
		return nil, base, nil
	}
	parent, err := fs.resolve(ctx, dir)
	if err != nil {
		return nil, "", err
	}
	if parent == nil || !parent.IsDirectory {
		return nil, "", os.ErrInvalid
	}
	return parent, base, nil
}

func (fs *dateiFS) Mkdir(ctx context.Context, name string, _ os.FileMode) error {
	parent, base, err := fs.resolveParent(ctx, name)
	if err != nil {
		return err
	}
	var parentID *uuid.UUID
	if parent != nil {
		parentID = &parent.ID
	}
	if _, err := fs.findChild(ctx, parentID, base); err == nil {
		return os.ErrExist
	}
	_, err = fs.service.CreateDatei(ctx, datei.CreateDateiInput{
		ParentID: parentID,
		FileName: base,
	})
	return err
}

func (fs *dateiFS) RemoveAll(ctx context.Context, name string) error {
	proj, err := fs.resolve(ctx, name)
	if err != nil {
		return err
	}
	if proj == nil {
		return os.ErrPermission
	}
	return fs.service.DeleteDatei(ctx, proj.ID)
}

func (fs *dateiFS) Rename(ctx context.Context, oldName, newName string) error {
	proj, err := fs.resolve(ctx, oldName)
	if err != nil {
		return err
	}
	if proj == nil {
		return os.ErrPermission
	}

	newParent, newBase, err := fs.resolveParent(ctx, newName)
	if err != nil {
		return err
	}
	var newParentID *uuid.UUID
	if newParent != nil {
		newParentID = &newParent.ID
	}

	if _, err := fs.findChild(ctx, newParentID, newBase); err == nil {
		return os.ErrExist
	}

	sameParent := (proj.ParentID == nil && newParentID == nil) ||
		(proj.ParentID != nil && newParentID != nil && *proj.ParentID == *newParentID)

	input := datei.UpdateDateiInput{ID: proj.ID, Name: &newBase}
	if !sameParent {
		input.MoveRequested = true
		input.NewParentID = newParentID
	}
	_, err = fs.service.UpdateDatei(ctx, input)
	return err
}

func (fs *dateiFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name == "/" {
		return rootInfo(), nil
	}
	proj, err := fs.resolve(ctx, name)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return rootInfo(), nil
	}
	return projInfo(proj), nil
}

func (fs *dateiFS) OpenFile(ctx context.Context, name string, flag int, _ os.FileMode) (xdav.File, error) {
	isWrite := flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE) != 0

	if name == "/" {
		children, err := fs.queries.ListRootDateiProjections(ctx)
		if err != nil {
			return nil, err
		}
		return newDirFile(rootInfo(), children), nil
	}

	if !isWrite {
		proj, err := fs.resolve(ctx, name)
		if err != nil {
			return nil, err
		}
		if proj.IsDirectory {
			children, err := fs.queries.ListDateiProjectionsByParent(ctx, &proj.ID)
			if err != nil {
				return nil, err
			}
			return newDirFile(projInfo(proj), children), nil
		}
		out, err := fs.service.DownloadDatei(ctx, proj.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		return newReadFile(projInfo(proj), out)
	}

	parent, base, err := fs.resolveParent(ctx, name)
	if err != nil {
		return nil, err
	}
	var parentID *uuid.UUID
	if parent != nil {
		parentID = &parent.ID
	}

	existing, lookupErr := fs.findChild(ctx, parentID, base)
	var existingID *uuid.UUID
	if lookupErr == nil {
		id := existing.ID
		existingID = &id
	} else if !errors.Is(lookupErr, os.ErrNotExist) {
		return nil, lookupErr
	}

	return newWriteFile(ctx, base, parentID, existingID, fs.service)
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, dateierrors.ErrNotFound):
		return os.ErrNotExist
	case errors.Is(err, dateierrors.ErrIsDirectory):
		return os.ErrInvalid
	case errors.Is(err, dateierrors.ErrParentNotFound):
		return os.ErrNotExist
	case errors.Is(err, dateierrors.ErrCycleDetected):
		return os.ErrInvalid
	default:
		return err
	}
}

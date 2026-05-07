package webdav

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	xdav "golang.org/x/net/webdav"
)

type davCacheKey struct{}

type davPathCache map[string]*api.Datei

func (c davPathCache) store(parentPath string, children []api.Datei) {
	for i := range children {
		child := &children[i]
		if child.Name == nil {
			continue
		}
		var key string
		if parentPath == "" {
			key = *child.Name
		} else {
			key = parentPath + "/" + *child.Name
		}
		c[key] = child
	}
}

func readFromCache(ctx context.Context, name string) *api.Datei {
	if cache := cacheFromContext(ctx); cache != nil {
		if d, ok := cache[name]; ok {
			return d
		}
	}
	return nil
}

func writeToCache(ctx context.Context, parentPath string, children []api.Datei) {
	if cache := cacheFromContext(ctx); cache != nil {
		cache.store(parentPath, children)
	}
}

func cacheFromContext(ctx context.Context) davPathCache {
	c, _ := ctx.Value(davCacheKey{}).(davPathCache)
	return c
}

// CacheMiddleware injects a per-request path cache into the context.
// x/net/webdav calls OpenFile for every child during PROPFIND to fetch dead
// properties; the cache lets resolve() skip the DB for paths already loaded
// by ListDateiChildren.
func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), davCacheKey{}, make(davPathCache))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type dateiFS struct {
	service *datei.Service
}

// NewHandler returns a webdav.Handler that serves the Datei file system.
// It must be mounted at /dav.
func NewHandler(service *datei.Service) *xdav.Handler {
	return &xdav.Handler{
		Prefix:     "/dav",
		FileSystem: &dateiFS{service: service},
		LockSystem: xdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				slog.Warn("error encountered in webdav handler",
					"method", r.Method,
					"path", r.URL.Path,
					"error", err)
			}
		},
	}
}

// resolve returns the api.Datei for the given WebDAV path.
// Returns nil, nil for the virtual root ("/").
func (fs *dateiFS) resolve(ctx context.Context, name string) (*api.Datei, error) {
	name = strings.Trim(name, "/")
	if name == "" {
		return nil, nil
	}

	if d := readFromCache(ctx, name); d != nil {
		return d, nil
	}

	segments := strings.Split(name, "/")
	item, err := fs.service.FindDateiByPath(ctx, segments)
	if errors.Is(err, dateierrors.ErrNotFound) {
		return nil, os.ErrNotExist
	}
	return item, err
}

// findChild returns the non-trashed child of parentID with the given name.
func (fs *dateiFS) findChild(ctx context.Context, parentID *uuid.UUID, name string) (*api.Datei, error) {
	item, err := fs.service.FindDateiByName(ctx, parentID, name)
	if errors.Is(err, dateierrors.ErrNotFound) {
		return nil, os.ErrNotExist
	}
	return item, err
}

// resolveParent returns the parent datei and the base filename for a path.
// parent is nil when the item lives directly under the virtual root.
func (fs *dateiFS) resolveParent(ctx context.Context, name string) (*api.Datei, string, error) {
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
	if base == "" {
		return os.ErrInvalid
	}
	var parentID *uuid.UUID
	if parent != nil {
		parentID = &parent.Id
	}
	if _, err := fs.findChild(ctx, parentID, base); err == nil {
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_, err = fs.service.CreateDatei(ctx, datei.CreateDateiInput{
		ParentID: parentID,
		FileName: base,
	})
	return mapErr(err)
}

func (fs *dateiFS) RemoveAll(ctx context.Context, name string) error {
	proj, err := fs.resolve(ctx, name)
	if err != nil {
		return err
	}
	if proj == nil {
		return os.ErrPermission
	}
	return mapErr(fs.service.DeleteDatei(ctx, proj.Id))
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
	if newBase == "" {
		return os.ErrInvalid
	}
	var newParentID *uuid.UUID
	if newParent != nil {
		newParentID = &newParent.Id
	}

	if existing, err := fs.findChild(ctx, newParentID, newBase); err == nil {
		if existing.Id != proj.Id {
			return os.ErrExist
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	sameParent := (proj.ParentId == nil && newParentID == nil) ||
		(proj.ParentId != nil && newParentID != nil && *proj.ParentId == *newParentID)

	input := datei.UpdateDateiInput{ID: proj.Id}
	currentName := ""
	if proj.Name != nil {
		currentName = *proj.Name
	}
	if newBase != currentName {
		input.Name = &newBase
	}
	if !sameParent {
		input.MoveRequested = true
		input.NewParentID = newParentID
	}
	_, err = fs.service.UpdateDatei(ctx, input)
	return mapErr(err)
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

	if name == "/" || name == "" {
		return newDirFile(rootInfo(), func() ([]api.Datei, error) {
			children, err := fs.service.ListDateiChildren(ctx, nil)
			if err == nil {
				writeToCache(ctx, "", children)
			}
			return children, err
		}), nil
	}

	if !isWrite {
		proj, err := fs.resolve(ctx, name)
		if err != nil {
			return nil, err
		}

		if proj.IsDirectory {
			id := proj.Id
			parentTrimmed := strings.Trim(name, "/")
			return newDirFile(projInfo(proj), func() ([]api.Datei, error) {
				children, err := fs.service.ListDateiChildren(ctx, &id)
				if err == nil {
					writeToCache(ctx, parentTrimmed, children)
				}
				return children, err
			}), nil
		}

		id := proj.Id
		return newReadFile(projInfo(proj), func() (*datei.DownloadDateiOutput, error) {
			return fs.service.DownloadDatei(ctx, id)
		}), nil
	}

	parent, base, err := fs.resolveParent(ctx, name)
	if err != nil {
		return nil, err
	}
	var parentID *uuid.UUID
	if parent != nil {
		parentID = &parent.Id
	}

	existing, lookupErr := fs.findChild(ctx, parentID, base)
	var existingID *uuid.UUID
	if lookupErr == nil {
		if existing.IsDirectory {
			return nil, os.ErrInvalid
		}
		existingID = &existing.Id
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

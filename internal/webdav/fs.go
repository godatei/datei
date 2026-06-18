package webdav

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/file"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	xdav "golang.org/x/net/webdav"
)

type davCacheKey struct{}

type davPathCache map[string]*api.File

func (c davPathCache) store(parentPath string, children []api.File) {
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

func readFromCache(ctx context.Context, name string) *api.File {
	if cache := cacheFromContext(ctx); cache != nil {
		if d, ok := cache[name]; ok {
			return d
		}
	}
	return nil
}

func writeToCache(ctx context.Context, parentPath string, children []api.File) {
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
// by ListFileChildren.
func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), davCacheKey{}, make(davPathCache))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type fileFS struct {
	service *file.Service
}

// NewHandler returns a webdav.Handler that serves the Datei file system.
// It must be mounted at /dav.
func NewHandler(service *file.Service) *xdav.Handler {
	return &xdav.Handler{
		Prefix:     "/dav",
		FileSystem: &fileFS{service: service},
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

// resolve returns the api.File for the given WebDAV path.
// Returns nil, nil for the virtual root ("/").
func (fs *fileFS) resolve(ctx context.Context, name string) (*api.File, error) {
	name = strings.Trim(name, "/")
	if name == "" {
		return nil, nil
	}

	if d := readFromCache(ctx, name); d != nil {
		return d, nil
	}

	segments := strings.Split(name, "/")
	item, err := fs.service.FindFileByPath(ctx, segments)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, os.ErrNotExist
	}
	return item, err
}

// findChild returns the non-trashed child of parentID with the given name.
func (fs *fileFS) findChild(ctx context.Context, parentID *uuid.UUID, name string) (*api.File, error) {
	item, err := fs.service.FindFileByName(ctx, parentID, name)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, os.ErrNotExist
	}
	return item, err
}

// resolveParent returns the parent file and the base filename for a path.
// parent is nil when the item lives directly under the virtual root.
func (fs *fileFS) resolveParent(ctx context.Context, name string) (*api.File, string, error) {
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

func (fs *fileFS) Mkdir(ctx context.Context, name string, _ os.FileMode) error {
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
	_, err = fs.service.CreateFile(ctx, file.CreateFileInput{
		ParentID: parentID,
		FileName: base,
	})
	return mapErr(err)
}

func (fs *fileFS) RemoveAll(ctx context.Context, name string) error {
	proj, err := fs.resolve(ctx, name)
	if err != nil {
		return err
	}
	if proj == nil {
		return os.ErrPermission
	}
	return mapErr(fs.service.DeleteFile(ctx, proj.Id))
}

func (fs *fileFS) Rename(ctx context.Context, oldName, newName string) error {
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

	input := file.UpdateFileInput{ID: proj.Id}
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
	_, err = fs.service.UpdateFile(ctx, input)
	return mapErr(err)
}

func (fs *fileFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
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

func (fs *fileFS) OpenFile(ctx context.Context, name string, flag int, _ os.FileMode) (xdav.File, error) {
	isWrite := flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE) != 0

	if name == "/" || name == "" {
		return newDirFile(rootInfo(), func() ([]api.File, error) {
			children, err := fs.service.ListFileChildren(ctx, nil)
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
			return newDirFile(projInfo(proj), func() ([]api.File, error) {
				children, err := fs.service.ListFileChildren(ctx, &id)
				if err == nil {
					writeToCache(ctx, parentTrimmed, children)
				}
				return children, err
			}), nil
		}

		id := proj.Id
		return newReadFile(projInfo(proj), func() (*file.DownloadFileOutput, error) {
			return fs.service.DownloadFile(ctx, id)
		}), nil
	}

	parent, base, err := fs.resolveParent(ctx, name)
	if err != nil {
		return nil, err
	}
	if base == "" {
		return nil, os.ErrInvalid
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
	case errors.Is(err, apperrors.ErrNotFound),
		errors.Is(err, apperrors.ErrParentNotFound),
		errors.Is(err, apperrors.ErrParentTrashed):
		return os.ErrNotExist
	case errors.Is(err, apperrors.ErrIsDirectory),
		errors.Is(err, apperrors.ErrParentNotDirectory),
		errors.Is(err, apperrors.ErrCycleDetected):
		return os.ErrInvalid
	default:
		return err
	}
}

package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	xdav "golang.org/x/net/webdav"
)

// fileInfo implements os.FileInfo for an api.Datei or the virtual root.
type fileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (fi *fileInfo) Name() string       { return fi.name }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) IsDir() bool        { return fi.isDir }
func (fi *fileInfo) ModTime() time.Time { return fi.modTime }
func (fi *fileInfo) Sys() any           { return nil }
func (fi *fileInfo) Mode() os.FileMode {
	if fi.isDir {
		return os.ModeDir | 0o755
	}
	return 0o644
}

func rootInfo() *fileInfo {
	return &fileInfo{name: "/", isDir: true}
}

func projInfo(p *api.Datei) *fileInfo {
	size := int64(0)
	if p.Size != nil {
		size = *p.Size
	}
	name := ""
	if p.Name != nil {
		name = *p.Name
	}
	return &fileInfo{
		name:    name,
		size:    size,
		modTime: p.UpdatedAt,
		isDir:   p.IsDirectory,
	}
}

// davBase provides Stat, DeadProps, and Patch shared by all three file types.
type davBase struct {
	info os.FileInfo
}

func (b *davBase) Stat() (os.FileInfo, error)                        { return b.info, nil }
func (b *davBase) DeadProps() (map[xml.Name]xdav.Property, error)    { return nil, nil }
func (b *davBase) Patch(_ []xdav.Proppatch) ([]xdav.Propstat, error) { return nil, nil }

// dateiDirFile implements xdav.File for directory listings.
type dateiDirFile struct {
	davBase
	getChildren func() ([]api.Datei, error)
	childPos    int
}

func newDirFile(info os.FileInfo, load func() ([]api.Datei, error)) *dateiDirFile {
	return &dateiDirFile{davBase: davBase{info: info}, getChildren: sync.OnceValues(load)}
}

func (f *dateiDirFile) Close() error                       { return nil }
func (f *dateiDirFile) Read(_ []byte) (int, error)         { return 0, os.ErrInvalid }
func (f *dateiDirFile) Write(_ []byte) (int, error)        { return 0, os.ErrInvalid }
func (f *dateiDirFile) Seek(_ int64, _ int) (int64, error) { return 0, os.ErrInvalid }

func (f *dateiDirFile) Readdir(count int) ([]os.FileInfo, error) {
	children, err := f.getChildren()
	if err != nil {
		return nil, err
	}
	remaining := children[f.childPos:]
	if count > 0 {
		if count > len(remaining) {
			count = len(remaining)
		}
		remaining = remaining[:count]
	}
	if len(remaining) == 0 && count > 0 {
		return nil, io.EOF
	}
	infos := make([]os.FileInfo, len(remaining))
	for i := range remaining {
		infos[i] = projInfo(&children[f.childPos+i])
	}
	f.childPos += len(remaining)
	return infos, nil
}

// dateiReadFile implements xdav.File for readable files.
// Content is loaded lazily from S3 on the first Read or Seek call.
type dateiReadFile struct {
	davBase
	getReader func() (*bytes.Reader, error)
}

func newReadFile(info os.FileInfo, load func() (*datei.DownloadDateiOutput, error)) *dateiReadFile {
	return &dateiReadFile{
		davBase: davBase{info: info},
		getReader: sync.OnceValues(func() (*bytes.Reader, error) {
			out, err := load()
			if err != nil {
				return nil, err
			}
			if rc, ok := out.Reader.(io.Closer); ok {
				defer rc.Close()
			}
			data, err := io.ReadAll(out.Reader)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(data), nil
		}),
	}
}

func (f *dateiReadFile) Close() error                         { return nil }
func (f *dateiReadFile) Write(_ []byte) (int, error)          { return 0, os.ErrInvalid }
func (f *dateiReadFile) Readdir(_ int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }

func (f *dateiReadFile) Read(p []byte) (int, error) {
	r, err := f.getReader()
	if err != nil {
		return 0, err
	}
	return r.Read(p)
}

func (f *dateiReadFile) Seek(offset int64, whence int) (int64, error) {
	r, err := f.getReader()
	if err != nil {
		return 0, err
	}
	return r.Seek(offset, whence)
}

// dateiWriteFile implements xdav.File for file uploads. Data is buffered to a
// temp file and uploaded (create or update) when Close is called.
type dateiWriteFile struct {
	davBase
	writeCtx   context.Context
	writeName  string
	parentID   *uuid.UUID
	existingID *uuid.UUID
	service    *datei.Service
	tmpFile    *os.File
}

func newWriteFile(
	ctx context.Context,
	name string,
	parentID, existingID *uuid.UUID,
	service *datei.Service,
) (*dateiWriteFile, error) {
	tmp, err := os.CreateTemp("", "datei-webdav-*")
	if err != nil {
		return nil, err
	}
	return &dateiWriteFile{
		davBase:    davBase{info: &fileInfo{name: name, modTime: time.Now()}},
		writeCtx:   ctx,
		writeName:  name,
		parentID:   parentID,
		existingID: existingID,
		service:    service,
		tmpFile:    tmp,
	}, nil
}

func (f *dateiWriteFile) Read(_ []byte) (int, error)           { return 0, os.ErrInvalid }
func (f *dateiWriteFile) Seek(_ int64, _ int) (int64, error)   { return 0, os.ErrInvalid }
func (f *dateiWriteFile) Readdir(_ int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }

func (f *dateiWriteFile) Write(p []byte) (int, error) {
	return f.tmpFile.Write(p)
}

// Close uploads the buffered content to the service and removes the temp file.
func (f *dateiWriteFile) Close() error {
	tmp := f.tmpFile
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}

	contentType := mime.TypeByExtension(filepath.Ext(f.writeName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if f.existingID == nil {
		_, err := f.service.CreateDatei(f.writeCtx, datei.CreateDateiInput{
			ParentID:    f.parentID,
			Reader:      tmp,
			FileName:    f.writeName,
			ContentType: contentType,
		})
		return mapErr(err)
	}
	_, err := f.service.UpdateDatei(f.writeCtx, datei.UpdateDateiInput{
		ID:          *f.existingID,
		Reader:      tmp,
		FileName:    f.writeName,
		ContentType: contentType,
	})
	return mapErr(err)
}

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

	"github.com/godatei/datei/internal/file"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	xdav "golang.org/x/net/webdav"
)

// fileInfo implements os.FileInfo for an api.File or the virtual root.
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

func projInfo(p *api.File) *fileInfo {
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

// fileDirFile implements xdav.File for directory listings.
type fileDirFile struct {
	davBase
	getChildren func() ([]api.File, error)
	childPos    int
}

func newDirFile(info os.FileInfo, load func() ([]api.File, error)) *fileDirFile {
	return &fileDirFile{davBase: davBase{info: info}, getChildren: sync.OnceValues(load)}
}

func (f *fileDirFile) Close() error                       { return nil }
func (f *fileDirFile) Read(_ []byte) (int, error)         { return 0, os.ErrInvalid }
func (f *fileDirFile) Write(_ []byte) (int, error)        { return 0, os.ErrInvalid }
func (f *fileDirFile) Seek(_ int64, _ int) (int64, error) { return 0, os.ErrInvalid }

func (f *fileDirFile) Readdir(count int) ([]os.FileInfo, error) {
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

// fileReadFile implements xdav.File for readable files.
// Content is loaded lazily from S3 on the first Read or Seek call.
type fileReadFile struct {
	davBase
	getReader func() (*bytes.Reader, error)
}

func newReadFile(info os.FileInfo, load func() (*file.DownloadFileOutput, error)) *fileReadFile {
	return &fileReadFile{
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

func (f *fileReadFile) Close() error                         { return nil }
func (f *fileReadFile) Write(_ []byte) (int, error)          { return 0, os.ErrInvalid }
func (f *fileReadFile) Readdir(_ int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }

func (f *fileReadFile) Read(p []byte) (int, error) {
	r, err := f.getReader()
	if err != nil {
		return 0, err
	}
	return r.Read(p)
}

func (f *fileReadFile) Seek(offset int64, whence int) (int64, error) {
	r, err := f.getReader()
	if err != nil {
		return 0, err
	}
	return r.Seek(offset, whence)
}

// fileWriteFile implements xdav.File for file uploads. Data is buffered to a
// temp file and uploaded (create or update) when Close is called.
type fileWriteFile struct {
	davBase
	writeCtx   context.Context
	writeName  string
	parentID   *uuid.UUID
	existingID *uuid.UUID
	service    *file.Service
	tmpFile    *os.File
}

func newWriteFile(
	ctx context.Context,
	name string,
	parentID, existingID *uuid.UUID,
	service *file.Service,
) (*fileWriteFile, error) {
	tmp, err := os.CreateTemp("", "datei-webdav-*")
	if err != nil {
		return nil, err
	}
	return &fileWriteFile{
		davBase:    davBase{info: &fileInfo{name: name, modTime: time.Now()}},
		writeCtx:   ctx,
		writeName:  name,
		parentID:   parentID,
		existingID: existingID,
		service:    service,
		tmpFile:    tmp,
	}, nil
}

func (f *fileWriteFile) Read(_ []byte) (int, error)           { return 0, os.ErrInvalid }
func (f *fileWriteFile) Seek(_ int64, _ int) (int64, error)   { return 0, os.ErrInvalid }
func (f *fileWriteFile) Readdir(_ int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }

func (f *fileWriteFile) Write(p []byte) (int, error) {
	return f.tmpFile.Write(p)
}

// Close uploads the buffered content to the service and removes the temp file.
func (f *fileWriteFile) Close() error {
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
		_, err := f.service.CreateFile(f.writeCtx, file.CreateFileInput{
			ParentID:    f.parentID,
			Reader:      tmp,
			FileName:    f.writeName,
			ContentType: contentType,
		})
		return mapErr(err)
	}
	_, err := f.service.UpdateFile(f.writeCtx, file.UpdateFileInput{
		ID:          *f.existingID,
		Reader:      tmp,
		FileName:    f.writeName,
		ContentType: contentType,
	})
	return mapErr(err)
}

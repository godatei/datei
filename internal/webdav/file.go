package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"mime"
	"os"
	"path/filepath"
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

// dateiFile implements webdav.File for directories, readable files, and
// writable files. Write-mode files buffer to a temp file and upload on Close.
type dateiFile struct {
	info os.FileInfo

	// dir mode
	children []api.Datei
	childPos int

	// read mode (file content buffered for seek support)
	reader *bytes.Reader

	// write mode
	writeCtx   context.Context
	writeName  string
	parentID   *uuid.UUID
	existingID *uuid.UUID
	service    *datei.Service
	tmpFile    *os.File
}

func newDirFile(info os.FileInfo, children []api.Datei) *dateiFile {
	return &dateiFile{info: info, children: children}
}

// newReadFile buffers the entire file content so that Seek works correctly
// for range requests and http.ServeContent.
func newReadFile(info os.FileInfo, out *datei.DownloadDateiOutput) (*dateiFile, error) {
	data, err := io.ReadAll(out.Reader)
	if rc, ok := out.Reader.(io.Closer); ok {
		rc.Close()
	}
	if err != nil {
		return nil, err
	}
	return &dateiFile{info: info, reader: bytes.NewReader(data)}, nil
}

func newWriteFile(
	ctx context.Context,
	name string,
	parentID, existingID *uuid.UUID,
	service *datei.Service,
) (*dateiFile, error) {
	tmp, err := os.CreateTemp("", "datei-webdav-*")
	if err != nil {
		return nil, err
	}
	return &dateiFile{
		info:       &fileInfo{name: name, modTime: time.Now()},
		writeCtx:   ctx,
		writeName:  name,
		parentID:   parentID,
		existingID: existingID,
		service:    service,
		tmpFile:    tmp,
	}, nil
}

func (f *dateiFile) Stat() (os.FileInfo, error) { return f.info, nil }

func (f *dateiFile) Read(p []byte) (int, error) {
	if f.reader != nil {
		return f.reader.Read(p)
	}
	return 0, os.ErrInvalid
}

func (f *dateiFile) Write(p []byte) (int, error) {
	if f.tmpFile != nil {
		return f.tmpFile.Write(p)
	}
	return 0, os.ErrInvalid
}

func (f *dateiFile) Seek(offset int64, whence int) (int64, error) {
	if f.reader != nil {
		return f.reader.Seek(offset, whence)
	}
	if f.tmpFile != nil {
		return f.tmpFile.Seek(offset, whence)
	}
	return 0, os.ErrInvalid
}

func (f *dateiFile) Readdir(count int) ([]os.FileInfo, error) {
	remaining := f.children[f.childPos:]
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
		infos[i] = projInfo(&f.children[f.childPos+i])
	}
	f.childPos += len(remaining)
	return infos, nil
}

// Close is a no-op for read/dir files. For write files it uploads the buffered
// content to the service and removes the temp file.
func (f *dateiFile) Close() error {
	if f.tmpFile == nil {
		return nil
	}
	tmp := f.tmpFile
	f.tmpFile = nil
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
		return err
	}
	_, err := f.service.UpdateDatei(f.writeCtx, datei.UpdateDateiInput{
		ID:          *f.existingID,
		Reader:      tmp,
		FileName:    f.writeName,
		ContentType: contentType,
	})
	return err
}

// DeadProps returns an empty map — no custom WebDAV properties are stored.
func (f *dateiFile) DeadProps() (map[xml.Name]xdav.Property, error) {
	return nil, nil
}

// Patch is a no-op — custom WebDAV properties are not persisted.
func (f *dateiFile) Patch(_ []xdav.Proppatch) ([]xdav.Propstat, error) {
	return nil, nil
}

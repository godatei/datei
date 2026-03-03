package storage

import (
	"context"
	"io"
)

type Store interface {
	Initialize(ctx context.Context) error
	GetObject(ctx context.Context, reference string) (io.ReadCloser, error)
	PutObject(ctx context.Context, data io.Reader, name, contentType string) (*PutObjectOutput, error)
	DeleteObject(ctx context.Context, reference string) error
}

type PutObjectOutput struct {
	StorageKey string
	Checksum   string
	Size       int64
}

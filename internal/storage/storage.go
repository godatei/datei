package storage

import (
	"context"
	"io"
)

type Store interface {
	Initialize(ctx context.Context) error
	GetObject(ctx context.Context, reference string) (io.ReadCloser, error)
	PutObject(ctx context.Context, data io.Reader, contentType string) (string, int64, error)
	DeleteObject(ctx context.Context, reference string) error
}

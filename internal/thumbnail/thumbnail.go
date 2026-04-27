package thumbnail

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/godatei/datei/internal/dateierrors"
)

const maxDimension = 512

// Generate creates a JPEG thumbnail from r for the given mimeType.
// Returns [dateierrors.ErrUnsupportedMediaType] if the MIME type is not supported.
func Generate(ctx context.Context, r io.Reader, mimeType string) ([]byte, error) {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))

	if strings.HasPrefix(mimeType, "image/") {
		return generateImage(r)
	}

	switch mimeType {
	case "application/pdf",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return generateDocument(ctx, r, mimeType)
	}

	return nil, fmt.Errorf("%w: %s", dateierrors.ErrUnsupportedMediaType, mimeType)
}

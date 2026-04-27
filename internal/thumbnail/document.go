package thumbnail

import (
	"context"
	"fmt"
	"image"
	"io"

	"github.com/gen2brain/go-fitz"
)

func generateDocument(_ context.Context, r io.Reader, mimeType string) ([]byte, error) {
	doc, err := fitz.NewFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("open document (%s): %w", mimeType, err)
	}
	defer doc.Close()

	if doc.NumPage() == 0 {
		return nil, fmt.Errorf("document has no pages")
	}

	img, err := doc.Image(0)
	if err != nil {
		return nil, fmt.Errorf("render page 0: %w", err)
	}

	return encodeJPEG(resizeFit(image.Image(img), maxDimension))
}

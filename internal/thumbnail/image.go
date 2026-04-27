package thumbnail

import (
	"bytes"
	"image"
	// Register additional decoders.
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func generateImage(r io.Reader) ([]byte, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return encodeJPEG(resizeFit(src, maxDimension))
}

// resizeFit scales img so its longest side is at most max, preserving aspect ratio.
// Returns the original if it already fits.
func resizeFit(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}

	var dw, dh int
	if w >= h {
		dw = max
		dh = (h * max) / w
	} else {
		dh = max
		dw = (w * max) / h
	}
	if dh < 1 {
		dh = 1
	}
	if dw < 1 {
		dw = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

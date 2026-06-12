package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Client calls a rapidocr_api server to extract text from images.
type Client struct {
	serverURI  string
	httpClient *http.Client
}

func NewClient(serverURI string) *Client {
	return &Client{
		serverURI: serverURI,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// ocrResponse mirrors the rapidocr_api response: a map keyed by the detected
// line index ("0", "1", ...) where each value holds the recognized text. An
// empty object is returned when no text is detected.
type ocrResponse map[string]struct {
	RecTxt string `json:"rec_txt"`
}

// ExtractText sends a single image to the OCR server and returns the recognized
// text, with one detected line per output line.
func (c *Client) ExtractText(ctx context.Context, r io.Reader) (string, error) {
	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)

	go func() {
		defer func() { pw.CloseWithError(w.Close()) }()

		fw, err := w.CreateFormFile("image_file", "image")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("ocr: create form file: %w", err))
			return
		}
		if _, err := io.Copy(fw, r); err != nil {
			pw.CloseWithError(fmt.Errorf("ocr: copy file: %w", err))
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURI, pr)
	if err != nil {
		pr.CloseWithError(err)
		return "", fmt.Errorf("ocr: create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		pr.CloseWithError(err)
		return "", fmt.Errorf("ocr: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ocr: server returned %d", resp.StatusCode)
	}

	var result ocrResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ocr: decode response: %w", err)
	}

	return joinLines(result), nil
}

// joinLines orders the detected lines by their numeric index and joins their
// recognized text with newlines.
func joinLines(result ocrResponse) string {
	keys := make([]int, 0, len(result))
	index := make(map[int]string, len(result))
	for k, v := range result {
		i, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		keys = append(keys, i)
		index[i] = v.RecTxt
	}
	sort.Ints(keys)

	var b strings.Builder
	for i, k := range keys {
		if i != 0 {
			b.WriteByte('\n')
		}
		b.WriteString(index[k])
	}
	return b.String()
}

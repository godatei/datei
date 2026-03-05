package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

// Client calls an OCR server to extract text from images and PDFs.
type Client struct {
	serverURI  string
	httpClient *http.Client
}

func NewClient(serverURI string) *Client {
	return &Client{
		serverURI:  serverURI,
		httpClient: &http.Client{},
	}
}

type ocrResponse struct {
	Text string `json:"text"`
}

// ExtractText sends a file to the OCR server and returns the extracted text.
// contentType must be an image MIME type or "application/pdf".
func (c *Client) ExtractText(ctx context.Context, r io.Reader, contentType string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="file"`)
	h.Set("Content-Type", contentType)
	fw, err := w.CreatePart(h)
	if err != nil {
		return "", fmt.Errorf("ocr: create form file: %w", err)
	}
	if _, err := io.Copy(fw, r); err != nil {
		return "", fmt.Errorf("ocr: copy file: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("ocr: close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURI, &buf)
	if err != nil {
		return "", fmt.Errorf("ocr: create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
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

	return result.Text, nil
}

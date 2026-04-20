package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/ledongthuc/pdf"
	"github.com/otiai10/gosseract/v2"
	"github.com/spf13/pflag"
	"gopkg.in/gographics/imagick.v3/imagick"
)

type Options struct {
	Addr string
}

func (opts *Options) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Addr, "addr", ":8585", "Serve address")
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	imagick.Initialize()
	defer imagick.Terminate()

	var opts Options
	opts.Bind(pflag.CommandLine)
	pflag.Parse()

	r := chi.NewMux()
	r.Post("/", handlePost)

	slog.Info("server started", "addr", opts.Addr)
	if err := http.ListenAndServe(opts.Addr, r); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("serve error", "error", err)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	// max memory 1GiB
	const maxMultipartMemory = 1 << 30
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		slog.Debug("error parsing multipart form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var lang []string
	if langStr := r.FormValue("lang"); langStr != "" {
		if err := json.Unmarshal([]byte(langStr), &lang); err != nil {
			slog.Debug("error decoding lang", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		lang = []string{"eng"}
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		slog.Debug("error getting multipart file", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		slog.Error("failed to create temp dir", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	slog.Debug("created temp dir", "dir", tmpDir)

	tmpFile, err := os.Create(path.Join(tmpDir, "file"))
	if err != nil {
		slog.Error("failed to create temp file", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		slog.Error("failed to copy file", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tmpFile.Close()

	var ocrFiles []string
	if contentType := fileHeader.Header.Get("Content-Type"); contentType == "application/pdf" {
		pdfText, err := pdfToText(tmpFile.Name())
		if err != nil {
			slog.Warn("error getting PDF text", "error", err)
		} else if pdfText != "" {
			w.Header().Add("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"text": pdfText}); err != nil {
				slog.Debug("failed to write response", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		} else {
			slog.Debug("pdf text is empty")
		}

		ocrFiles, err = split(tmpFile.Name())
		if err != nil {
			slog.Error("error splitting PDF", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		ocrFiles = []string{tmpFile.Name()}
	}

	if len(ocrFiles) > 1 {
		slices.SortFunc(ocrFiles, func(a, b string) int {
			aa, errA := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(path.Base(a), "page-"), ".jpg"))
			bb, errB := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(path.Base(b), "page-"), ".jpg"))
			if errA != nil || errB != nil {
				return strings.Compare(path.Base(a), path.Base(b))
			}
			return aa - bb
		})
	}

	slog.Debug("running ocr", "files", ocrFiles)

	out := new(strings.Builder)
	for i, file := range ocrFiles {
		slog.Debug("running ocr", "file", file)
		out1, err := ocr(file, lang, "")
		if err != nil {
			slog.Error("ocr error", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if i != 0 {
			// emulate page break
			out.WriteString("\n\n---\n\n")
		}
		out.WriteString(out1)
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"text": out.String()}); err != nil {
		slog.Debug("failed to write response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func pdfToText(file string) (string, error) {
	f, r, err := pdf.Open(file)
	if err != nil {
		return "", fmt.Errorf("pdf.Open: %w", err)
	}
	defer f.Close()

	tr, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("GetPlainText: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, tr); err != nil {
		return "", fmt.Errorf("Copy: %w", err)
	}

	return buf.String(), nil
}

func split(file string) ([]string, error) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	dir := path.Dir(file)

	_, err := imagick.ConvertImageCommand([]string{
		"convert",
		"-density", "300",
		file,
		path.Join(dir, "page.jpg"),
	})
	if err != nil {
		return nil, fmt.Errorf("ConvertImageCommand: %w", err)
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ReadDir: %w", err)
	}

	var result []string
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "page") {
			result = append(result, path.Join(dir, entry.Name()))
		}
	}

	return result, nil
}

func ocr(file string, langs []string, format string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetImage(file); err != nil {
		return "", fmt.Errorf("SetImage: %w", err)
	}

	if len(langs) > 0 {
		if err := client.SetLanguage(langs...); err != nil {
			return "", fmt.Errorf("SetLanguage: %w", err)
		}
	}

	switch format {
	case "hocr":
		return client.HOCRText()
	default:
		return client.Text()
	}
}

package server

import (
	"cmp"
	"mime"
)

func attachmentDisposition(filename string) string {
	return mime.FormatMediaType(
		"attachment",
		map[string]string{"filename": cmp.Or(filename, "download")},
	)
}

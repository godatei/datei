package server

import (
	"fmt"
	"io"
)

// readLimited reads at most maxBytes from r. If the input exceeds maxBytes, an error is returned.
func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	buf, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > maxBytes {
		return nil, fmt.Errorf("input exceeds %d bytes", maxBytes)
	}
	return buf, nil
}

package httpauth

import (
	"testing"

	. "github.com/onsi/gomega"
)

const wantToken = "abc123"

func TestParseBearer(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{name: "valid", header: "Bearer abc123", wantToken: wantToken, wantOK: true},
		{name: "lowercase scheme", header: "bearer abc123", wantToken: wantToken, wantOK: true},
		{name: "extra whitespace", header: "  Bearer    abc123  ", wantToken: wantToken, wantOK: true},
		{name: "empty", header: "", wantOK: false},
		{name: "missing token", header: "Bearer", wantOK: false},
		{name: "wrong scheme", header: "Basic abc123", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			token, ok := ParseBearer(tt.header)
			g.Expect(ok).To(Equal(tt.wantOK))
			g.Expect(token).To(Equal(tt.wantToken))
		})
	}
}

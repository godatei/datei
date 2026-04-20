package mailer

import (
	"context"
	"log/slog"
)

// NoopMailer logs emails but does not send them.
type NoopMailer struct{}

// NewNoopMailer creates a new no-op mailer.
func NewNoopMailer() *NoopMailer {
	return &NoopMailer{}
}

func (m *NoopMailer) Send(_ context.Context, mail Mail) error {
	slog.Info("noop mailer: email not sent", "to", mail.To, "subject", mail.Subject)
	return nil
}

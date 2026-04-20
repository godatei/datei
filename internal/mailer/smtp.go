package mailer

import (
	"context"
	"fmt"

	gomail "github.com/wneessen/go-mail"
)

// SMTPMailer sends emails via SMTP.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPMailer creates a new SMTP mailer.
func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, mail Mail) error {
	msg := gomail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("failed to set from: %w", err)
	}
	if err := msg.To(mail.To...); err != nil {
		return fmt.Errorf("failed to set to: %w", err)
	}
	msg.Subject(mail.Subject)
	msg.SetBodyString(gomail.TypeTextHTML, mail.HTML)

	opts := []gomail.Option{
		gomail.WithPort(m.port),
		gomail.WithTLSPortPolicy(gomail.TLSOpportunistic),
	}
	if m.username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(m.username),
			gomail.WithPassword(m.password),
		)
	}

	client, err := gomail.NewClient(m.host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}
	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("failed to send mail: %w", err)
	}
	return nil
}

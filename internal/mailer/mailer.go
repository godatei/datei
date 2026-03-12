package mailer

import "context"

// Mail represents an email to send.
type Mail struct {
	To      []string
	From    string
	Subject string
	HTML    string
}

// Mailer sends emails.
type Mailer interface {
	Send(ctx context.Context, mail Mail) error
}

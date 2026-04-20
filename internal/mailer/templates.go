package mailer

import (
	"bytes"
	"html/template"
)

var passwordResetTmpl = template.Must(template.New("password_reset").Parse(
	`<h2>Password Reset</h2>
<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="{{.URL}}">Reset Password</a></p>
<p>This link will expire in 1 hour. If you did not request this, please ignore this email.</p>`))

var emailVerificationTmpl = template.Must(template.New("email_verification").Parse(
	`<h2>Email Verification</h2>
<p>Please verify your email address by clicking the link below:</p>
<p><a href="{{.URL}}">Verify Email</a></p>
<p>If you did not create an account, please ignore this email.</p>`))

// PasswordResetEmail returns an email for password reset.
func PasswordResetEmail(to, resetURL string) Mail {
	var buf bytes.Buffer
	_ = passwordResetTmpl.Execute(&buf, struct{ URL string }{URL: resetURL})
	return Mail{
		To:      []string{to},
		Subject: "Password Reset",
		HTML:    buf.String(),
	}
}

// EmailVerificationEmail returns an email for email verification.
func EmailVerificationEmail(to, verifyURL string) Mail {
	var buf bytes.Buffer
	_ = emailVerificationTmpl.Execute(&buf, struct{ URL string }{URL: verifyURL})
	return Mail{
		To:      []string{to},
		Subject: "Verify your email",
		HTML:    buf.String(),
	}
}

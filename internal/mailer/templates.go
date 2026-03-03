package mailer

import "fmt"

// PasswordResetEmail returns an email for password reset.
func PasswordResetEmail(to, resetURL string) Mail {
	return Mail{
		To:      []string{to},
		Subject: "Password Reset",
		HTML: fmt.Sprintf(`<h2>Password Reset</h2>
<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link will expire in 1 hour. If you did not request this, please ignore this email.</p>`, resetURL),
	}
}

// EmailVerificationEmail returns an email for email verification.
func EmailVerificationEmail(to, verifyURL string) Mail {
	return Mail{
		To:      []string{to},
		Subject: "Verify your email",
		HTML: fmt.Sprintf(`<h2>Email Verification</h2>
<p>Please verify your email address by clicking the link below:</p>
<p><a href="%s">Verify Email</a></p>
<p>If you did not create an account, please ignore this email.</p>`, verifyURL),
	}
}

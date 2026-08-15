package services

import "log"

// Mailer handles transactional email sending.
// This is a stub — swap in SendGrid, Mailgun, SMTP, etc.
type Mailer struct{}

// NewMailer creates a new Mailer.
func NewMailer() *Mailer {
	return &Mailer{}
}

// SendInvitation sends an invitation email with the given token.
// The token is intentionally truncated in logs to avoid leaking the
// full secret into any log aggregation system.
func (m *Mailer) SendInvitation(email, token string) error {
	log.Printf("[MAILER] invitation to %s with token %s…", email, redactToken(token))
	return nil
}

// redactToken returns a shortened, non-reversible representation of a
// sensitive token for logging purposes. It never logs more than 8 chars.
func redactToken(token string) string {
	const prefixLen = 8
	if token == "" {
		return "***"
	}
	if len(token) <= prefixLen {
		return token[:1] + "***"
	}
	return token[:prefixLen] + "***"
}

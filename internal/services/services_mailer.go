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
func (m *Mailer) SendInvitation(email, token string) error {
	log.Printf("[MAILER] invitation to %s with token %s", email, token)
	return nil
}

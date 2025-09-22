package notification

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailNotifier implements the Notifier interface for sending emails.
type EmailNotifier struct {
	cfg  config.SMTPConfig
	auth smtp.Auth
}

// NewEmailNotifier creates a new EmailNotifier.
func NewEmailNotifier(cfg config.SMTPConfig) model.Notifier {
	// PlainAuth will not send credentials until the server identifies itself as a trusted one.
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	return &EmailNotifier{cfg: cfg, auth: auth}
}

// Send sends an email to the configured recipients.
func (n *EmailNotifier) Send(subject, body string) error {
	addr := fmt.Sprintf("%s:%d", n.cfg.Host, n.cfg.Port)
	recipients := strings.Split(n.cfg.To, ",")

	// Construct the email message.
	msg := []byte("To: " + n.cfg.To + "\r\n" +
		"From: " + n.cfg.From + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body)

	// Send the email.
	err := smtp.SendMail(addr, n.auth, n.cfg.From, recipients, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

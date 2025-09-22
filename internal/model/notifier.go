package model

// Notifier defines a generic interface for sending notifications.
type Notifier interface {
	Send(subject, body string) error
}

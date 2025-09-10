package model

// Writer defines a generic interface for writing aggregator data to a persistent store.
type Writer interface {
	// Write takes a data payload and persists it.
	// The implementation is expected to know how to handle the payload type it receives.
	Write(payload interface{}, timestamp string) error
}

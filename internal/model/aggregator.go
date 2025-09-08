package model

// Aggregator defines the common interface for an aggregation engine,
// allowing different types of aggregators (e.g., exact, probabilistic) to be used interchangeably.
type Aggregator interface {
	// Start launches the aggregator's processing workers.
	Start()

	// Stop gracefully shuts down the aggregator, ensuring all data is processed or flushed.
	Stop()

	// Input returns the channel to which packets should be sent for processing.
	Input() chan<- *PacketInfo
}

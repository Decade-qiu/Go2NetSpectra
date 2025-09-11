# Aggregation Task Implementations

This directory contains the concrete implementations of aggregation tasks (`model.Task`). Each subdirectory should represent a self-contained aggregator type (e.g., `exact`, `hyperloglog`, etc.).

## Creating a New Aggregator

When adding a new aggregator implementation, you must follow two critical steps to ensure it is correctly registered and available to the engine:

### 1. Register the Aggregator in `init()`

Your implementation package **must** have an `init()` function that calls `factory.RegisterAggregator` to register itself with the central factory. This makes the aggregator discoverable by its type name as defined in the configuration.

**Example (`internal/engine/impl/your_aggregator/task.go`):**
```go
package your_aggregator

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
)

func init() {
	factory.RegisterAggregator("your_aggregator_name", func(cfg *config.Config) ([]model.Task, model.Writer, error) {
		// ... factory logic to create tasks and a writer
	})
}

// ... rest of your task implementation
```

### 2. Add a Blank Import in the Manager

To ensure that the Go compiler includes your package and executes its `init()` function, you **must** add a blank import for your new package in `internal/engine/manager/manager.go`.

**Example (`internal/engine/manager/manager.go`):**
```go
package manager

import (
	// ... other imports
	_ "Go2NetSpectra/internal/engine/impl/exact" // Existing aggregator
	_ "Go2NetSpectra/internal/engine/impl/your_aggregator" // <-- Add your new package here
)

// ... rest of the manager code
```

By following these two steps, your new aggregator will be automatically available for use by the `Manager` when specified in the `config.yaml`.

package factory

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"fmt"
)

// TaskFactory defines a function that creates tasks and their associated writers.
type TaskFactory func(cfg *config.Config) ([]model.Task, []model.Writer, error)

// registry holds the mapping of aggregator types to their factory functions.
var registry = make(map[string]TaskFactory)

// RegisterAggregator registers a new aggregator type with its factory function.
func RegisterAggregator(name string, factory TaskFactory) {
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("aggregator type '%s' already registered", name))
	}
	registry[name] = factory
}

// Create creates tasks and writers based on the provided config.
func Create(cfg *config.Config) ([]model.Task, []model.Writer, error) {
	factory, ok := registry[cfg.Aggregator.Type]
	if !ok {
		return nil, nil, fmt.Errorf("unknown aggregator type: '%s'", cfg.Aggregator.Type)
	}
	return factory(cfg)
}
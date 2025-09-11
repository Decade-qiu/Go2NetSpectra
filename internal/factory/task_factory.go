package factory

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"fmt"
)

// TaskFactory defines a function that creates tasks and a writer based on the config.
type TaskFactory func(cfg *config.Config) ([]model.Task, model.Writer, error)

// registry holds the mapping of aggregator types to their factory functions.
var registry = map[string]TaskFactory{}

// RegisterAggregator registers a new aggregator type with its factory function.
// 1. should be called in init() of each implementation package.
// 2. each the implementation package should be imported in manager.go 
//   to ensure init() is executed.
func RegisterAggregator(name string, factory TaskFactory) {
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("aggregator type '%s' already registered", name))
	}
	registry[name] = factory
}

// Create creates tasks and a writer based on the provided config.
func Create(cfg *config.Config) ([]model.Task, model.Writer, error) {
	factory, ok := registry[cfg.Aggregator.Type]
	if !ok {
		return nil, nil, fmt.Errorf("unknown aggregator type: '%s'", cfg.Aggregator.Type)
	}
	return factory(cfg)
}

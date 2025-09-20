package factory

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"fmt"
	"log"
)

// TaskGroup is a logical grouping of tasks and their associated writers.
type TaskGroup struct {
	Tasks   []model.Task
	Writers []model.Writer
}

// TaskFactory defines a function that creates a group of tasks and their writers.
type TaskFactory func(cfg *config.Config) (*TaskGroup, error)

// registry holds the mapping of aggregator types to their factory functions.
var registry = make(map[string]TaskFactory)

// RegisterAggregator registers a new aggregator type with its factory function.
func RegisterAggregator(name string, factory TaskFactory) {
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("aggregator type '%s' already registered", name))
	}
	registry[name] = factory
}

// Create creates a list of TaskGroups based on the provided config.
func Create(cfg *config.Config) ([]TaskGroup, error) {
	var taskGroups []TaskGroup

	for _, aggType := range cfg.Aggregator.Types {
		log.Printf("Creating tasks and writers for aggregator type: '%s'\n", aggType)

		factory, ok := registry[aggType]
		if !ok {
			return nil, fmt.Errorf("unknown aggregator type: '%s'", aggType)
		}

		group, err := factory(cfg)
		if err != nil {
			return nil, fmt.Errorf("error creating aggregator type '%s': %w", aggType, err)
		}

		taskGroups = append(taskGroups, *group)
	}

	return taskGroups, nil
}
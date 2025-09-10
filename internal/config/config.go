package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AggregationTaskDef defines a single aggregation task from the config file.
type ExactAggregationTaskDef struct {
	Name      string   `yaml:"name"`
	NumShards uint32      `yaml:"num_shards"`
	KeyFields []string `yaml:"key_fields"`
}

// AggregatorConfig holds the configuration for the flow aggregator.
type AggregatorConfig struct {
	Type                string                    `yaml:"type"`
	ExactTasks          []ExactAggregationTaskDef `yaml:"exact_tasks"`
	SnapshotInterval    string                    `yaml:"snapshot_interval"`
	StorageRootPath     string                    `yaml:"storage_root_path"`
	NumWorkers          int                       `yaml:"num_workers"`
	SizeOfPacketChannel int                       `yaml:"size_of_packet_channel"`
}

// Config is the top-level configuration struct for the entire application.
type Config struct {
	Aggregator AggregatorConfig `yaml:"aggregator"`
}

// LoadConfig reads the configuration from a YAML file and returns a Config struct.
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}

	return &cfg, nil
}

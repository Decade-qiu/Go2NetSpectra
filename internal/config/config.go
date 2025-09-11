package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ClickHouseConfig holds the configuration for the ClickHouse writer.
type ClickHouseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// GobConfig holds the configuration for the gob file writer.
type GobConfig struct {
	RootPath string `yaml:"root_path"`
}

// WriterDef defines a writer configuration.
type WriterDef struct {
	Type             string           `yaml:"type"`
	Enabled          bool             `yaml:"enabled"`
	SnapshotInterval string           `yaml:"snapshot_interval"`
	Gob              GobConfig        `yaml:"gob"`
	ClickHouse       ClickHouseConfig `yaml:"clickhouse"`
}

// ExactTaskDef defines a single task's parameters within the exact aggregator group.
type ExactTaskDef struct {
	Name      string   `yaml:"name"`
	NumShards uint32   `yaml:"num_shards"`
	KeyFields []string `yaml:"key_fields"`
}

// ExactAggregatorConfig holds all configuration for the "exact" aggregator type.
type ExactAggregatorConfig struct {
	Writers []WriterDef    `yaml:"writers"`
	Tasks   []ExactTaskDef `yaml:"tasks"`
}

// AggregatorConfig holds the top-level aggregator settings.
type AggregatorConfig struct {
	Type                string                `yaml:"type"`
	Period              string                `yaml:"period"`
	NumWorkers          int                   `yaml:"num_workers"`
	SizeOfPacketChannel int                   `yaml:"size_of_packet_channel"`
	Exact               ExactAggregatorConfig `yaml:"exact"`
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

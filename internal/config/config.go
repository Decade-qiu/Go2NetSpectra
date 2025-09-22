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
	Cloud    bool   `yaml:"cloud"`
}

// GobConfig holds the configuration for the gob file writer.
type GobConfig struct {
	RootPath string `yaml:"root_path"`
}

// TextConfig holds the configuration for the text file writer.
type TextConfig struct {
	RootPath string `yaml:"root_path"`
}

// WriterDef defines a writer configuration.
type WriterDef struct {
	Type             string           `yaml:"type"`
	Enabled          bool             `yaml:"enabled"`
	SnapshotInterval string           `yaml:"snapshot_interval"`
	Gob              GobConfig        `yaml:"gob"`
	Text             TextConfig       `yaml:"text"`
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

// SketchTaskDef defines a single task's parameters within the sketch aggregator group.
type SketchTaskDef struct {
	Name            string   `yaml:"name"`
	SktType         uint8    `yaml:"skt_type"` // 0 for CountMin, 1 for SuperSpread
	FlowFields      []string `yaml:"flow_fields"`
	ElementFields   []string `yaml:"element_fields"`
	Width           uint32   `yaml:"width"`
	Depth           uint32   `yaml:"depth"`
	SizeThereshold  uint32   `yaml:"size_thereshold"`
	CountThereshold uint32   `yaml:"count_thereshold"`
	// SuperSpread specific parameters
	M    uint32  `yaml:"m"`
	Size uint32  `yaml:"size"`
	Base float64 `yaml:"base"`
	B    float64 `yaml:"b"`
}

// SketchTaskDef defines a single task's parameters within the sketch aggregator group.
type SketchAggregatorConfig struct {
	Writers []WriterDef     `yaml:"writers"`
	Tasks   []SketchTaskDef `yaml:"tasks"`
}

// AggregatorConfig holds the top-level aggregator settings.
type AggregatorConfig struct {
	Types               []string               `yaml:"types"`
	Period              string                 `yaml:"period"`
	NumWorkers          int                    `yaml:"num_workers"`
	SizeOfPacketChannel int                    `yaml:"size_of_packet_channel"`
	Exact               ExactAggregatorConfig  `yaml:"exact"`
	Sketch              SketchAggregatorConfig `yaml:"sketch"`
}

// PersistenceConfig holds the configuration for the probe's local persistence worker.
type PersistenceConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Path              string `yaml:"path"`
	NumWorkers        int    `yaml:"num_workers"`
	Encoding          string `yaml:"encoding"` // "text" or "gob"
	ChannelBufferSize int    `yaml:"channel_buffer_size"`
}

// ProbeConfig holds the configuration for the probe component.
type ProbeConfig struct {
	NATSURL     string            `yaml:"nats_url"`
	Subject     string            `yaml:"subject"`
	Persistence PersistenceConfig `yaml:"persistence"`
}

// APIConfig holds the configuration for the API server.
type APIConfig struct {
	GrpcListenAddr string `yaml:"grpc_listen_addr"`
	HttpListenAddr string `yaml:"http_listen_addr"`
}

// AlerterRule defines a single condition for triggering an alert.
type AlerterRule struct {
	Name      string  `yaml:"name"`
	TaskName  string  `yaml:"task_name"`
	Metric    string  `yaml:"metric"`    // e.g., "heavy_hitter_count", "super_spreader_spread", "total_bytes"
	Operator  string  `yaml:"operator"`  // e.g., ">", "<", "="
	Threshold float64 `yaml:"threshold"`
}

// AlerterConfig holds all configuration for the alerter component.
type AlerterConfig struct {
	Enabled       bool          `yaml:"enabled"`
	CheckInterval string        `yaml:"check_interval"`
	Rules         []AlerterRule `yaml:"rules"`
}

// SMTPConfig holds the configuration for the email notifier.
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	To       string `yaml:"to"` // Comma-separated list of recipients
}

// Config is the top-level configuration struct for the entire application.
type Config struct {
	Aggregator AggregatorConfig `yaml:"aggregator"`
	Probe      ProbeConfig      `yaml:"probe"`
	API        APIConfig        `yaml:"api"`
	Alerter    AlerterConfig    `yaml:"alerter"`
	SMTP       SMTPConfig       `yaml:"smtp"`
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

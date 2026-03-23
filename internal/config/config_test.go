package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigExpandsEnvironmentVariables(t *testing.T) {
	t.Setenv("GO2NETSPECTRA_TEST_API_KEY", "super-secret")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
ai:
  api_key: ${GO2NETSPECTRA_TEST_API_KEY}
  grpc_listen_addr: ":50052"
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error: %v", configPath, err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", configPath, err)
	}

	if cfg.AI.APIKey != "super-secret" {
		t.Fatalf("LoadConfig(%q) APIKey = %q, want %q", configPath, cfg.AI.APIKey, "super-secret")
	}
	if cfg.AI.GRPCListenAddr != ":50052" {
		t.Fatalf("LoadConfig(%q) GRPCListenAddr = %q, want %q", configPath, cfg.AI.GRPCListenAddr, ":50052")
	}
}

func TestLoadConfigReturnsYAMLError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("ai: [broken")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error: %v", configPath, err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatalf("LoadConfig(%q) error = nil, want non-nil", configPath)
	}
	if !strings.Contains(err.Error(), "failed to unmarshal config yaml") {
		t.Fatalf("LoadConfig(%q) error = %q, want substring %q", configPath, err.Error(), "failed to unmarshal config yaml")
	}
}

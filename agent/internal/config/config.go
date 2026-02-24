package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerURL string `yaml:"server_url"` // e.g., ws://panel.example.com/api/v1/agents/ws
	AgentID   string `yaml:"agent_id"`
	Token     string `yaml:"token"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Allow env overrides
	if v := os.Getenv("SAKURA_AGENT_SERVER_URL"); v != "" {
		cfg.ServerURL = v
	}
	if v := os.Getenv("SAKURA_AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
	if v := os.Getenv("SAKURA_AGENT_TOKEN"); v != "" {
		cfg.Token = v
	}

	if cfg.ServerURL == "" || cfg.AgentID == "" || cfg.Token == "" {
		return nil, fmt.Errorf("server_url, agent_id, and token are required")
	}

	return &cfg, nil
}

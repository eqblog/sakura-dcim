package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `server_url: ws://localhost:8080/api/v1/agents/ws
agent_id: test-agent-001
token: test-token-abc123
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ServerURL != "ws://localhost:8080/api/v1/agents/ws" {
		t.Errorf("unexpected ServerURL: %s", cfg.ServerURL)
	}
	if cfg.AgentID != "test-agent-001" {
		t.Errorf("unexpected AgentID: %s", cfg.AgentID)
	}
	if cfg.Token != "test-token-abc123" {
		t.Errorf("unexpected Token: %s", cfg.Token)
	}
}

func TestLoad_MissingFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `server_url: ws://localhost:8080/api/v1/agents/ws
`
	os.WriteFile(cfgPath, []byte(content), 0644)

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("expected error for missing agent_id and token")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `server_url: ws://original
agent_id: original-id
token: original-token
`
	os.WriteFile(cfgPath, []byte(content), 0644)

	t.Setenv("SAKURA_AGENT_SERVER_URL", "ws://override")
	t.Setenv("SAKURA_AGENT_ID", "override-id")
	t.Setenv("SAKURA_AGENT_TOKEN", "override-token")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ServerURL != "ws://override" {
		t.Errorf("env override not applied: ServerURL = %s", cfg.ServerURL)
	}
	if cfg.AgentID != "override-id" {
		t.Errorf("env override not applied: AgentID = %s", cfg.AgentID)
	}
	if cfg.Token != "override-token" {
		t.Errorf("env override not applied: Token = %s", cfg.Token)
	}
}

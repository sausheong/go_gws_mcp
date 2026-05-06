package core

import (
	"strings"
	"testing"
)

func TestConfig_DefaultsWhenEnvUnset(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("WORKSPACE_MCP_PORT", "")
	t.Setenv("WORKSPACE_MCP_BASE_URI", "")
	t.Setenv("WORKSPACE_MCP_HOST", "")
	t.Setenv("WORKSPACE_MCP_TRANSPORT", "")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test-client")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8000 {
		t.Errorf("Port = %d, want 8000", cfg.Port)
	}
	if cfg.BaseURI != "http://localhost" {
		t.Errorf("BaseURI = %q, want http://localhost", cfg.BaseURI)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Transport != "stdio" {
		t.Errorf("Transport = %q, want stdio", cfg.Transport)
	}
}

func TestConfig_PortFromEnv(t *testing.T) {
	t.Setenv("WORKSPACE_MCP_PORT", "9001")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test")
	cfg, _ := Load()
	if cfg.Port != 9001 {
		t.Errorf("Port = %d, want 9001", cfg.Port)
	}
}

func TestConfig_RejectsInvalidTransport(t *testing.T) {
	t.Setenv("WORKSPACE_MCP_TRANSPORT", "carrier-pigeon")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "transport") {
		t.Fatalf("want transport error, got %v", err)
	}
}

func TestConfig_RejectsInvalidTier(t *testing.T) {
	t.Setenv("WORKSPACE_MCP_TOOL_TIER", "gold")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "tier") {
		t.Fatalf("want tier error, got %v", err)
	}
}

func TestConfig_PermissionsAndToolsAreMutuallyExclusive(t *testing.T) {
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test")
	cfg := &Config{
		Permissions:  map[string]string{"gmail": "send"},
		EnabledTools: []string{"gmail"},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("want mutex error, got %v", err)
	}
}

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sausheong/go_gws_mcp/internal/auth"
)

// Config holds parsed runtime configuration for the server.
type Config struct {
	Port              int
	BaseURI           string
	Host              string
	ExternalURL       string
	Transport         string
	ClientID          string
	ClientSecret      string
	RedirectURI       string
	SingleUser        bool
	DefaultEmail      string
	EnabledTools      []string
	ToolTier          string
	Permissions       map[string]string
	CredentialsDir    string
	InsecureTransport bool
}

// Load builds a Config from env vars. Returns descriptive error on bad values.
// Flags (parsed in main.go) override the returned config before Validate().
func Load() (*Config, error) {
	cfg := &Config{
		Port:           8000,
		BaseURI:        "http://localhost",
		Host:           "0.0.0.0",
		Transport:      "stdio",
		CredentialsDir: defaultCredentialsDir(),
	}

	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("invalid PORT %q", v)
		}
		cfg.Port = p
	}
	if v := os.Getenv("WORKSPACE_MCP_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("invalid WORKSPACE_MCP_PORT %q", v)
		}
		cfg.Port = p
	}
	if v := os.Getenv("WORKSPACE_MCP_BASE_URI"); v != "" {
		cfg.BaseURI = v
	}
	if v := os.Getenv("WORKSPACE_MCP_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("WORKSPACE_EXTERNAL_URL"); v != "" {
		cfg.ExternalURL = v
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("WORKSPACE_MCP_TRANSPORT"))); v != "" {
		if v != "stdio" && v != "streamable-http" {
			return nil, fmt.Errorf("invalid transport %q from WORKSPACE_MCP_TRANSPORT (want stdio|streamable-http)", v)
		}
		cfg.Transport = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_CLIENT_ID"); v != "" {
		cfg.ClientID = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"); v != "" {
		cfg.ClientSecret = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_REDIRECT_URI"); v != "" {
		cfg.RedirectURI = v
	}
	if v := os.Getenv("USER_GOOGLE_EMAIL"); v != "" {
		cfg.DefaultEmail = v
	}
	if v := strings.TrimSpace(os.Getenv("WORKSPACE_MCP_TOOLS")); v != "" {
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(strings.ToLower(parts[i]))
		}
		cfg.EnabledTools = parts
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("WORKSPACE_MCP_TOOL_TIER"))); v != "" {
		if v != "core" && v != "extended" && v != "complete" {
			return nil, fmt.Errorf("invalid tier %q from WORKSPACE_MCP_TOOL_TIER (want core|extended|complete)", v)
		}
		cfg.ToolTier = v
	}
	if v := strings.TrimSpace(os.Getenv("WORKSPACE_MCP_PERMISSIONS")); v != "" {
		entries := strings.Fields(v)
		parsed, err := auth.ParsePermissions(entries)
		if err != nil {
			return nil, fmt.Errorf("WORKSPACE_MCP_PERMISSIONS: %w", err)
		}
		cfg.Permissions = parsed
	}
	if v := os.Getenv("MCP_SINGLE_USER_MODE"); v == "1" || strings.EqualFold(v, "true") {
		cfg.SingleUser = true
	}
	if v := os.Getenv("WORKSPACE_MCP_CREDENTIALS_DIR"); v != "" {
		cfg.CredentialsDir = expandHome(v)
	}
	if v := os.Getenv("OAUTHLIB_INSECURE_TRANSPORT"); v == "1" || strings.EqualFold(v, "true") {
		cfg.InsecureTransport = true
	}

	if cfg.RedirectURI == "" {
		cfg.RedirectURI = fmt.Sprintf("%s:%d/oauth2callback", cfg.BaseURI, cfg.Port)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate enforces mutual-exclusion rules. Called after flag overrides.
func (c *Config) Validate() error {
	if len(c.Permissions) > 0 && len(c.EnabledTools) > 0 {
		return fmt.Errorf("--permissions and --tools are mutually exclusive (also via WORKSPACE_MCP_PERMISSIONS / WORKSPACE_MCP_TOOLS env vars)")
	}
	return nil
}

// EffectiveExternalURL returns ExternalURL if set, else "BaseURI:Port".
func (c *Config) EffectiveExternalURL() string {
	if c.ExternalURL != "" {
		return c.ExternalURL
	}
	return fmt.Sprintf("%s:%d", c.BaseURI, c.Port)
}

func defaultCredentialsDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".google_workspace_mcp", "credentials")
	}
	return ".credentials"
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

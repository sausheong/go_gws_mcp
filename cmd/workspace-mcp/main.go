// Command workspace-mcp is the entry point for the Go MCP server.
// Parses CLI flags and env vars, builds the OAuth client, registers tools,
// and runs the configured transport.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
	"github.com/sausheong/go_gws_mcp/internal/core/tooltier"
	"github.com/sausheong/go_gws_mcp/internal/gmail"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// Load env-derived config first; flags override.
	cfg, err := core.Load()
	if err != nil {
		return err
	}

	transport := flag.String("transport", cfg.Transport, "stdio or streamable-http")
	singleUser := flag.Bool("single-user", cfg.SingleUser, "Bypass session->user mapping (use any cred from store)")
	toolsCSV := flag.String("tools", strings.Join(cfg.EnabledTools, ","), "Comma-separated services to enable")
	tier := flag.String("tool-tier", cfg.ToolTier, "core|extended|complete")
	permsCSV := flag.String("permissions", "", "Per-service levels, e.g. 'gmail:organize'")
	readOnly := flag.Bool("read-only", false, "Stubbed: accepted but no enforcement (logs warning)")
	flag.Parse()

	cfg.Transport = *transport
	cfg.SingleUser = *singleUser
	if *toolsCSV != "" {
		parts := strings.Split(*toolsCSV, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(strings.ToLower(parts[i]))
		}
		cfg.EnabledTools = parts
	}
	if *tier != "" {
		cfg.ToolTier = *tier
	}
	if *permsCSV != "" {
		entries := strings.Fields(*permsCSV)
		parsed, err := auth.ParsePermissions(entries)
		if err != nil {
			return fmt.Errorf("--permissions: %w", err)
		}
		cfg.Permissions = parsed
	}
	if *readOnly {
		slog.Warn("--read-only is accepted but not enforced in the skeleton")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	if cfg.ClientID == "" {
		return errors.New("GOOGLE_OAUTH_CLIENT_ID is required (set env var)")
	}

	// Resolve scopes via tier loader if --tool-tier is set; else use full ToolScopesMap.
	enabledServices := cfg.EnabledTools
	if len(enabledServices) == 0 {
		enabledServices = []string{"gmail"}
	}
	if cfg.ToolTier != "" {
		loader, err := tooltier.New()
		if err != nil {
			return fmt.Errorf("tier loader: %w", err)
		}
		_, services, err := loader.ResolveToolsFromTier(cfg.ToolTier, enabledServices)
		if err != nil {
			return err
		}
		enabledServices = services
	}
	scopes := auth.ScopesForTools(enabledServices)

	// Build credential store + OAuth client.
	store, err := auth.NewLocalDirectoryStore(cfg.CredentialsDir)
	if err != nil {
		return err
	}
	oauthClient := auth.NewOAuthClient(cfg.ClientID, cfg.ClientSecret, cfg.RedirectURI, scopes, store)

	// Build MCP server.
	srv := server.NewMCPServer(
		"go-gws-mcp",
		"0.1.0",
		server.WithToolCapabilities(false),
	)
	registry := core.NewRegistry()

	// Register Gmail tools (the only service in the skeleton).
	gmail.RegisterTools(srv, registry, oauthClient, cfg.DefaultEmail)

	// Filter pass (logs intended removals; mcp-go's runtime removal API is
	// out of scope for the skeleton — see internal/core/server.go).
	registry.Filter(cfg)

	// Set up signal handling.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info("starting workspace-mcp",
		"transport", cfg.Transport,
		"port", cfg.Port,
		"tools", registry.Names(),
		"default_email", cfg.DefaultEmail,
	)

	if err := core.Run(ctx, srv, cfg, oauthClient); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

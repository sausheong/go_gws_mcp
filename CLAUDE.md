# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build and install
go install ./cmd/workspace-mcp

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/auth/...
go test ./internal/core/...

# Run the server (stdio transport, default)
workspace-mcp

# Run with streamable-HTTP transport
workspace-mcp --transport streamable-http
```

The binary requires `GOOGLE_OAUTH_CLIENT_ID` (and optionally `GOOGLE_OAUTH_CLIENT_SECRET`) set in the environment. Copy `.env.example` to `.env` for local dev.

## Architecture

This is a Go MCP server (`github.com/mark3labs/mcp-go`) that wraps the Gmail API with OAuth 2.0. It is a Go port of the [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp) Python project, scoped to Gmail as a proof-of-concept skeleton.

**No `cobra`, `viper`, `zap`, or `logrus`.** Only stdlib + four direct dependencies: `mcp-go`, `golang.org/x/oauth2`, `google.golang.org/api`, and `gopkg.in/yaml.v3`.

### Startup sequence (`cmd/workspace-mcp/main.go`)

1. `core.Load()` reads env vars into `*Config`
2. `flag.Parse()` overlays CLI flags onto `cfg`
3. `cfg.Validate()` enforces mutual-exclusion rules (e.g. `--permissions` cannot combine with `--tools`)
4. `auth.NewLocalDirectoryStore` + `auth.NewOAuthClient` build the credential store and OAuth client
5. `gmail.RegisterTools(srv, registry, oauthClient, defaultEmail)` registers all tools
6. `registry.Filter(cfg)` logs intended removals (actual mcp-go removal is a future hook)
7. `core.Run(ctx, srv, cfg, oauthClient)` dispatches to stdio or streamable-HTTP

### The decorator pattern (`internal/auth/service.go`)

Go lacks Python's `@require_google_service` decorator, so the pattern uses `RequireGmailService[T]`, a generic higher-order function that produces an `mcp.ToolHandlerFunc`. Every Gmail tool body has the signature:

```go
func(ctx context.Context, svc *gmail.Service, userEmail string, args T) (string, error)
```

`RequireGmailService` wraps this with: arg binding (JSON round-trip, not `BindArguments`), email resolution (request → ctx → `defaultEmail`), credential loading + refresh, `gmail.Service` construction, and `apierror.Format` on errors. On `*AuthRequiredError` it returns the "ACTION REQUIRED" auth-flow message as a text tool result rather than an error.

### Tool registration pattern (`internal/gmail/register.go`)

Each tool does three things in the same function:
1. Define the `mcp.Tool` (name, description, parameters)
2. `reg.Record(name, scopes)` — tells the registry about the tool for the filter pass
3. `srv.AddTool(tool, auth.RequireGmailService(..., handler))` — wires it to mcp-go

`gmail.RegisterTools` calls one `register*` function per tool.

### Auth / credential flow (`internal/auth/`)

- `OAuthClient` holds the `oauth2.Config` and a `CredentialStore`
- `credstore.go`: `CredentialStore` interface + `LocalDirectoryStore` (one `<url-encoded-email>.json` per user, mode `0600`, atomic rename writes)
- `oauth.go`: PKCE flow with `crypto/rand`; `GetCredentials` loads, auto-refreshes, and scope-checks tokens
- `callback.go`: in stdio mode, spins up a minimal `net/http` goroutine on `localhost:<port>` to receive the OAuth callback; in streamable-HTTP mode the main mux handles `/oauth2callback`
- `scopes.go`: constants, `ScopeHierarchy` (broader scopes cover narrower ones), `HasRequiredScopes`, `ToolScopesMap`
- `permissions.go`: parses `--permissions gmail:organize` syntax; validates against `servicePermissionLevels`; **enforcement is a no-op** (the filter pass only logs)

### Config (`internal/core/config.go`)

`core.Load()` returns a singleton `*Config` from env vars. `cfg.Validate()` enforces mutual exclusions. `GOOGLE_OAUTH_CLIENT_ID` is the only required field. All other env vars have defaults.

### Transport (`internal/core/transport.go`)

`core.Run` switches on `cfg.Transport`:
- `"stdio"` (default): ensures callback server is running, calls `server.ServeStdio`
- `"streamable-http"`: builds an `http.ServeMux` with `/health`, `/oauth2callback`, and `/mcp` (mcp-go's streamable handler)

### Error handling (`internal/core/apierror/handler.go`)

`apierror.Format(toolName, err)` detects `*googleapi.Error` and returns LLM-targeted prose with "ACTION REQUIRED" headers and Cloud Console links. 401/403 get auth-flow instructions; other API errors get a plain error string.

### Tool tier loader (`internal/core/tooltier/`)

`tiers.yaml` (embedded via `//go:embed`) maps service → tier → tool names. `loader.ResolveToolsFromTier(tier, services)` returns `(toolNames, serviceNames)`. Currently only Gmail is populated.

## Adding a new service

1. Create `internal/<service>/` with one `*.go` per tool body and `register.go`
2. Add scope constants to `internal/auth/scopes.go`; extend `ToolScopesMap`
3. Add tier entries to `internal/core/tooltier/tiers.yaml`
4. Add the service to `servicePermissionLevels` in `internal/auth/permissions.go`
5. Call `<service>.RegisterTools(srv, registry, oauthClient, cfg.DefaultEmail)` in `cmd/workspace-mcp/main.go`

The new service's tools should follow the same registration pattern as Gmail: define tool → `reg.Record` → `srv.AddTool` with a `RequireXxxService` wrapper analogous to `RequireGmailService`.

## Testing

Unit tests cover scopes, permissions parser, config validation, credential store, PKCE, tier loader, and Gmail body extraction. `go test ./...` passes with no Google credentials. Live API tests are explicitly out of scope.

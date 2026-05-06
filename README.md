# go-gws-mcp

Go port (architectural skeleton + Gmail) of [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp). Demonstrates the patterns that make the Python project work; ships with 5 Gmail tools as the proof-of-concept.

## What's in scope

- 5 Gmail tools: `search_gmail_messages`, `get_gmail_message_content`, `get_gmail_messages_content_batch`, `send_gmail_message`, `list_gmail_labels`
- OAuth 2.0 with PKCE
- Local-directory credential store
- stdio and streamable-HTTP transports
- Tool tier loader (YAML-driven)
- Granular permissions parser (parsed but not enforced — by design)

See [docs/design.md](docs/design.md) for a full architecture overview, and [docs/plans/](docs/plans/) for the implementation plan.

## Setup

1. **Create OAuth client** at [Google Cloud Console](https://console.cloud.google.com/apis/credentials):
   - Application type: Desktop application (or Web for confidential client)
   - For Web: add `http://localhost:8000/oauth2callback` as a redirect URI
   - Enable the Gmail API for your project

2. **Configure credentials** — copy `.env.example` to `.env` and fill in `GOOGLE_OAUTH_CLIENT_ID` / `GOOGLE_OAUTH_CLIENT_SECRET`.

3. **Build and install:**

   ```bash
   go install ./cmd/workspace-mcp
   ```

## Usage

### stdio (Claude Desktop, Codex CLI, etc.)

```bash
workspace-mcp
```

Or, for a Claude Desktop config snippet:

```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "workspace-mcp",
      "env": {
        "GOOGLE_OAUTH_CLIENT_ID": "...",
        "GOOGLE_OAUTH_CLIENT_SECRET": "...",
        "USER_GOOGLE_EMAIL": "you@example.com"
      }
    }
  }
}
```

On the first tool call, the server returns an "ACTION REQUIRED" message containing an OAuth URL. Visit it, complete the consent flow in your browser, then retry the original tool call.

### streamable-HTTP (hosted)

```bash
workspace-mcp --transport streamable-http
```

Endpoints:
- `POST /mcp` — MCP transport
- `GET /oauth2callback` — OAuth callback
- `GET /health` — liveness probe

## Adding a new service

1. Create `internal/<service>/` with one `*.go` per tool body and a `register.go`
2. Add scope constants to `internal/auth/scopes.go`; add to `ToolScopesMap`
3. Add tier entries to `internal/core/tooltier/tiers.yaml`
4. Add the service entry to `servicePermissionLevels` in `internal/auth/permissions.go`
5. Call `<service>.RegisterTools(...)` from `cmd/workspace-mcp/main.go`

## Differences from the Python project

- **No** OAuth 2.1, external OAuth, service-account modes
- **No** GCS credential store, Valkey OAuth proxy storage
- **No** stateless mode
- **No** read-only mode enforcement (flag accepted, logs warning)
- **No** granular permissions enforcement (parsed correctly, no tool removal)
- **No** attachment storage, SSRF-safe HTTP, file uploads
- **No** Helm chart, Dockerfile, Smithery / FastMCP Cloud entry points
- **No** workspace-cli companion
- 1 service (Gmail) instead of 12

The architecture leaves named hooks for each — see `docs/design.md` § 17.

## Tests

```bash
go test ./...
```

Unit tests cover scopes, permissions parser, config validation, credential store, PKCE, tier loader, and Gmail body extraction. Live Google API tests are out of scope.

## License

MIT

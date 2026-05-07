# go-gws-mcp

Go port of [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp), covering 5 services (Gmail, Drive, Docs, Sheets, Slides) with 5 core tools each (25 total).

## What's in scope

- 5 Gmail tools: `search_gmail_messages`, `get_gmail_message_content`, `get_gmail_messages_content_batch`, `send_gmail_message`, `list_gmail_labels`
- 5 Google Drive tools: `search_drive_files`, `list_drive_items`, `get_drive_file_content`, `create_drive_folder`, `create_drive_file`
- 5 Google Docs tools: `search_docs`, `get_doc_content`, `get_doc_as_markdown`, `create_doc`, `modify_doc_text`
- 5 Google Sheets tools: `list_spreadsheets`, `get_spreadsheet_info`, `read_sheet_values`, `modify_sheet_values`, `create_spreadsheet`
- 5 Google Slides tools: `create_presentation`, `get_presentation`, `batch_update_presentation`, `get_page`, `get_page_thumbnail`
- OAuth 2.0 with PKCE
- Local-directory credential store
- stdio and streamable-HTTP transports
- Tool tier loader (YAML-driven)
- Granular permissions parser (parsed but not enforced — by design)

## Setup

1. **Create OAuth client** at [Google Cloud Console](https://console.cloud.google.com/apis/credentials):
   - Application type: Desktop application (or Web for confidential client)
   - For Web: add `http://localhost:8000/oauth2callback` as a redirect URI
   - Enable the Gmail, Drive, Docs, Sheets, and Slides APIs for your project

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
- 5 services (Gmail, Drive, Docs, Sheets, Slides) instead of 12 — Calendar, Chat, Forms, Tasks, Contacts, Custom Search, Apps Script not ported

## Tests

```bash
go test ./...
```

Unit tests cover scopes, permissions parser, config validation, credential store, PKCE, tier loader, and Gmail body extraction. Live Google API tests are out of scope.

## License

MIT — see [LICENSE](LICENSE).

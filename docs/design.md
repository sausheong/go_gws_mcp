# go-gws-mcp — Design

A Go port of the [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp) Python server, scoped to **architectural skeleton + Gmail service**. Demonstrates the patterns that make the Python project work, leaves clearly-marked hooks for the rest.

**Status:** design proposal, awaiting approval before implementation.

---

## 1. Goals & non-goals

### Goals

- Reproduce the Python server's **architectural patterns** in idiomatic Go:
  - Single decorator-equivalent that handles OAuth, scope checks, service injection, error wrapping, and cleanup
  - Decorator-time tool registration with post-hoc filtering (tier-aware, scope-aware)
  - Centralized config singleton that fails loud on mutual-exclusion violations
  - Trust boundaries: emails come from validated sources only
  - LLM-shaped error messages with embedded recovery steps
- Ship Gmail with its 5 core tools (search, get, batch get, send, list labels) so the auth + transport plumbing is exercised end-to-end
- Support **stdio** and **streamable-HTTP** transports
- Include **granular permissions** as no-op scaffolding so the architecture is complete

### Non-goals (explicit)

- OAuth 2.1, external OAuth, service-account modes (only OAuth 2.0)
- GCS credential store (only local directory)
- Stateless mode
- Read-only mode (the scope-filter pass is stubbed for future use)
- Attachment storage / SSRF-safe URL fetching / file uploads
- The other 11 services (Drive, Calendar, Docs, Sheets, etc.)
- `workspace-cli` companion, Helm chart, Dockerfile, FastMCP Cloud entrypoint

The architecture leaves named hooks for each non-goal so they can be added incrementally without re-shaping core packages.

---

## 2. Dependencies

| Purpose | Module |
|---|---|
| MCP server + stdio + streamable-HTTP | `github.com/mark3labs/mcp-go` |
| Gmail API client | `google.golang.org/api/gmail/v1` |
| OAuth 2.0 flow | `golang.org/x/oauth2` + `golang.org/x/oauth2/google` |
| Tool tier YAML | `gopkg.in/yaml.v3` |
| Structured logging | stdlib `log/slog` (Go 1.21+) |
| CLI flags | stdlib `flag` |
| HTTP server (callback + streamable-HTTP) | stdlib `net/http` |

No `cobra`, no `viper`, no `zap`, no `logrus`. Stdlib + the four direct dependencies above.

Go version: **1.22+** (for `slog`, `http.ServeMux` enhancements).

---

## 3. Project layout

```
go_gws_mcp/
├── go.mod
├── go.sum
├── README.md
├── .env.example
├── docs/
│   └── design.md                    # this document
├── cmd/
│   └── workspace-mcp/
│       └── main.go                  # entry: parse flags/env, register tools, run transport
└── internal/
    ├── core/
    │   ├── server.go                # MCPServer construction + tool-registration tracking + filter pass
    │   ├── config.go                # Config singleton; env vars; mutual-exclusion validation
    │   ├── transport.go             # Transport selection (stdio vs streamable-HTTP) + HTTP routes
    │   ├── apierror/
    │   │   └── handler.go           # WrapHandler — Python's @handle_http_errors equivalent
    │   ├── mcpcontext/
    │   │   └── context.go           # context.WithValue helpers (session id, credentials)
    │   └── tooltier/
    │       ├── loader.go            # YAML loader; tier resolution
    │       └── tiers.yaml           # //go:embed-ed; just gmail entries for the skeleton
    ├── auth/
    │   ├── oauth.go                 # OAuth 2.0 flow: start, exchange, refresh, get_credentials
    │   ├── callback.go              # MinimalOAuthCallbackServer (net/http on goroutine)
    │   ├── credstore.go             # CredentialStore interface + LocalDirectory impl
    │   ├── scopes.go                # scope constants, hierarchy, HasRequiredScopes, ToolScopesMap
    │   ├── service.go               # RequireGmailService — the decorator equivalent
    │   └── permissions.go           # Granular permissions (parsed but no-op for the skeleton)
    └── gmail/
        ├── register.go              # RegisterTools(srv) — wires all Gmail tools
        ├── search.go                # search_gmail_messages
        ├── get.go                   # get_gmail_message_content
        ├── batch.go                 # get_gmail_messages_content_batch
        ├── send.go                  # send_gmail_message
        ├── labels.go                # list_gmail_labels
        └── helpers.go               # body extraction, MIME helpers, formatters
```

**Why `internal/`:** Prevents external imports. The skeleton is meant to be extended in-tree, not consumed as a library.

---

## 4. Configuration model

`internal/core/config.go` exposes a singleton `*Config` built once from env vars at startup. Mirrors `auth/oauth_config.py:OAuthConfig`.

```go
type Config struct {
    // Server
    Port           int           // PORT or WORKSPACE_MCP_PORT, default 8000
    BaseURI        string        // WORKSPACE_MCP_BASE_URI, default "http://localhost"
    Host           string        // WORKSPACE_MCP_HOST, default "0.0.0.0"
    ExternalURL    string        // WORKSPACE_EXTERNAL_URL (reverse proxy)
    Transport      string        // --transport flag, default "stdio"

    // OAuth client
    ClientID       string        // GOOGLE_OAUTH_CLIENT_ID (required)
    ClientSecret   string        // GOOGLE_OAUTH_CLIENT_SECRET
    RedirectURI    string        // derived or GOOGLE_OAUTH_REDIRECT_URI

    // Single-user mode
    SingleUser     bool          // --single-user / MCP_SINGLE_USER_MODE
    DefaultEmail   string        // USER_GOOGLE_EMAIL

    // Tool selection
    EnabledTools   []string      // --tools or WORKSPACE_MCP_TOOLS (CSV)
    ToolTier       string        // --tool-tier or WORKSPACE_MCP_TOOL_TIER (core/extended/complete)

    // Granular permissions (parsed; not enforced in skeleton)
    Permissions    map[string]string  // --permissions svc:level

    // Credentials directory
    CredentialsDir string        // WORKSPACE_MCP_CREDENTIALS_DIR or default ~/.google_workspace_mcp/credentials

    // Insecure transport for local dev
    InsecureTransport bool       // OAUTHLIB_INSECURE_TRANSPORT
}

func Load() (*Config, error) {
    // Read env vars, then override with flags in main.go
    // Validate mutual exclusions and return descriptive error
}
```

**Mutual-exclusion rules** (from Python `OAuthConfig.__init__`):

- `--permissions` cannot combine with `--read-only` or `--tools`
- Invalid env var values (bad transport name, bad tier, bad bool) return an error rather than silently ignoring

`Load()` returns an error on any violation; `main.go` exits with `os.Exit(1)` and the error printed to stderr.

---

## 5. The decorator equivalent

The most consequential design decision. Python's `@require_google_service` injects a `service` argument and removes `user_google_email` from the schema; Go has no decorators, so we use a higher-order function that produces an `mcp.ToolHandlerFunc`.

```go
// internal/auth/service.go

// GmailHandler is the inner shape every Gmail tool implementation has.
// `svc` is injected; tool body is pure logic.
type GmailHandler[T any] func(
    ctx context.Context,
    svc *gmail.Service,
    userEmail string,
    args T,
) (string, error)

// RequireGmailService produces an mcp.ToolHandlerFunc that:
//   1. Binds JSON args into T
//   2. Resolves user_google_email (request -> session ctx -> Config.DefaultEmail)
//   3. Loads + refreshes credentials via auth.GetCredentials
//   4. Builds gmail.Service
//   5. Calls handler
//   6. Wraps Google API errors with apierror.Wrap
//   7. Closes service / cancels context
func RequireGmailService[T any](
    toolName string,
    requiredScopes []string,
    handler GmailHandler[T],
) server.ToolHandlerFunc {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var args T
        if err := req.BindArguments(&args); err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        userEmail, err := resolveUserEmail(ctx, req)
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        creds, err := GetCredentials(ctx, userEmail, requiredScopes)
        if err != nil {
            // Returns a formatted "ACTION REQUIRED: Google Auth..." block
            return mcp.NewToolResultText(formatAuthRequired(err, toolName, userEmail)), nil
        }

        svc, err := gmail.NewService(ctx, option.WithTokenSource(creds.TokenSource(ctx)))
        if err != nil {
            return apierror.Wrap(toolName, err), nil
        }

        result, err := handler(ctx, svc, userEmail, args)
        if err != nil {
            return apierror.Wrap(toolName, err), nil
        }
        return mcp.NewToolResultText(result), nil
    }
}
```

**Tool registration** records required scopes for the future filter pass:

```go
// internal/core/server.go
type ToolMetadata struct {
    Name           string
    RequiredScopes []string
}

var registeredTools []ToolMetadata

func RegisterTool(srv *server.MCPServer, tool mcp.Tool, scopes []string, handler server.ToolHandlerFunc) {
    registeredTools = append(registeredTools, ToolMetadata{
        Name:           tool.Name,
        RequiredScopes: scopes,
    })
    srv.AddTool(tool, handler)
}

// FilterTools removes registered tools that don't fit the active mode.
// In the skeleton this is mostly a no-op; the structure lets read-only mode
// and granular permissions hook in later.
func FilterTools(srv *server.MCPServer, cfg *Config) {
    // Tier filter (active in skeleton)
    if cfg.ToolTier != "" {
        // ...
    }
    // Permissions filter (stubbed: cfg.Permissions parsed but no removal)
    // Read-only filter (stubbed: not exposed yet)
}
```

**Tool definition** in `internal/gmail/search.go`:

```go
type SearchArgs struct {
    Query           string `json:"query"`
    UserGoogleEmail string `json:"user_google_email"`
    PageSize        int    `json:"page_size,omitempty"`
    PageToken       string `json:"page_token,omitempty"`
}

func registerSearch(srv *server.MCPServer) {
    tool := mcp.NewTool("search_gmail_messages",
        mcp.WithDescription("Searches messages in a user's Gmail account based on a query..."),
        mcp.WithString("query", mcp.Required(), mcp.Description("Gmail search query")),
        mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email")),
        mcp.WithNumber("page_size", mcp.Description("Max results, default 10")),
        mcp.WithString("page_token", mcp.Description("Pagination token")),
        mcp.WithReadOnlyHintAnnotation(true),
        mcp.WithIdempotentHintAnnotation(true),
        mcp.WithOpenWorldHintAnnotation(true),
    )

    core.RegisterTool(srv, tool,
        []string{auth.GmailReadonlyScope},
        auth.RequireGmailService("search_gmail_messages",
            []string{auth.GmailReadonlyScope},
            searchGmailMessages,
        ),
    )
}

func searchGmailMessages(ctx context.Context, svc *gmail.Service, userEmail string, a SearchArgs) (string, error) {
    pageSize := a.PageSize
    if pageSize == 0 { pageSize = 10 }

    call := svc.Users.Messages.List("me").Q(a.Query).MaxResults(int64(pageSize))
    if a.PageToken != "" {
        call = call.PageToken(a.PageToken)
    }
    resp, err := call.Context(ctx).Do()
    if err != nil { return "", err }

    return formatSearchResults(resp, a.Query), nil
}
```

This preserves the Python pattern's properties:

- Per-tool body is small and pure logic
- Scopes/auth/service/error-handling all live in the wrapper
- Required scopes are attached at registration so the filter pass can read them
- One handler per tool, one args struct per tool — discoverable, statically typed

---

## 6. OAuth flow

Mirrors `auth/google_auth.py` for OAuth 2.0:

### Initial auth (`StartAuthFlow`)

1. Generate 16-byte hex `state`
2. Build `oauth2.Config{ClientID, ClientSecret, Scopes, RedirectURL: cfg.RedirectURI, Endpoint: google.Endpoint}`
3. Generate PKCE code verifier (`crypto/rand` → 32 bytes → URL-safe base64); compute `S256` challenge. Mirrors the Python `create_oauth_flow` behaviour where `autogenerate_code_verifier=True` for both public and confidential clients.
4. `authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce, oauth2.SetAuthURLParam("code_challenge", challenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))` — use `prompt=select_account` instead of `ApprovalForce` when re-authing with sufficient scopes
5. Store `{state → {verifier, sessionID}}` in process-local map (and persist to `<credsDir>/oauth_states.json` to survive callback-server restarts, mirroring Python)
6. Return formatted "**ACTION REQUIRED: Google Authentication Needed for...**" block (mirrors Python `start_auth_flow:541-568`)

### Callback (`HandleAuthCallback`)

1. Validate state from query params; remove from store
2. `token, err := oauthConfig.Exchange(ctx, code)`
3. Build minimal Credentials, fetch user email via `oauth2.UserInfo` endpoint
4. Persist via `credstore.Store(email, token)`
5. Return success HTML page

### `GetCredentials(ctx, email, requiredScopes) (*oauth2.Token, error)`

1. Look up token in credstore by email
2. If expired and refresh token present, refresh via `tokenSource.Token()` — token source is `oauthConfig.TokenSource(ctx, existingToken)` which auto-refreshes
3. Persist refreshed token (atomic write)
4. Check scopes (hierarchy-aware via `HasRequiredScopes`)
5. Return token (caller wraps in `option.WithTokenSource(...)`)

### Minimal callback HTTP server (stdio mode)

`internal/auth/callback.go`:

```go
type CallbackServer struct {
    port int
    base string
    srv  *http.Server
    once sync.Once
}

func EnsureCallbackAvailable(transport string, port int, base string) error {
    if transport == "streamable-http" {
        return nil  // main HTTP server handles it
    }
    // Start a goroutine'd net/http on (host, port) with /oauth2callback handler
}

func Cleanup() { ... }
```

For streamable-HTTP mode, the routes are registered on the main mcp-go HTTP mux instead.

---

## 7. Credential store

```go
// internal/auth/credstore.go
type CredentialStore interface {
    Get(email string) (*oauth2.Token, error)
    Store(email string, token *oauth2.Token) error
    Delete(email string) error
    List() ([]string, error)
}

// LocalDirectoryStore writes one JSON file per user under cfg.CredentialsDir.
type LocalDirectoryStore struct {
    baseDir string
}
```

- One `<url-encoded-email>.json` file per user, mode `0600`
- `_resolve_credential_path` equivalent: `filepath.Clean` + check `strings.HasPrefix(resolved, baseDir+string(os.PathSeparator))`
- Atomic write: write to temp file, `os.Rename`
- `List()` filters out non-`@`-containing filenames

Singleton retrieved via `GetCredentialStore()` so swapping in a GCS impl later is a one-line change.

---

## 8. Scopes & hierarchy

`internal/auth/scopes.go` mirrors `auth/scopes.py`:

```go
const (
    UserinfoEmailScope    = "https://www.googleapis.com/auth/userinfo.email"
    UserinfoProfileScope  = "https://www.googleapis.com/auth/userinfo.profile"
    OpenIDScope           = "openid"
    GmailReadonlyScope    = "https://www.googleapis.com/auth/gmail.readonly"
    GmailSendScope        = "https://www.googleapis.com/auth/gmail.send"
    GmailComposeScope     = "https://www.googleapis.com/auth/gmail.compose"
    GmailModifyScope      = "https://www.googleapis.com/auth/gmail.modify"
    GmailLabelsScope      = "https://www.googleapis.com/auth/gmail.labels"
    GmailSettingsScope    = "https://www.googleapis.com/auth/gmail.settings.basic"
)

var BaseScopes = []string{UserinfoEmailScope, UserinfoProfileScope, OpenIDScope}

// ScopeHierarchy: broader scopes that cover narrower ones.
var ScopeHierarchy = map[string][]string{
    GmailModifyScope: {GmailReadonlyScope, GmailSendScope, GmailComposeScope, GmailLabelsScope},
    // (Other services left empty in skeleton; structure here for additions.)
}

func HasRequiredScopes(available, required []string) bool { ... }

var ToolScopesMap = map[string][]string{
    "gmail": {GmailReadonlyScope, GmailSendScope, GmailComposeScope, GmailModifyScope, GmailLabelsScope, GmailSettingsScope},
}
```

`ScopesForTools(enabled []string) []string` returns `BaseScopes ∪ ToolScopesMap[svc] for svc in enabled`. This is what `StartAuthFlow` requests.

---

## 9. Granular permissions (no-op scaffolding)

`internal/auth/permissions.go` parses `--permissions gmail:organize drive:readonly` syntax into `map[string]string`, validates against a `ServicePermissionLevels` table (gmail levels: readonly, organize, drafts, send, full), and stores the parsed result on `Config`. **The filter pass does not actually remove tools** — that's noted as a TODO in the file with a one-line link to the Python implementation.

This satisfies the user's "include them as no-ops" requirement: the parsing/validation/CLI surface is real, the enforcement is stubbed.

---

## 10. Transport selection

```go
// internal/core/transport.go
func Run(ctx context.Context, srv *server.MCPServer, cfg *Config) error {
    switch cfg.Transport {
    case "stdio":
        if err := auth.EnsureCallbackAvailable("stdio", cfg.Port, cfg.BaseURI); err != nil {
            return err
        }
        defer auth.Cleanup()
        return server.ServeStdio(srv)

    case "streamable-http":
        // Build HTTP mux with /health, /oauth2callback, MCP routes
        mux := http.NewServeMux()
        mux.HandleFunc("/health", healthHandler(cfg))
        mux.HandleFunc("/oauth2callback", auth.HTTPCallbackHandler())
        // mcp-go provides a streamable-HTTP handler; mount at /mcp
        sseSrv := server.NewStreamableHTTPServer(srv)
        mux.Handle("/mcp", sseSrv)
        return http.ListenAndServe(net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), mux)

    default:
        return fmt.Errorf("unknown transport: %s", cfg.Transport)
    }
}
```

**Both transports share** the OAuth callback handler — only the dispatch differs.

---

## 11. Error handling — `apierror.Wrap`

Mirrors Python `@handle_http_errors`:

```go
// internal/core/apierror/handler.go
func Wrap(toolName string, err error) *mcp.CallToolResult {
    var gerr *googleapi.Error
    if errors.As(err, &gerr) {
        switch gerr.Code {
        case 403:
            if isAccessNotConfigured(gerr) {
                return mcp.NewToolResultText(apiEnablementMessage(toolName, gerr))
            }
            fallthrough
        case 401:
            return mcp.NewToolResultText(authRequiredMessage(toolName, gerr))
        default:
            return mcp.NewToolResultError(fmt.Sprintf("API error in %s: %s", toolName, gerr.Message))
        }
    }
    return mcp.NewToolResultError(fmt.Sprintf("Unexpected error in %s: %v", toolName, err))
}
```

The `apiEnablementMessage` and `authRequiredMessage` functions return the exact LLM-targeted prose from the Python codebase (with `**LLM:**` markers, "ACTION REQUIRED" headers, etc.).

---

## 12. CLI surface

```
workspace-mcp [flags]

Flags:
  --transport string         "stdio" (default) or "streamable-http"
  --single-user              Bypass session→user mapping; use any cred from store
  --tools strings            Comma-separated services (only "gmail" valid in skeleton)
  --tool-tier string         "core", "extended", or "complete"
  --permissions strings      Per-service levels, e.g. "gmail:organize"
  --read-only                Stubbed; flag accepted but no enforcement (logs warning)

Env vars (all optional, override CLI when CLI absent):
  GOOGLE_OAUTH_CLIENT_ID         (required for any OAuth flow)
  GOOGLE_OAUTH_CLIENT_SECRET     (required for confidential client; optional for public client)
  GOOGLE_OAUTH_REDIRECT_URI      (default: http://localhost:8000/oauth2callback)
  USER_GOOGLE_EMAIL              (default email for single-user)
  WORKSPACE_MCP_PORT / PORT      (default 8000)
  WORKSPACE_MCP_BASE_URI         (default "http://localhost")
  WORKSPACE_MCP_HOST             (default "0.0.0.0")
  WORKSPACE_EXTERNAL_URL         (reverse-proxy override)
  WORKSPACE_MCP_CREDENTIALS_DIR  (default: ~/.google_workspace_mcp/credentials)
  WORKSPACE_MCP_TRANSPORT        (env equivalent of --transport)
  WORKSPACE_MCP_TOOLS            (env equivalent of --tools)
  WORKSPACE_MCP_TOOL_TIER        (env equivalent of --tool-tier)
  WORKSPACE_MCP_PERMISSIONS      (env equivalent of --permissions)
  MCP_SINGLE_USER_MODE           (env equivalent of --single-user)
  OAUTHLIB_INSECURE_TRANSPORT    (allow http:// redirect for dev)
```

Mutual exclusion enforced in `Config.Load()`.

---

## 13. Tool tier YAML

`internal/core/tooltier/tiers.yaml` (embedded via `//go:embed`):

```yaml
gmail:
  core:
    - search_gmail_messages
    - get_gmail_message_content
    - get_gmail_messages_content_batch
    - send_gmail_message
    - list_gmail_labels
  extended: []
  complete: []
```

Just Gmail. When other services are added, new entries land here.

`Loader.ResolveToolsFromTier(tier, services []string) ([]string, []string)` returns `(toolNames, serviceNames)` — same signature shape as the Python `resolve_tools_from_tier`.

---

## 14. End-to-end: a single tool call

```
1. MCP client sends:
   {"method":"tools/call","params":{"name":"search_gmail_messages",
    "arguments":{"query":"is:unread","user_google_email":"alice@example.com"}}}

2. mcp-go decodes, dispatches to the registered handler — which is
   the closure returned by RequireGmailService.

3. RequireGmailService closure:
   a. BindArguments(&SearchArgs{}) → args populated
   b. resolveUserEmail(ctx, req) → "alice@example.com"
   c. GetCredentials(ctx, "alice@example.com", [gmail.readonly])
      → loads from credstore; refreshes if expired; checks scopes
   d. gmail.NewService(ctx, option.WithTokenSource(...))
   e. Call searchGmailMessages(ctx, svc, "alice@example.com", args)
      → svc.Users.Messages.List("me").Q("is:unread").Do()
      → format response into a string
   f. Return mcp.NewToolResultText(result)

4. If GetCredentials returns NoCredentials:
   → Closure returns "**ACTION REQUIRED: Google Authentication Needed...**"
     with the OAuth URL the user must visit. The MCP client renders this.

5. If the API returns 403 accessNotConfigured:
   → apierror.Wrap returns the "Enable this API" message with the Cloud
     Console link, plus an `IMPORTANT - LLM:` instruction to surface it.

6. Cleanup: gmail.Service has no Close(); the http.Client behind it is GC'd.
```

---

## 15. Testing strategy (skeleton scope)

- **Unit tests** for `auth/scopes.go` (`HasRequiredScopes` hierarchy correctness — port the Python `test_scopes.py`)
- **Unit tests** for `auth/permissions.go` parser (port the Python `test_permissions.py`)
- **Unit tests** for `core/tooltier/loader.go` (tier resolution, service backfill)
- **Unit tests** for `core/config.go` mutual-exclusion validation
- **Integration tests** for Gmail tools are out of scope for the skeleton (require live credentials); placeholder `_test.go` files documenting the expected shape are added.

`go test ./...` should pass with no Google credentials.

---

## 16. README highlights

The `README.md` will cover:

- What this is + scope (link to this design doc)
- Setup: create OAuth client, set env vars, `go install ./cmd/workspace-mcp`
- Run: `workspace-mcp` (stdio) or `workspace-mcp --transport streamable-http`
- Sample MCP client config (Claude Desktop snippet)
- "Adding a new service" — step-by-step pointing to where each piece lives
- Differences from Python version

---

## 17. Out-of-scope hooks (where to add things later)

| Future feature | Where it slots in |
|---|---|
| Drive/Calendar/etc. service | New `internal/<service>/` package; add to `tools/registry.go`; extend `ToolScopesMap` and `tiers.yaml` |
| OAuth 2.1 multi-user | Branch in `auth/service.go:RequireGmailService` based on `cfg.OAuth21Enabled`; new auth provider impl |
| GCS credential store | New `auth/credstore_gcs.go` implementing `CredentialStore`; `GetCredentialStore()` factory branches on env |
| Read-only mode | Real implementation of `core/server.go:FilterTools` read-only branch, using already-collected `RequiredScopes` |
| Granular permissions enforcement | Real implementation of the permissions branch in `FilterTools` |
| Stateless mode | Skip `credstore.Store` calls; in-memory only |
| Service-account mode | Branch in `RequireGmailService` to use `google.JWTConfig` instead of OAuth flow |
| `workspace-cli` | New `cmd/workspace-cli/` with `mcp-go` client + persistent token storage |
| Helm chart / Dockerfile | Add at repo root; no code changes needed |

---

## 18. Open questions

None at design time. All scoping decisions are settled:

1. Inventory: 5 Gmail tools (search/get/batch/send/list_labels) ✓
2. Both transports (stdio + streamable-HTTP) ✓
3. Granular permissions: parsed but no-op enforcement ✓
4. mark3labs/mcp-go as the MCP library ✓
5. Idiomatic Go layout with `cmd/` + `internal/` ✓
6. OAuth 2.0 only ✓
7. Local-directory credential store only ✓

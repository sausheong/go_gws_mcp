# go-gws-mcp Skeleton + Gmail Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the architectural skeleton of [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp) from Python to Go, with the Gmail service as the proof-of-concept (5 core tools), supporting both stdio and streamable-HTTP transports, OAuth 2.0 only.

**Architecture:** Idiomatic Go layout (`cmd/` + `internal/`). The Python `@require_google_service` decorator is reproduced as a Go generic higher-order function `RequireGmailService[T]` that wraps `mcp.ToolHandlerFunc`. Tool registration is tracked centrally so a post-hoc filter pass (tier-based, scope-aware) can run after all tools are registered. Configuration is a singleton built from env vars + flags with mutual-exclusion validation that fails loud at startup.

**Tech Stack:** Go 1.22+, [`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go), `google.golang.org/api/gmail/v1`, `golang.org/x/oauth2`, `gopkg.in/yaml.v3`, stdlib `log/slog`, `flag`, `net/http`.

**Reference:** Design doc at `docs/design.md`. Source Python project lives at `~/projects/google_workspace_mcp`.

---

## Notes for the implementer

- **Working directory:** `~/projects/go_gws_mcp`
- **mcp-go API caveats:** The exact shape of `mark3labs/mcp-go`'s API has evolved. The plan uses API names from the v0.x line; if a name has changed (e.g. `BindArguments` vs `GetArguments`), use the current name and adjust. The plan's *intent* is canonical, the API names are guidance.
- **TDD discipline:** For each task, write the test first, run it red, implement, run it green, commit. Some tasks (HTTP server, OAuth flow with browser) can't be unit-tested cleanly — those use focused unit tests on the testable parts plus a smoke-test step.
- **Commit message format:** Conventional commits — `feat:`, `test:`, `chore:`, `docs:`, `refactor:`.
- **Don't run integration tests against real Google APIs** — the skeleton ships with placeholder integration test files that document expected shapes but don't run by default.

---

## Task 1: Project init

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `README.md` (placeholder)

- [ ] **Step 1: Initialize the Go module**

```bash
cd ~/projects/go_gws_mcp
go mod init github.com/sausheong/go_gws_mcp
```

Expected: creates `go.mod` with module path and `go 1.22` (or current).

- [ ] **Step 2: Create .gitignore**

Write `~/projects/go_gws_mcp/.gitignore`:

```
# Binaries
/workspace-mcp
/dist/

# Test artifacts
/coverage.out
/coverage.html

# Local credentials
.env
client_secret.json
.workspace-mcp/

# Editor
.idea/
.vscode/
*.swp

# OS
.DS_Store
```

- [ ] **Step 3: Create placeholder README**

Write `~/projects/go_gws_mcp/README.md`:

```markdown
# go-gws-mcp

Go port (skeleton + Gmail) of [google_workspace_mcp](https://github.com/taylorwilsdon/google_workspace_mcp).

See `docs/design.md` for architecture and `docs/plans/` for the implementation plan.
```

- [ ] **Step 4: Init git and commit**

```bash
cd ~/projects/go_gws_mcp
git init -b main
git add .gitignore README.md go.mod docs/
git commit -m "chore: initialize Go module and project scaffold"
```

Expected: clean initial commit with no Go code yet.

---

## Task 2: Scopes constants and hierarchy

**Files:**
- Create: `internal/auth/scopes.go`
- Test: `internal/auth/scopes_test.go`

- [ ] **Step 1: Write the failing test**

Write `~/projects/go_gws_mcp/internal/auth/scopes_test.go`:

```go
package auth

import (
	"sort"
	"testing"
)

func TestHasRequiredScopes_DirectMatch(t *testing.T) {
	if !HasRequiredScopes([]string{GmailReadonlyScope}, []string{GmailReadonlyScope}) {
		t.Fatal("direct match should succeed")
	}
}

func TestHasRequiredScopes_MultipleRequiredAllPresent(t *testing.T) {
	avail := []string{GmailReadonlyScope, GmailSendScope}
	req := []string{GmailReadonlyScope, GmailSendScope}
	if !HasRequiredScopes(avail, req) {
		t.Fatal("all required scopes present should succeed")
	}
}

func TestHasRequiredScopes_MissingScopeFails(t *testing.T) {
	if HasRequiredScopes([]string{GmailReadonlyScope}, []string{GmailSendScope}) {
		t.Fatal("missing scope should fail")
	}
}

func TestHasRequiredScopes_ModifyCoversReadonly(t *testing.T) {
	if !HasRequiredScopes([]string{GmailModifyScope}, []string{GmailReadonlyScope}) {
		t.Fatal("gmail.modify should cover gmail.readonly via hierarchy")
	}
}

func TestHasRequiredScopes_ModifyCoversSendComposeLabels(t *testing.T) {
	avail := []string{GmailModifyScope}
	req := []string{GmailSendScope, GmailComposeScope, GmailLabelsScope}
	if !HasRequiredScopes(avail, req) {
		t.Fatal("gmail.modify should cover send/compose/labels")
	}
}

func TestHasRequiredScopes_NarrowDoesNotCoverBroad(t *testing.T) {
	if HasRequiredScopes([]string{GmailReadonlyScope}, []string{GmailModifyScope}) {
		t.Fatal("readonly should not cover modify")
	}
}

func TestScopesForTools_IncludesBaseAndPerService(t *testing.T) {
	got := ScopesForTools([]string{"gmail"})
	sort.Strings(got)
	want := []string{
		GmailComposeScope, GmailLabelsScope, GmailModifyScope,
		GmailReadonlyScope, GmailSendScope, GmailSettingsScope,
		OpenIDScope, UserinfoEmailScope, UserinfoProfileScope,
	}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("got %d scopes, want %d: got=%v", len(got), len(want), got)
	}
	for i, s := range want {
		if got[i] != s {
			t.Fatalf("scope mismatch at %d: got %s, want %s", i, got[i], s)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestHasRequiredScopes|TestScopesForTools" -v
```

Expected: build failure with `undefined: HasRequiredScopes`, `undefined: GmailReadonlyScope`, etc.

- [ ] **Step 3: Implement `internal/auth/scopes.go`**

```go
// Package auth provides OAuth 2.0 flow, credential storage, and scope helpers
// for Google Workspace MCP tools.
package auth

// Identity scopes — always requested.
const (
	UserinfoEmailScope   = "https://www.googleapis.com/auth/userinfo.email"
	UserinfoProfileScope = "https://www.googleapis.com/auth/userinfo.profile"
	OpenIDScope          = "openid"
)

// Gmail scopes.
const (
	GmailReadonlyScope = "https://www.googleapis.com/auth/gmail.readonly"
	GmailSendScope     = "https://www.googleapis.com/auth/gmail.send"
	GmailComposeScope  = "https://www.googleapis.com/auth/gmail.compose"
	GmailModifyScope   = "https://www.googleapis.com/auth/gmail.modify"
	GmailLabelsScope   = "https://www.googleapis.com/auth/gmail.labels"
	GmailSettingsScope = "https://www.googleapis.com/auth/gmail.settings.basic"
)

// BaseScopes are required for user identification on every OAuth flow.
var BaseScopes = []string{UserinfoEmailScope, UserinfoProfileScope, OpenIDScope}

// ScopeHierarchy maps broader scopes to the narrower scopes they cover.
// See https://developers.google.com/gmail/api/auth/scopes.
var ScopeHierarchy = map[string][]string{
	GmailModifyScope: {GmailReadonlyScope, GmailSendScope, GmailComposeScope, GmailLabelsScope},
}

// ToolScopesMap is the full scope set per service.
var ToolScopesMap = map[string][]string{
	"gmail": {
		GmailReadonlyScope, GmailSendScope, GmailComposeScope,
		GmailModifyScope, GmailLabelsScope, GmailSettingsScope,
	},
}

// HasRequiredScopes reports whether `available` satisfies all of `required`,
// expanding the available set with implied narrower scopes from ScopeHierarchy.
func HasRequiredScopes(available, required []string) bool {
	expanded := make(map[string]struct{}, len(available)*2)
	for _, s := range available {
		expanded[s] = struct{}{}
		for _, narrow := range ScopeHierarchy[s] {
			expanded[narrow] = struct{}{}
		}
	}
	for _, s := range required {
		if _, ok := expanded[s]; !ok {
			return false
		}
	}
	return true
}

// ScopesForTools returns the union of BaseScopes and the per-service scopes
// for each enabled tool. Returns deduplicated list.
func ScopesForTools(enabled []string) []string {
	set := make(map[string]struct{}, len(BaseScopes)*2)
	for _, s := range BaseScopes {
		set[s] = struct{}{}
	}
	for _, tool := range enabled {
		for _, s := range ToolScopesMap[tool] {
			set[s] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestHasRequiredScopes|TestScopesForTools" -v
```

Expected: all 7 tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/auth/scopes.go internal/auth/scopes_test.go
git commit -m "feat(auth): add scope constants, hierarchy, and HasRequiredScopes"
```

---

## Task 3: Granular permissions parser

**Files:**
- Create: `internal/auth/permissions.go`
- Test: `internal/auth/permissions_test.go`

- [ ] **Step 1: Write the failing test**

Write `~/projects/go_gws_mcp/internal/auth/permissions_test.go`:

```go
package auth

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParsePermissions_ValidSingle(t *testing.T) {
	got, err := ParsePermissions([]string{"gmail:organize"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{"gmail": "organize"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParsePermissions_ValidMultiple(t *testing.T) {
	got, err := ParsePermissions([]string{"gmail:send", "drive:readonly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["gmail"] != "send" || got["drive"] != "readonly" {
		t.Fatalf("got %v", got)
	}
}

func TestParsePermissions_BadFormat(t *testing.T) {
	_, err := ParsePermissions([]string{"gmail-organize"})
	if err == nil || !strings.Contains(err.Error(), "service:level") {
		t.Fatalf("want format error, got %v", err)
	}
}

func TestParsePermissions_UnknownService(t *testing.T) {
	_, err := ParsePermissions([]string{"unknownsvc:readonly"})
	if err == nil || !strings.Contains(err.Error(), "Unknown service") {
		t.Fatalf("want unknown service error, got %v", err)
	}
}

func TestParsePermissions_UnknownLevel(t *testing.T) {
	_, err := ParsePermissions([]string{"gmail:bogus"})
	if err == nil || !strings.Contains(err.Error(), "Unknown level") {
		t.Fatalf("want unknown level error, got %v", err)
	}
}

func TestParsePermissions_DuplicateService(t *testing.T) {
	_, err := ParsePermissions([]string{"gmail:send", "gmail:readonly"})
	if err == nil || !strings.Contains(err.Error(), "Duplicate") {
		t.Fatalf("want duplicate error, got %v", err)
	}
}

func TestScopesForPermission_GmailOrganize(t *testing.T) {
	got, err := ScopesForPermission("gmail", "organize")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(got)
	want := []string{GmailLabelsScope, GmailModifyScope, GmailReadonlyScope}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestScopesForPermission_GmailSendIsCumulative(t *testing.T) {
	got, _ := ScopesForPermission("gmail", "send")
	gotSet := make(map[string]bool)
	for _, s := range got {
		gotSet[s] = true
	}
	for _, must := range []string{GmailReadonlyScope, GmailLabelsScope, GmailModifyScope, GmailComposeScope, GmailSendScope} {
		if !gotSet[must] {
			t.Errorf("gmail:send missing cumulative scope %s", must)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestParsePermissions|TestScopesForPermission" -v
```

Expected: build failure with `undefined: ParsePermissions`, etc.

- [ ] **Step 3: Implement `internal/auth/permissions.go`**

```go
package auth

import (
	"fmt"
	"sort"
	"strings"
)

// permissionLevel pairs a level name with the additional scopes it adds.
type permissionLevel struct {
	name   string
	scopes []string
}

// servicePermissionLevels holds ordered, cumulative permission levels per service.
// Skeleton scope: only Gmail is fully populated; other services are placeholders
// to demonstrate the structure (extend when adding services).
var servicePermissionLevels = map[string][]permissionLevel{
	"gmail": {
		{"readonly", []string{GmailReadonlyScope}},
		{"organize", []string{GmailLabelsScope, GmailModifyScope}},
		{"drafts", []string{GmailComposeScope}},
		{"send", []string{GmailSendScope}},
		{"full", []string{GmailSettingsScope}},
	},
	"drive":    {{"readonly", nil}, {"full", nil}},
	"calendar": {{"readonly", nil}, {"full", nil}},
	"docs":     {{"readonly", nil}, {"full", nil}},
	"sheets":   {{"readonly", nil}, {"full", nil}},
	"chat":     {{"readonly", nil}, {"full", nil}},
	"forms":    {{"readonly", nil}, {"full", nil}},
	"slides":   {{"readonly", nil}, {"full", nil}},
	"tasks":    {{"readonly", nil}, {"manage", nil}, {"full", nil}},
	"contacts": {{"readonly", nil}, {"full", nil}},
}

// ParsePermissions parses ["service:level", ...] entries.
// Returns map[service]level, or descriptive error.
func ParsePermissions(entries []string) (map[string]string, error) {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid permission format: %q. Expected 'service:level' (e.g., 'gmail:organize')", entry)
		}
		svc, level := parts[0], parts[1]
		if _, dup := result[svc]; dup {
			return nil, fmt.Errorf("Duplicate service in permissions: %q", svc)
		}
		levels, ok := servicePermissionLevels[svc]
		if !ok {
			valid := make([]string, 0, len(servicePermissionLevels))
			for s := range servicePermissionLevels {
				valid = append(valid, s)
			}
			sort.Strings(valid)
			return nil, fmt.Errorf("Unknown service: %q. Valid: %v", svc, valid)
		}
		valid := make([]string, 0, len(levels))
		found := false
		for _, l := range levels {
			valid = append(valid, l.name)
			if l.name == level {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("Unknown level %q for service %q. Valid: %v", level, svc, valid)
		}
		result[svc] = level
	}
	return result, nil
}

// ScopesForPermission returns cumulative scopes for service at the given level.
func ScopesForPermission(service, level string) ([]string, error) {
	levels, ok := servicePermissionLevels[service]
	if !ok {
		return nil, fmt.Errorf("Unknown service: %q", service)
	}
	cumulative := make(map[string]struct{})
	for _, l := range levels {
		for _, s := range l.scopes {
			cumulative[s] = struct{}{}
		}
		if l.name == level {
			out := make([]string, 0, len(cumulative))
			for s := range cumulative {
				out = append(out, s)
			}
			return out, nil
		}
	}
	valid := make([]string, 0, len(levels))
	for _, l := range levels {
		valid = append(valid, l.name)
	}
	return nil, fmt.Errorf("Unknown level %q for service %q. Valid: %v", level, service, valid)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestParsePermissions|TestScopesForPermission" -v
```

Expected: all 8 tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/auth/permissions.go internal/auth/permissions_test.go
git commit -m "feat(auth): add granular permissions parser (no-op enforcement)"
```

---

## Task 4: Tool tier loader

**Files:**
- Create: `internal/core/tooltier/loader.go`
- Create: `internal/core/tooltier/tiers.yaml`
- Test: `internal/core/tooltier/loader_test.go`

- [ ] **Step 1: Add yaml.v3 dependency**

```bash
cd ~/projects/go_gws_mcp
go get gopkg.in/yaml.v3
```

- [ ] **Step 2: Write the failing test**

Write `~/projects/go_gws_mcp/internal/core/tooltier/loader_test.go`:

```go
package tooltier

import (
	"sort"
	"testing"
)

func TestResolveToolsFromTier_GmailCore(t *testing.T) {
	loader, err := New()
	if err != nil {
		t.Fatalf("loader init: %v", err)
	}
	tools, services, err := loader.ResolveToolsFromTier("core", []string{"gmail"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	wantTools := []string{
		"get_gmail_message_content",
		"get_gmail_messages_content_batch",
		"list_gmail_labels",
		"search_gmail_messages",
		"send_gmail_message",
	}
	sort.Strings(tools)
	sort.Strings(wantTools)
	if len(tools) != len(wantTools) {
		t.Fatalf("got %d tools (%v), want %d (%v)", len(tools), tools, len(wantTools), wantTools)
	}
	for i, want := range wantTools {
		if tools[i] != want {
			t.Errorf("tool[%d] = %s, want %s", i, tools[i], want)
		}
	}
	if len(services) != 1 || services[0] != "gmail" {
		t.Fatalf("services = %v, want [gmail]", services)
	}
}

func TestResolveToolsFromTier_ExtendedIsCumulative(t *testing.T) {
	loader, _ := New()
	tools, _, _ := loader.ResolveToolsFromTier("extended", []string{"gmail"})
	hasCore := false
	for _, t := range tools {
		if t == "search_gmail_messages" {
			hasCore = true
		}
	}
	if !hasCore {
		t.Fatal("extended should include core tools (cumulative)")
	}
}

func TestResolveToolsFromTier_UnknownTier(t *testing.T) {
	loader, _ := New()
	_, _, err := loader.ResolveToolsFromTier("bogus", []string{"gmail"})
	if err == nil {
		t.Fatal("expected error for unknown tier")
	}
}
```

- [ ] **Step 3: Create `internal/core/tooltier/tiers.yaml`**

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

- [ ] **Step 4: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/tooltier/... -v
```

Expected: build failure with `undefined: New`.

- [ ] **Step 5: Implement `internal/core/tooltier/loader.go`**

```go
// Package tooltier loads tool tier definitions from an embedded YAML file
// and resolves tier+services queries into concrete tool/service lists.
package tooltier

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed tiers.yaml
var tiersYAML []byte

// Tier name constants; these match the YAML keys.
const (
	TierCore     = "core"
	TierExtended = "extended"
	TierComplete = "complete"
)

var validTiers = []string{TierCore, TierExtended, TierComplete}

// Loader reads and resolves tier configuration.
type Loader struct {
	// services maps service name -> tier name -> tool names.
	services map[string]map[string][]string
}

// New parses the embedded tiers.yaml.
func New() (*Loader, error) {
	var raw map[string]map[string][]string
	if err := yaml.Unmarshal(tiersYAML, &raw); err != nil {
		return nil, fmt.Errorf("parse tiers.yaml: %w", err)
	}
	return &Loader{services: raw}, nil
}

// AvailableServices returns the sorted list of services defined in the YAML.
func (l *Loader) AvailableServices() []string {
	out := make([]string, 0, len(l.services))
	for s := range l.services {
		out = append(out, s)
	}
	return out
}

// ResolveToolsFromTier returns (toolNames, serviceNames) for tools at or below
// the given tier within the optional service filter. If services is nil/empty,
// all available services are considered.
func (l *Loader) ResolveToolsFromTier(tier string, services []string) ([]string, []string, error) {
	if !isValidTier(tier) {
		return nil, nil, fmt.Errorf("unknown tier %q (valid: %v)", tier, validTiers)
	}
	if len(services) == 0 {
		services = l.AvailableServices()
	}

	tierIdx := tierIndex(tier)
	seen := make(map[string]struct{})
	var toolsOut []string
	servicesOut := make(map[string]struct{})

	for _, svc := range services {
		svcTiers, ok := l.services[svc]
		if !ok {
			continue
		}
		for i := 0; i <= tierIdx; i++ {
			for _, tool := range svcTiers[validTiers[i]] {
				if _, dup := seen[tool]; dup {
					continue
				}
				seen[tool] = struct{}{}
				toolsOut = append(toolsOut, tool)
				servicesOut[svc] = struct{}{}
			}
		}
	}

	svcList := make([]string, 0, len(servicesOut))
	for s := range servicesOut {
		svcList = append(svcList, s)
	}
	return toolsOut, svcList, nil
}

func isValidTier(t string) bool {
	for _, v := range validTiers {
		if v == t {
			return true
		}
	}
	return false
}

func tierIndex(t string) int {
	for i, v := range validTiers {
		if v == t {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/tooltier/... -v
```

Expected: all 3 tests pass.

- [ ] **Step 7: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/core/tooltier/ go.mod go.sum
git commit -m "feat(tooltier): add YAML-backed tier loader with embedded gmail tiers"
```

---

## Task 5: Config singleton with mutual-exclusion validation

**Files:**
- Create: `internal/core/config.go`
- Test: `internal/core/config_test.go`

- [ ] **Step 1: Write the failing test**

Write `~/projects/go_gws_mcp/internal/core/config_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/... -run "TestConfig" -v
```

Expected: build failure with `undefined: Load`, `undefined: Config`.

- [ ] **Step 3: Implement `internal/core/config.go`**

```go
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
		Port:              8000,
		BaseURI:           "http://localhost",
		Host:              "0.0.0.0",
		Transport:         "stdio",
		CredentialsDir:    defaultCredentialsDir(),
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
			return nil, fmt.Errorf("invalid WORKSPACE_MCP_TRANSPORT %q (want stdio|streamable-http)", v)
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
			return nil, fmt.Errorf("invalid WORKSPACE_MCP_TOOL_TIER %q (want core|extended|complete)", v)
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/... -run "TestConfig" -v
```

Expected: all 5 tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/core/config.go internal/core/config_test.go
git commit -m "feat(core): add Config singleton with env parsing and mutex validation"
```

---

## Task 6: Local credential store

**Files:**
- Create: `internal/auth/credstore.go`
- Test: `internal/auth/credstore_test.go`

- [ ] **Step 1: Add oauth2 dependency**

```bash
cd ~/projects/go_gws_mcp
go get golang.org/x/oauth2
```

- [ ] **Step 2: Write the failing test**

Write `~/projects/go_gws_mcp/internal/auth/credstore_test.go`:

```go
package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestLocalDirectoryStore_StoreAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalDirectoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "atk",
		RefreshToken: "rtk",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Round(time.Second),
	}
	if err := store.Store("alice@example.com", tok); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "atk" || got.RefreshToken != "rtk" {
		t.Fatalf("token mismatch: %+v", got)
	}
}

func TestLocalDirectoryStore_GetMissingReturnsNilNoError(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	tok, err := store.Get("nobody@example.com")
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}
	if tok != nil {
		t.Fatal("want nil token for missing user")
	}
}

func TestLocalDirectoryStore_FileMode0600(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	store.Store("bob@example.com", &oauth2.Token{AccessToken: "x"})

	matches, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(matches) != 1 {
		t.Fatalf("want 1 file, got %v", matches)
	}
	info, _ := os.Stat(matches[0])
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("want 0600, got %o", info.Mode().Perm())
	}
}

func TestLocalDirectoryStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	store.Store("c@example.com", &oauth2.Token{AccessToken: "x"})
	if err := store.Delete("c@example.com"); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get("c@example.com")
	if got != nil {
		t.Fatal("expected gone after delete")
	}
}

func TestLocalDirectoryStore_List(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	store.Store("a@example.com", &oauth2.Token{AccessToken: "x"})
	store.Store("b@example.com", &oauth2.Token{AccessToken: "x"})
	users, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("want 2 users, got %v", users)
	}
}

func TestLocalDirectoryStore_RejectsPathTraversalEmail(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	err := store.Store("../../../etc/passwd", &oauth2.Token{AccessToken: "x"})
	if err == nil {
		t.Fatal("want rejection for traversal-style email")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestLocalDirectoryStore" -v
```

Expected: build failure with `undefined: NewLocalDirectoryStore`.

- [ ] **Step 4: Implement `internal/auth/credstore.go`**

```go
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
)

// CredentialStore is the interface for OAuth credential persistence.
// Implementations: LocalDirectoryStore (file-based). Future: GCS, Valkey.
type CredentialStore interface {
	Get(email string) (*oauth2.Token, error)
	Store(email string, token *oauth2.Token) error
	Delete(email string) error
	List() ([]string, error)
}

// LocalDirectoryStore writes one URL-encoded JSON file per user under baseDir.
type LocalDirectoryStore struct {
	baseDir string
}

// NewLocalDirectoryStore ensures baseDir exists with mode 0700 and returns a store.
func NewLocalDirectoryStore(baseDir string) (*LocalDirectoryStore, error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create credentials dir: %w", err)
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	return &LocalDirectoryStore{baseDir: abs}, nil
}

func (s *LocalDirectoryStore) credPath(email string) (string, error) {
	if strings.TrimSpace(email) == "" {
		return "", errors.New("email must not be empty")
	}
	safe := url.QueryEscape(email)
	p := filepath.Join(s.baseDir, safe+".json")
	resolved, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(resolved, s.baseDir+string(os.PathSeparator)) && resolved != s.baseDir {
		return "", fmt.Errorf("invalid credential path: %q", resolved)
	}
	return resolved, nil
}

func (s *LocalDirectoryStore) Get(email string) (*oauth2.Token, error) {
	p, err := s.credPath(email)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse token for %s: %w", email, err)
	}
	return &tok, nil
}

func (s *LocalDirectoryStore) Store(email string, token *oauth2.Token) error {
	p, err := s.credPath(email)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (s *LocalDirectoryStore) Delete(email string) error {
	p, err := s.credPath(email)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *LocalDirectoryStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		stem := strings.TrimSuffix(name, ".json")
		decoded, err := url.QueryUnescape(stem)
		if err != nil {
			continue
		}
		if !strings.Contains(decoded, "@") {
			continue
		}
		out = append(out, decoded)
	}
	return out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestLocalDirectoryStore" -v
```

Expected: all 6 tests pass.

- [ ] **Step 6: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/auth/credstore.go internal/auth/credstore_test.go go.mod go.sum
git commit -m "feat(auth): add LocalDirectoryStore credential backend"
```

---

## Task 7: PKCE helpers and OAuth state store

**Files:**
- Create: `internal/auth/oauth.go` (initial portion: PKCE + state store)
- Test: `internal/auth/oauth_test.go`

- [ ] **Step 1: Write the failing test**

Write `~/projects/go_gws_mcp/internal/auth/oauth_test.go`:

```go
package auth

import (
	"strings"
	"testing"
)

func TestGeneratePKCE_ProducesS256Pair(t *testing.T) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length %d out of RFC 7636 range", len(verifier))
	}
	if challenge == verifier {
		t.Error("challenge must not equal verifier (S256 must hash)")
	}
	if strings.ContainsAny(challenge, "+/=") {
		t.Errorf("challenge contains non-base64url chars: %s", challenge)
	}
}

func TestStateStore_StoreAndConsume(t *testing.T) {
	store := newStateStore()
	store.Store("state-abc", "verifier-xyz")
	v, ok := store.Consume("state-abc")
	if !ok || v != "verifier-xyz" {
		t.Fatalf("got (%q, %v), want (verifier-xyz, true)", v, ok)
	}
	// Second consume should fail (single-use).
	if _, ok := store.Consume("state-abc"); ok {
		t.Fatal("state should be single-use")
	}
}

func TestStateStore_ConsumeMissing(t *testing.T) {
	store := newStateStore()
	if _, ok := store.Consume("never-stored"); ok {
		t.Fatal("missing state should return ok=false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestGeneratePKCE|TestStateStore" -v
```

Expected: build failure with `undefined: generatePKCE`, `undefined: newStateStore`.

- [ ] **Step 3: Implement initial `internal/auth/oauth.go`**

```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
)

// generatePKCE returns (verifier, S256-challenge, error) per RFC 7636.
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// stateStore is a process-local map of OAuth state -> PKCE verifier.
// State is single-use; Consume removes the entry.
type stateStore struct {
	mu sync.Mutex
	m  map[string]string
}

func newStateStore() *stateStore {
	return &stateStore{m: make(map[string]string)}
}

func (s *stateStore) Store(state, verifier string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[state] = verifier
}

func (s *stateStore) Consume(state string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[state]
	if ok {
		delete(s.m, state)
	}
	return v, ok
}

// globalStateStore is the package-level singleton used by StartAuthFlow / HandleAuthCallback.
var globalStateStore = newStateStore()
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/auth/... -run "TestGeneratePKCE|TestStateStore" -v
```

Expected: all 3 tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/auth/oauth.go internal/auth/oauth_test.go
git commit -m "feat(auth): add PKCE generator and OAuth state store"
```

---

## Task 8: OAuth flow (StartAuthFlow, HandleAuthCallback, GetCredentials)

**Files:**
- Modify: `internal/auth/oauth.go` (append flow functions)

This task does **not** add new unit tests — the OAuth flow requires a live Google endpoint and a browser. We'll smoke-test in Task 24. The functions are still small and direct.

- [ ] **Step 1: Add Google oauth2 endpoint dependency**

```bash
cd ~/projects/go_gws_mcp
go get golang.org/x/oauth2/google
```

- [ ] **Step 2: Replace the import block at the top of `internal/auth/oauth.go`**

Open `internal/auth/oauth.go` and replace its existing `import (...)` block (which currently lists only the four imports from Task 7) with this expanded block:

```go
import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)
```

Leave everything else from Task 7 (the `generatePKCE`, `stateStore`, `globalStateStore` definitions) in place.

- [ ] **Step 2b: Append the OAuth flow functions to the end of `internal/auth/oauth.go`**

Append the following block after the existing Task 7 code:

```go
// AuthRequiredError signals the caller (typically a tool wrapper) that the
// user must complete an OAuth flow. Its Error() returns the LLM-targeted prose.
type AuthRequiredError struct {
	Message string
	AuthURL string
}

func (e *AuthRequiredError) Error() string { return e.Message }

// OAuthClient holds the configured OAuth client + dependencies needed by the flow.
type OAuthClient struct {
	Config       *oauth2.Config
	Store        CredentialStore
	StatePersist string // optional: path to oauth_states.json (skeleton: in-memory only)
}

// NewOAuthClient builds an oauth2.Config from server config + scopes.
func NewOAuthClient(clientID, clientSecret, redirectURI string, scopes []string, store CredentialStore) *OAuthClient {
	return &OAuthClient{
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
		},
		Store: store,
	}
}

// StartAuthFlow returns an LLM-targeted "ACTION REQUIRED" message containing
// the auth URL the user must visit. Stores PKCE state for callback validation.
func (c *OAuthClient) StartAuthFlow(userEmail, serviceName string) (*AuthRequiredError, error) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, err
	}
	globalStateStore.Store(state, verifier)

	authURL := c.Config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	display := serviceName
	if userEmail != "" {
		display = fmt.Sprintf("%s for '%s'", serviceName, userEmail)
	}

	msg := fmt.Sprintf(`**ACTION REQUIRED: Google Authentication Needed for %s**

To proceed, the user must authorize this application for %s access using all required permissions.
**LLM, please present this exact authorization URL to the user as a clickable hyperlink:**
Authorization URL: %s
Markdown for hyperlink: [Click here to authorize %s access](%s)

**LLM, after presenting the link, instruct the user as follows:**
1. Click the link and complete the authorization in their browser.
2. After successful authorization, **retry their original command**.

The application will use the new credentials. If '%s' was provided, it must match the authenticated account.`,
		display, serviceName, authURL, serviceName, authURL, userEmail)

	return &AuthRequiredError{Message: msg, AuthURL: authURL}, nil
}

// HandleAuthCallback exchanges the authorization code for tokens, fetches the
// authenticated user's email, persists the token, and returns (email, nil).
func (c *OAuthClient) HandleAuthCallback(ctx context.Context, state, code string) (string, error) {
	verifier, ok := globalStateStore.Consume(state)
	if !ok {
		return "", errors.New("invalid or expired OAuth state")
	}

	token, err := c.Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
	)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}

	email, err := fetchUserEmail(ctx, c.Config, token)
	if err != nil {
		return "", fmt.Errorf("fetch user email: %w", err)
	}

	if err := c.Store.Store(email, token); err != nil {
		return "", fmt.Errorf("persist token: %w", err)
	}
	return email, nil
}

// GetCredentials returns a refreshed *oauth2.Token for userEmail with at least
// the requested scopes (hierarchy-aware). Returns *AuthRequiredError when the
// user needs to authenticate.
func (c *OAuthClient) GetCredentials(ctx context.Context, userEmail string, requiredScopes []string) (*oauth2.Token, error) {
	if userEmail == "" {
		return nil, errors.New("user_google_email is required")
	}
	stored, err := c.Store.Get(userEmail)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, &AuthRequiredError{Message: fmt.Sprintf("No credentials for %s; user must authenticate", userEmail)}
	}

	// Refresh if needed via TokenSource (auto-refreshes when expired).
	src := c.Config.TokenSource(ctx, stored)
	fresh, err := src.Token()
	if err != nil {
		// Refresh failed (revoked / expired beyond refresh).
		_ = c.Store.Delete(userEmail)
		return nil, &AuthRequiredError{Message: fmt.Sprintf("Credentials for %s expired or revoked; user must re-authenticate. Underlying error: %v", userEmail, err)}
	}
	if fresh.AccessToken != stored.AccessToken {
		_ = c.Store.Store(userEmail, fresh) // persist rotated token
	}

	// Scope check via stored scopes (oauth2.Token doesn't carry scopes by default;
	// we approximate by checking the requested scopes in c.Config.Scopes).
	// In Python, scopes are stored alongside the token; we mirror that by
	// re-storing with extras. For the skeleton, we trust c.Config.Scopes and
	// expect callers (like RequireGmailService) to ensure the OAuth flow was
	// kicked off with sufficient scopes.
	if !HasRequiredScopes(c.Config.Scopes, requiredScopes) {
		return nil, &AuthRequiredError{Message: fmt.Sprintf("Token for %s lacks required scopes; user must re-authorize", userEmail)}
	}
	return fresh, nil
}

func fetchUserEmail(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (string, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo returned %d", resp.StatusCode)
	}
	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}
	if info.Email == "" {
		return "", errors.New("userinfo response missing email")
	}
	return info.Email, nil
}

```

**Note to implementer:** the `Token` returned by `oauth2.Config.Exchange` does not carry the granted scopes natively. The skeleton sidesteps this by checking against the requesting `Config.Scopes`. A future enhancement would persist the granted scope set alongside the token (Python does this via `credentials.granted_scopes`).

- [ ] **Step 3: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors. (Existing tests still pass.)

- [ ] **Step 4: Run all tests**

```bash
cd ~/projects/go_gws_mcp
go test ./...
```

Expected: PASS for all existing test files.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/auth/oauth.go go.mod go.sum
git commit -m "feat(auth): add OAuth 2.0 flow with PKCE and credential persistence"
```

---

## Task 9: Minimal OAuth callback HTTP server

**Files:**
- Create: `internal/auth/callback.go`

The callback server is hard to TDD without a full integration harness. We'll write it directly; smoke-test in Task 24.

- [ ] **Step 1: Implement `internal/auth/callback.go`**

```go
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// CallbackServer runs a minimal HTTP server with /oauth2callback for stdio mode.
// In streamable-HTTP mode the main HTTP mux handles the route directly.
type CallbackServer struct {
	port   int
	host   string
	client *OAuthClient

	srv      *http.Server
	once     sync.Once
	mu       sync.Mutex
	running  bool
}

// NewCallbackServer constructs a callback server bound to host:port.
func NewCallbackServer(host string, port int, client *OAuthClient) *CallbackServer {
	return &CallbackServer{host: host, port: port, client: client}
}

// Start listens in a goroutine and returns when the server is reachable, or error.
func (s *CallbackServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/oauth2callback", s.HTTPHandler())

	s.srv = &http.Server{
		Addr:    net.JoinHostPort(s.host, strconv.Itoa(s.port)),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("bind callback server: %w", err)
	}

	go func() {
		if err := s.srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("callback server error", "err", err)
		}
	}()
	s.running = true
	slog.Info("OAuth callback server listening", "addr", s.srv.Addr)
	return nil
}

// Stop gracefully shuts down the server.
func (s *CallbackServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || s.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
	s.running = false
}

// HTTPHandler is the OAuth callback HTTP handler; useful for embedding into the
// main HTTP mux in streamable-HTTP mode (instead of running a separate server).
func (s *CallbackServer) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errParam := q.Get("error"); errParam != "" {
			renderError(w, fmt.Sprintf("Google returned error: %s", errParam))
			return
		}
		state := q.Get("state")
		code := q.Get("code")
		if state == "" || code == "" {
			renderError(w, "Missing state or code in callback")
			return
		}

		email, err := s.client.HandleAuthCallback(r.Context(), state, code)
		if err != nil {
			renderError(w, err.Error())
			return
		}
		renderSuccess(w, email)
	}
}

func renderSuccess(w http.ResponseWriter, email string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Authentication Successful</title></head>
<body style="font-family:system-ui;max-width:480px;margin:64px auto;padding:24px;background:#f4f4f5">
<h2>✅ Authentication successful</h2>
<p>Authenticated as <strong>%s</strong>.</p>
<p>You can close this tab and return to your assistant.</p>
</body></html>`, htmlEscape(email))
}

func renderError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Authentication Failed</title></head>
<body style="font-family:system-ui;max-width:480px;margin:64px auto;padding:24px;background:#fee">
<h2>❌ Authentication failed</h2>
<pre style="white-space:pre-wrap">%s</pre>
</body></html>`, htmlEscape(msg))
}

func htmlEscape(s string) string {
	return url.PathEscape(s) // sufficient for our embedded plain strings; not full HTML-escape but safe
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/auth/callback.go
git commit -m "feat(auth): add minimal OAuth callback HTTP server"
```

---

## Task 10: Request-scoped context helpers

**Files:**
- Create: `internal/core/mcpcontext/context.go`
- Test: `internal/core/mcpcontext/context_test.go`

- [ ] **Step 1: Write the failing test**

Write `~/projects/go_gws_mcp/internal/core/mcpcontext/context_test.go`:

```go
package mcpcontext

import (
	"context"
	"testing"
)

func TestUserEmail_RoundTrip(t *testing.T) {
	ctx := WithUserEmail(context.Background(), "alice@example.com")
	got, ok := UserEmail(ctx)
	if !ok || got != "alice@example.com" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
}

func TestUserEmail_AbsentReturnsFalse(t *testing.T) {
	if _, ok := UserEmail(context.Background()); ok {
		t.Fatal("expected ok=false on bare context")
	}
}

func TestSessionID_RoundTrip(t *testing.T) {
	ctx := WithSessionID(context.Background(), "sess-123")
	got, ok := SessionID(ctx)
	if !ok || got != "sess-123" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/mcpcontext/... -v
```

Expected: build failure.

- [ ] **Step 3: Implement `internal/core/mcpcontext/context.go`**

```go
// Package mcpcontext provides typed accessors for request-scoped values
// (user email, session ID) attached to context.Context.
package mcpcontext

import "context"

type ctxKey int

const (
	keyUserEmail ctxKey = iota
	keySessionID
)

// WithUserEmail returns a context carrying the resolved user email.
func WithUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, keyUserEmail, email)
}

// UserEmail retrieves a previously-stored user email.
func UserEmail(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keyUserEmail).(string)
	return v, ok && v != ""
}

// WithSessionID returns a context carrying the MCP session ID.
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keySessionID, id)
}

// SessionID retrieves the MCP session ID.
func SessionID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keySessionID).(string)
	return v, ok && v != ""
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/mcpcontext/... -v
```

Expected: 3 tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/core/mcpcontext/
git commit -m "feat(core): add mcpcontext package for request-scoped values"
```

---

## Task 11: API error wrapper

**Files:**
- Create: `internal/core/apierror/handler.go`
- Test: `internal/core/apierror/handler_test.go`

- [ ] **Step 1: Add Google API client dependency**

```bash
cd ~/projects/go_gws_mcp
go get google.golang.org/api/googleapi
```

- [ ] **Step 2: Write the failing test**

Write `~/projects/go_gws_mcp/internal/core/apierror/handler_test.go`:

```go
package apierror

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestFormat_AccessNotConfigured(t *testing.T) {
	err := &googleapi.Error{
		Code:    403,
		Message: "Gmail API has not been used in project 12345 before...",
		Errors: []googleapi.ErrorItem{
			{Reason: "accessNotConfigured"},
		},
	}
	out := Format("search_gmail_messages", err)
	if !strings.Contains(out, "API is not enabled") {
		t.Errorf("expected enable-API hint, got: %s", out)
	}
	if !strings.Contains(out, "IMPORTANT - LLM:") {
		t.Errorf("expected LLM directive, got: %s", out)
	}
}

func TestFormat_AuthError401(t *testing.T) {
	err := &googleapi.Error{Code: 401, Message: "Invalid Credentials"}
	out := Format("get_gmail_message_content", err)
	if !strings.Contains(out, "re-authenticate") {
		t.Errorf("expected re-auth hint, got: %s", out)
	}
}

func TestFormat_GenericGoogleError(t *testing.T) {
	err := &googleapi.Error{Code: 400, Message: "Bad Request"}
	out := Format("send_gmail_message", err)
	if !strings.Contains(out, "API error in send_gmail_message") {
		t.Errorf("expected generic error format, got: %s", out)
	}
}

func TestFormat_NonGoogleError(t *testing.T) {
	out := Format("search_gmail_messages", errors.New("network down"))
	if !strings.Contains(out, "Unexpected error") {
		t.Errorf("expected unexpected wrapper, got: %s", out)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/apierror/... -v
```

Expected: build failure with `undefined: Format`.

- [ ] **Step 4: Implement `internal/core/apierror/handler.go`**

```go
// Package apierror wraps Google API errors with LLM-targeted messages that
// embed actionable recovery steps.
package apierror

import (
	"errors"
	"fmt"

	"google.golang.org/api/googleapi"
)

// Format returns a string suitable for an MCP tool result body. Mirrors
// Python's @handle_http_errors decorator behavior.
func Format(toolName string, err error) string {
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		switch gerr.Code {
		case 403:
			if isAccessNotConfigured(gerr) {
				return apiEnablementMessage(toolName, gerr)
			}
			return authErrorMessage(toolName, gerr)
		case 401:
			return authErrorMessage(toolName, gerr)
		default:
			return fmt.Sprintf("API error in %s: %s (HTTP %d)", toolName, gerr.Message, gerr.Code)
		}
	}
	return fmt.Sprintf("Unexpected error in %s: %v", toolName, err)
}

func isAccessNotConfigured(g *googleapi.Error) bool {
	for _, item := range g.Errors {
		if item.Reason == "accessNotConfigured" {
			return true
		}
	}
	return false
}

func apiEnablementMessage(toolName string, g *googleapi.Error) string {
	return fmt.Sprintf(
		`API error in %s: The required Google API is not enabled for your project.

Details: %s

Enable the API in the Google Cloud Console (search for the relevant API by name and click 'Enable').
After enabling, wait 1-2 minutes for the change to propagate, then retry.

IMPORTANT - LLM: instruct the user to enable the required API and surface the Cloud Console link as a clickable hyperlink.`,
		toolName, g.Message,
	)
}

func authErrorMessage(toolName string, g *googleapi.Error) string {
	return fmt.Sprintf(
		`API error in %s: %s (HTTP %d).
You might need to re-authenticate.
LLM: ask the user to re-run start_google_auth (or re-authenticate via their MCP client) and then retry the original command.`,
		toolName, g.Message, g.Code,
	)
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/apierror/... -v
```

Expected: 4 tests pass.

- [ ] **Step 6: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/core/apierror/ go.mod go.sum
git commit -m "feat(core): add apierror.Format for LLM-targeted Google API errors"
```

---

## Task 12: Tool registration tracking

**Files:**
- Create: `internal/core/server.go`
- Test: `internal/core/server_test.go`

- [ ] **Step 1: Add mcp-go dependency**

```bash
cd ~/projects/go_gws_mcp
go get github.com/mark3labs/mcp-go
```

- [ ] **Step 2: Write the failing test**

Write `~/projects/go_gws_mcp/internal/core/server_test.go`:

```go
package core

import (
	"reflect"
	"sort"
	"testing"
)

func TestRegistry_RecordsTools(t *testing.T) {
	r := NewRegistry()
	r.Record("search_gmail_messages", []string{"https://www.googleapis.com/auth/gmail.readonly"})
	r.Record("send_gmail_message", []string{"https://www.googleapis.com/auth/gmail.send"})

	names := r.Names()
	sort.Strings(names)
	want := []string{"search_gmail_messages", "send_gmail_message"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("got %v, want %v", names, want)
	}
}

func TestRegistry_FilterByEnabled_KeepsEnabledOnly(t *testing.T) {
	r := NewRegistry()
	r.Record("a", nil)
	r.Record("b", nil)
	r.Record("c", nil)

	enabled := map[string]struct{}{"a": {}, "c": {}}
	removed := r.computeRemovals(enabled, nil)
	sort.Strings(removed)
	if !reflect.DeepEqual(removed, []string{"b"}) {
		t.Fatalf("got removals %v, want [b]", removed)
	}
}

func TestRegistry_FilterByEnabled_NilMeansKeepAll(t *testing.T) {
	r := NewRegistry()
	r.Record("a", nil)
	r.Record("b", nil)
	removed := r.computeRemovals(nil, nil)
	if len(removed) != 0 {
		t.Fatalf("expected no removals, got %v", removed)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/... -run "TestRegistry" -v
```

Expected: build failure.

- [ ] **Step 4: Implement `internal/core/server.go`**

```go
package core

import (
	"log/slog"
)

// ToolMetadata records the OAuth scopes a registered tool requires.
type ToolMetadata struct {
	Name           string
	RequiredScopes []string
}

// Registry tracks tools registered with the MCP server so a post-hoc filter
// pass can remove tools that don't fit the active mode.
type Registry struct {
	tools []ToolMetadata
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Record adds metadata for a tool. Call from each service's Register* function
// alongside the actual srv.AddTool() call.
func (r *Registry) Record(name string, requiredScopes []string) {
	r.tools = append(r.tools, ToolMetadata{Name: name, RequiredScopes: requiredScopes})
}

// Names returns all recorded tool names.
func (r *Registry) Names() []string {
	out := make([]string, len(r.tools))
	for i, t := range r.tools {
		out[i] = t.Name
	}
	return out
}

// computeRemovals returns tool names that should be removed given an optional
// `enabled` allowlist (nil = keep all) and an optional allowed-scope set
// (nil = no scope filtering). A tool is removed if it's not in `enabled` OR
// any of its required scopes are absent from `allowedScopes`.
func (r *Registry) computeRemovals(enabled, allowedScopes map[string]struct{}) []string {
	var remove []string
	for _, t := range r.tools {
		if enabled != nil {
			if _, ok := enabled[t.Name]; !ok {
				remove = append(remove, t.Name)
				continue
			}
		}
		if allowedScopes != nil {
			for _, s := range t.RequiredScopes {
				if _, ok := allowedScopes[s]; !ok {
					remove = append(remove, t.Name)
					break
				}
			}
		}
	}
	return remove
}

// Filter logs the result of computeRemovals. Removal from the live mcp-go
// server is the responsibility of the integration code in cmd/workspace-mcp
// (mcp-go's removal API may evolve; the registry stays library-agnostic).
//
// In the skeleton, the filter pass logs intended removals when permissions
// mode is active (granular permissions enforcement is a no-op per design).
func (r *Registry) Filter(cfg *Config) []string {
	var enabled map[string]struct{}
	if len(cfg.EnabledTools) > 0 {
		enabled = make(map[string]struct{})
		// Skeleton: --tools selects services not individual tools. With only
		// gmail wired up, all gmail tools are kept. The hook stays here.
	}
	removals := r.computeRemovals(enabled, nil)
	if len(removals) > 0 {
		slog.Info("tool registry filter", "removed", removals)
	}
	if len(cfg.Permissions) > 0 {
		slog.Info("granular permissions parsed but enforcement is a no-op in the skeleton",
			"permissions", cfg.Permissions)
	}
	return removals
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/core/... -run "TestRegistry" -v
```

Expected: 3 tests pass.

- [ ] **Step 6: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/core/server.go internal/core/server_test.go go.mod go.sum
git commit -m "feat(core): add tool Registry with filter computation"
```

---

## Task 13: RequireGmailService — the decorator equivalent

**Files:**
- Create: `internal/auth/service.go`

This file ties together the OAuth client, error wrapping, and mcp-go's tool handler signature. Tests for the wrapper itself need a mock server and live OAuth — too heavy for unit tests. The behavior is exercised end-to-end in Task 24.

- [ ] **Step 1: Implement `internal/auth/service.go`**

```go
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/sausheong/go_gws_mcp/internal/core/apierror"
	"github.com/sausheong/go_gws_mcp/internal/core/mcpcontext"
)

// GmailHandler is the body shape every Gmail tool implementation has.
// `svc` and `userEmail` are injected; the body is pure logic.
type GmailHandler[T any] func(
	ctx context.Context,
	svc *gmailapi.Service,
	userEmail string,
	args T,
) (string, error)

// EmailExtractor pulls user_google_email from request arguments. We use it
// because mcp-go arguments come in as map[string]any and we need both the
// typed args struct (for the handler) and the email (before binding).
func extractUserEmail(req mcp.CallToolRequest, defaultEmail string) string {
	if req.Params.Arguments != nil {
		if m, ok := req.Params.Arguments.(map[string]any); ok {
			if v, ok := m["user_google_email"].(string); ok && v != "" {
				return v
			}
		}
	}
	return defaultEmail
}

// bindArgs deserializes req.Params.Arguments into T using a JSON round-trip.
// mcp-go's arguments are map[string]any; this is the simplest reliable path.
func bindArgs[T any](req mcp.CallToolRequest) (T, error) {
	var out T
	raw := req.Params.Arguments
	if raw == nil {
		return out, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

// RequireGmailService produces an mcp.ToolHandlerFunc that:
//   1. Binds args into T
//   2. Resolves user email (request -> ctx -> client default)
//   3. Loads + refreshes credentials via OAuthClient.GetCredentials
//   4. Builds gmail.Service
//   5. Calls handler(ctx, svc, userEmail, args)
//   6. Wraps Google API errors via apierror.Format
//
// On AuthRequiredError, returns the formatted instructions as a tool result.
// On other errors, returns an error result.
func RequireGmailService[T any](
	toolName string,
	requiredScopes []string,
	client *OAuthClient,
	defaultEmail string,
	handler GmailHandler[T],
) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[T](req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		userEmail := extractUserEmail(req, defaultEmail)
		if userEmail == "" {
			if v, ok := mcpcontext.UserEmail(ctx); ok {
				userEmail = v
			}
		}
		if userEmail == "" {
			return mcp.NewToolResultError("user_google_email is required"), nil
		}
		ctx = mcpcontext.WithUserEmail(ctx, userEmail)

		token, err := client.GetCredentials(ctx, userEmail, requiredScopes)
		if err != nil {
			var authErr *AuthRequiredError
			if errors.As(err, &authErr) {
				return mcp.NewToolResultText(authErr.Message), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("auth error: %v", err)), nil
		}

		ts := client.Config.TokenSource(ctx, token)
		svc, err := gmailapi.NewService(ctx, option.WithTokenSource(ts))
		if err != nil {
			return mcp.NewToolResultError(apierror.Format(toolName, err)), nil
		}

		result, err := handler(ctx, svc, userEmail, args)
		if err != nil {
			slog.Warn("tool handler error", "tool", toolName, "user", userEmail, "err", err)
			return mcp.NewToolResultText(apierror.Format(toolName, err)), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
```

- [ ] **Step 2: Add Google API Gmail dependency**

```bash
cd ~/projects/go_gws_mcp
go get google.golang.org/api/gmail/v1
go get google.golang.org/api/option
```

- [ ] **Step 3: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Run all tests**

```bash
cd ~/projects/go_gws_mcp
go test ./...
```

Expected: PASS for all existing tests.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/auth/service.go go.mod go.sum
git commit -m "feat(auth): add RequireGmailService generic wrapper for tool handlers"
```

---

## Task 14: Transport selection

**Files:**
- Create: `internal/core/transport.go`

- [ ] **Step 1: Implement `internal/core/transport.go`**

```go
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/mark3labs/mcp-go/server"

	"github.com/sausheong/go_gws_mcp/internal/auth"
)

// Transport names accepted by Run.
const (
	TransportStdio          = "stdio"
	TransportStreamableHTTP = "streamable-http"
)

// Run starts the MCP server on the configured transport and blocks until exit.
// In stdio mode, also starts the minimal OAuth callback HTTP server on
// cfg.Host:cfg.Port for the OAuth flow. In streamable-HTTP mode, the callback
// is mounted on the same HTTP mux as the MCP endpoint.
func Run(ctx context.Context, srv *server.MCPServer, cfg *Config, oauthClient *auth.OAuthClient) error {
	switch cfg.Transport {
	case TransportStdio:
		callback := auth.NewCallbackServer(callbackHost(cfg), cfg.Port, oauthClient)
		if err := callback.Start(); err != nil {
			return err
		}
		defer callback.Stop()
		return server.ServeStdio(srv)

	case TransportStreamableHTTP:
		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthHandler(cfg))
		callback := auth.NewCallbackServer(cfg.Host, cfg.Port, oauthClient)
		mux.Handle("/oauth2callback", callback.HTTPHandler())

		streamable := server.NewStreamableHTTPServer(srv)
		mux.Handle("/mcp", streamable)
		mux.Handle("/mcp/", streamable)

		addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
		httpSrv := &http.Server{Addr: addr, Handler: mux}
		go func() {
			<-ctx.Done()
			_ = httpSrv.Close()
		}()
		return httpSrv.ListenAndServe()

	default:
		return fmt.Errorf("unknown transport: %s", cfg.Transport)
	}
}

// callbackHost returns the host the stdio callback server should bind to,
// extracted from cfg.BaseURI (e.g. "http://localhost" -> "localhost").
func callbackHost(cfg *Config) string {
	// Strip scheme.
	uri := cfg.BaseURI
	for _, prefix := range []string{"https://", "http://"} {
		if len(uri) > len(prefix) && uri[:len(prefix)] == prefix {
			return uri[len(prefix):]
		}
	}
	return "localhost"
}

func healthHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "healthy",
			"service":   "go-gws-mcp",
			"transport": cfg.Transport,
		})
	}
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/core/transport.go
git commit -m "feat(core): add transport selection (stdio + streamable-http)"
```

---

## Task 15: Gmail helpers (body extraction, formatters)

**Files:**
- Create: `internal/gmail/helpers.go`
- Test: `internal/gmail/helpers_test.go`

- [ ] **Step 1: Write the failing test**

Write `~/projects/go_gws_mcp/internal/gmail/helpers_test.go`:

```go
package gmail

import (
	"encoding/base64"
	"strings"
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func TestExtractTextBody_PlainOnly(t *testing.T) {
	body := "Hello world"
	encoded := base64.URLEncoding.EncodeToString([]byte(body))
	payload := &gmailapi.MessagePart{
		MimeType: "text/plain",
		Body:     &gmailapi.MessagePartBody{Data: encoded},
	}
	got := ExtractTextBody(payload)
	if got != "Hello world" {
		t.Fatalf("got %q, want %q", got, body)
	}
}

func TestExtractTextBody_PrefersPlainOverHTML(t *testing.T) {
	plain := base64.URLEncoding.EncodeToString([]byte("plain body"))
	html := base64.URLEncoding.EncodeToString([]byte("<p>html</p>"))
	payload := &gmailapi.MessagePart{
		MimeType: "multipart/alternative",
		Parts: []*gmailapi.MessagePart{
			{MimeType: "text/html", Body: &gmailapi.MessagePartBody{Data: html}},
			{MimeType: "text/plain", Body: &gmailapi.MessagePartBody{Data: plain}},
		},
	}
	got := ExtractTextBody(payload)
	if got != "plain body" {
		t.Fatalf("got %q, want plain body", got)
	}
}

func TestExtractTextBody_FallsBackToHTML(t *testing.T) {
	html := base64.URLEncoding.EncodeToString([]byte("<p>only html</p>"))
	payload := &gmailapi.MessagePart{
		MimeType: "multipart/alternative",
		Parts: []*gmailapi.MessagePart{
			{MimeType: "text/html", Body: &gmailapi.MessagePartBody{Data: html}},
		},
	}
	got := ExtractTextBody(payload)
	if !strings.Contains(got, "only html") {
		t.Fatalf("got %q", got)
	}
}

func TestHeaderValue(t *testing.T) {
	headers := []*gmailapi.MessagePartHeader{
		{Name: "From", Value: "alice@example.com"},
		{Name: "Subject", Value: "Hi"},
	}
	if v := HeaderValue(headers, "From"); v != "alice@example.com" {
		t.Errorf("From = %q", v)
	}
	if v := HeaderValue(headers, "from"); v != "alice@example.com" {
		t.Errorf("case-insensitive lookup failed: %q", v)
	}
	if v := HeaderValue(headers, "Missing"); v != "" {
		t.Errorf("missing header should return empty: %q", v)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/gmail/... -v
```

Expected: build failure.

- [ ] **Step 3: Implement `internal/gmail/helpers.go`**

```go
// Package gmail implements Gmail MCP tools.
package gmail

import (
	"encoding/base64"
	"html"
	"regexp"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// ExtractTextBody walks a Gmail MessagePart tree and returns the best
// available textual body. Prefers text/plain, falls back to a stripped
// version of text/html.
func ExtractTextBody(payload *gmailapi.MessagePart) string {
	if payload == nil {
		return ""
	}
	plain, htmlBody := walkParts(payload)
	if plain != "" {
		return strings.TrimSpace(plain)
	}
	if htmlBody != "" {
		return stripHTML(htmlBody)
	}
	return ""
}

func walkParts(part *gmailapi.MessagePart) (plain, htmlBody string) {
	if part == nil {
		return "", ""
	}
	if part.Body != nil && part.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			switch part.MimeType {
			case "text/plain":
				if plain == "" {
					plain = string(decoded)
				}
			case "text/html":
				if htmlBody == "" {
					htmlBody = string(decoded)
				}
			}
		}
	}
	for _, p := range part.Parts {
		subPlain, subHTML := walkParts(p)
		if plain == "" {
			plain = subPlain
		}
		if htmlBody == "" {
			htmlBody = subHTML
		}
	}
	return plain, htmlBody
}

var tagRE = regexp.MustCompile(`<[^>]+>`)
var spaceRE = regexp.MustCompile(`\s+`)

func stripHTML(in string) string {
	out := tagRE.ReplaceAllString(in, " ")
	out = html.UnescapeString(out)
	return strings.TrimSpace(spaceRE.ReplaceAllString(out, " "))
}

// HeaderValue returns the value of a header (case-insensitive lookup).
func HeaderValue(headers []*gmailapi.MessagePartHeader, name string) string {
	target := strings.ToLower(name)
	for _, h := range headers {
		if strings.ToLower(h.Name) == target {
			return h.Value
		}
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/projects/go_gws_mcp
go test ./internal/gmail/... -v
```

Expected: 4 tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/gmail/helpers.go internal/gmail/helpers_test.go
git commit -m "feat(gmail): add body extraction and header helpers"
```

---

## Task 16: search_gmail_messages tool

**Files:**
- Create: `internal/gmail/search.go`

This task adds a tool implementation. The body is straightforward Gmail API; integration testing is deferred.

- [ ] **Step 1: Implement `internal/gmail/search.go`**

```go
package gmail

import (
	"context"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// SearchArgs is the arg shape for search_gmail_messages.
type SearchArgs struct {
	Query           string `json:"query"`
	UserGoogleEmail string `json:"user_google_email"`
	PageSize        int    `json:"page_size,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
}

// SearchGmailMessages lists message IDs matching a Gmail search query.
func SearchGmailMessages(ctx context.Context, svc *gmailapi.Service, userEmail string, a SearchArgs) (string, error) {
	pageSize := a.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	call := svc.Users.Messages.List("me").Q(a.Query).MaxResults(int64(pageSize))
	if a.PageToken != "" {
		call = call.PageToken(a.PageToken)
	}
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Search results for query: %q (user: %s)\n", a.Query, userEmail)
	fmt.Fprintf(&b, "Found %d message(s)\n\n", len(resp.Messages))
	for _, m := range resp.Messages {
		fmt.Fprintf(&b, "- Message ID: %s | Thread ID: %s\n", m.Id, m.ThreadId)
		fmt.Fprintf(&b, "  URL: https://mail.google.com/mail/u/0/#inbox/%s\n", m.Id)
	}
	if resp.NextPageToken != "" {
		fmt.Fprintf(&b, "\nNext page token: %s\n", resp.NextPageToken)
	}
	return b.String(), nil
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/gmail/search.go
git commit -m "feat(gmail): add search_gmail_messages tool body"
```

---

## Task 17: get_gmail_message_content tool

**Files:**
- Create: `internal/gmail/get.go`

- [ ] **Step 1: Implement `internal/gmail/get.go`**

```go
package gmail

import (
	"context"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// GetMessageArgs is the arg shape for get_gmail_message_content.
type GetMessageArgs struct {
	MessageID       string `json:"message_id"`
	UserGoogleEmail string `json:"user_google_email"`
}

// GetGmailMessageContent fetches a single message and returns subject/from/body.
func GetGmailMessageContent(ctx context.Context, svc *gmailapi.Service, userEmail string, a GetMessageArgs) (string, error) {
	if a.MessageID == "" {
		return "", fmt.Errorf("message_id is required")
	}
	msg, err := svc.Users.Messages.Get("me", a.MessageID).Format("full").Context(ctx).Do()
	if err != nil {
		return "", err
	}

	subject := HeaderValue(msg.Payload.Headers, "Subject")
	from := HeaderValue(msg.Payload.Headers, "From")
	to := HeaderValue(msg.Payload.Headers, "To")
	date := HeaderValue(msg.Payload.Headers, "Date")
	body := ExtractTextBody(msg.Payload)

	var b strings.Builder
	fmt.Fprintf(&b, "Message ID: %s\nThread ID: %s\nUser: %s\n\n", msg.Id, msg.ThreadId, userEmail)
	fmt.Fprintf(&b, "Subject: %s\nFrom: %s\nTo: %s\nDate: %s\n\n", subject, from, to, date)
	fmt.Fprintf(&b, "Body:\n%s\n", body)
	return b.String(), nil
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/gmail/get.go
git commit -m "feat(gmail): add get_gmail_message_content tool body"
```

---

## Task 18: get_gmail_messages_content_batch tool

**Files:**
- Create: `internal/gmail/batch.go`

The Go Gmail SDK doesn't expose batch HTTP requests as cleanly as Python's `BatchHttpRequest`. We use a worker pool with bounded concurrency.

- [ ] **Step 1: Implement `internal/gmail/batch.go`**

```go
package gmail

import (
	"context"
	"fmt"
	"strings"
	"sync"

	gmailapi "google.golang.org/api/gmail/v1"
)

const batchConcurrency = 5

// BatchGetArgs is the arg shape for get_gmail_messages_content_batch.
type BatchGetArgs struct {
	MessageIDs      []string `json:"message_ids"`
	UserGoogleEmail string   `json:"user_google_email"`
}

type batchResult struct {
	idx int
	out string
	err error
}

// GetGmailMessagesContentBatch fetches up to N messages in parallel.
func GetGmailMessagesContentBatch(ctx context.Context, svc *gmailapi.Service, userEmail string, a BatchGetArgs) (string, error) {
	if len(a.MessageIDs) == 0 {
		return "", fmt.Errorf("message_ids is required")
	}

	results := make([]batchResult, len(a.MessageIDs))
	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup

	for i, id := range a.MessageIDs {
		i, id := i, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			msg, err := svc.Users.Messages.Get("me", id).Format("metadata").Context(ctx).Do()
			if err != nil {
				results[i] = batchResult{idx: i, err: err}
				return
			}
			subject := HeaderValue(msg.Payload.Headers, "Subject")
			from := HeaderValue(msg.Payload.Headers, "From")
			results[i] = batchResult{
				idx: i,
				out: fmt.Sprintf("- %s | From: %s | Subject: %s", msg.Id, from, subject),
			}
		}()
	}
	wg.Wait()

	var b strings.Builder
	fmt.Fprintf(&b, "Batch results (%d messages, user: %s):\n\n", len(a.MessageIDs), userEmail)
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(&b, "- ERROR fetching %s: %v\n", a.MessageIDs[r.idx], r.err)
			continue
		}
		fmt.Fprintln(&b, r.out)
	}
	return b.String(), nil
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/gmail/batch.go
git commit -m "feat(gmail): add get_gmail_messages_content_batch with bounded concurrency"
```

---

## Task 19: send_gmail_message tool

**Files:**
- Create: `internal/gmail/send.go`

- [ ] **Step 1: Implement `internal/gmail/send.go`**

```go
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// SendArgs is the arg shape for send_gmail_message.
type SendArgs struct {
	To              string `json:"to"`
	Subject         string `json:"subject"`
	Body            string `json:"body"`
	UserGoogleEmail string `json:"user_google_email"`
	Cc              string `json:"cc,omitempty"`
	Bcc             string `json:"bcc,omitempty"`
}

// SendGmailMessage composes a plain-text RFC 822 message and sends it.
func SendGmailMessage(ctx context.Context, svc *gmailapi.Service, userEmail string, a SendArgs) (string, error) {
	if a.To == "" || a.Subject == "" {
		return "", fmt.Errorf("to and subject are required")
	}
	raw := buildRFC822Message(userEmail, a)
	encoded := base64.URLEncoding.EncodeToString([]byte(raw))

	msg, err := svc.Users.Messages.Send("me", &gmailapi.Message{Raw: encoded}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Sent message ID: %s\nThread ID: %s", msg.Id, msg.ThreadId), nil
}

func buildRFC822Message(from string, a SendArgs) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", a.To)
	if a.Cc != "" {
		fmt.Fprintf(&b, "Cc: %s\r\n", a.Cc)
	}
	if a.Bcc != "" {
		fmt.Fprintf(&b, "Bcc: %s\r\n", a.Bcc)
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", a.Subject)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	fmt.Fprintf(&b, "\r\n%s", a.Body)
	return b.String()
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/gmail/send.go
git commit -m "feat(gmail): add send_gmail_message tool body"
```

---

## Task 20: list_gmail_labels tool

**Files:**
- Create: `internal/gmail/labels.go`

- [ ] **Step 1: Implement `internal/gmail/labels.go`**

```go
package gmail

import (
	"context"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// LabelsArgs is the arg shape for list_gmail_labels (only user email).
type LabelsArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
}

// ListGmailLabels returns the user's Gmail labels.
func ListGmailLabels(ctx context.Context, svc *gmailapi.Service, userEmail string, a LabelsArgs) (string, error) {
	resp, err := svc.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Labels for %s (%d total):\n\n", userEmail, len(resp.Labels))
	for _, l := range resp.Labels {
		fmt.Fprintf(&b, "- %s [%s] (id: %s)\n", l.Name, l.Type, l.Id)
	}
	return b.String(), nil
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/gmail/labels.go
git commit -m "feat(gmail): add list_gmail_labels tool body"
```

---

## Task 21: Gmail tool registration

**Files:**
- Create: `internal/gmail/register.go`

- [ ] **Step 1: Implement `internal/gmail/register.go`**

```go
package gmail

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// RegisterTools wires all Gmail tools onto srv and records them in registry.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	registerSearch(srv, registry, oauthClient, defaultEmail)
	registerGet(srv, registry, oauthClient, defaultEmail)
	registerBatch(srv, registry, oauthClient, defaultEmail)
	registerSend(srv, registry, oauthClient, defaultEmail)
	registerLabels(srv, registry, oauthClient, defaultEmail)
}

func registerSearch(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("search_gmail_messages",
		mcp.WithDescription("Searches messages in a user's Gmail account based on a query. Supports standard Gmail search operators."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Gmail search query (e.g., 'is:unread')")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithNumber("page_size", mcp.Description("Max results, default 10")),
		mcp.WithString("page_token", mcp.Description("Pagination token for next page")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("search_gmail_messages", scopes)
	srv.AddTool(tool, auth.RequireGmailService("search_gmail_messages", scopes, c, email, SearchGmailMessages))
}

func registerGet(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_gmail_message_content",
		mcp.WithDescription("Retrieves the full content (subject, from, to, date, body) of a specific Gmail message."),
		mcp.WithString("message_id", mcp.Required(), mcp.Description("The unique Gmail message ID")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("get_gmail_message_content", scopes)
	srv.AddTool(tool, auth.RequireGmailService("get_gmail_message_content", scopes, c, email, GetGmailMessageContent))
}

func registerBatch(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_gmail_messages_content_batch",
		mcp.WithDescription("Fetches metadata (id, from, subject) for multiple message IDs in parallel."),
		mcp.WithArray("message_ids", mcp.Required(), mcp.Description("List of Gmail message IDs to fetch")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("get_gmail_messages_content_batch", scopes)
	srv.AddTool(tool, auth.RequireGmailService("get_gmail_messages_content_batch", scopes, c, email, GetGmailMessagesContentBatch))
}

func registerSend(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("send_gmail_message",
		mcp.WithDescription("Sends a plain-text email from the authenticated Gmail account."),
		mcp.WithString("to", mcp.Required(), mcp.Description("Recipient email address(es), comma-separated")),
		mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Plain-text message body")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("Sender's Google email address")),
		mcp.WithString("cc", mcp.Description("CC recipients")),
		mcp.WithString("bcc", mcp.Description("BCC recipients")),
	)
	scopes := []string{auth.GmailSendScope}
	reg.Record("send_gmail_message", scopes)
	srv.AddTool(tool, auth.RequireGmailService("send_gmail_message", scopes, c, email, SendGmailMessage))
}

func registerLabels(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("list_gmail_labels",
		mcp.WithDescription("Lists all labels (system and user-defined) in the user's Gmail account."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("list_gmail_labels", scopes)
	srv.AddTool(tool, auth.RequireGmailService("list_gmail_labels", scopes, c, email, ListGmailLabels))
}
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add internal/gmail/register.go
git commit -m "feat(gmail): register all Gmail tools onto MCP server"
```

---

## Task 22: Main entry point

**Files:**
- Create: `cmd/workspace-mcp/main.go`

- [ ] **Step 1: Implement `cmd/workspace-mcp/main.go`**

```go
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
```

- [ ] **Step 2: Verify build succeeds**

```bash
cd ~/projects/go_gws_mcp
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Build the binary**

```bash
cd ~/projects/go_gws_mcp
go build -o workspace-mcp ./cmd/workspace-mcp
ls -la workspace-mcp
```

Expected: a binary file, ~10-30MB.

- [ ] **Step 4: Commit**

```bash
cd ~/projects/go_gws_mcp
git add cmd/workspace-mcp/main.go
git commit -m "feat(cmd): add main entry point with flag parsing and transport dispatch"
```

---

## Task 23: README and .env.example

**Files:**
- Modify: `README.md` (replace placeholder)
- Create: `.env.example`

- [ ] **Step 1: Write `.env.example`**

```bash
# Required
GOOGLE_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret

# Optional (single-user convenience)
USER_GOOGLE_EMAIL=you@example.com

# Optional (transport)
WORKSPACE_MCP_TRANSPORT=stdio        # or streamable-http
WORKSPACE_MCP_PORT=8000
WORKSPACE_MCP_HOST=0.0.0.0
WORKSPACE_MCP_BASE_URI=http://localhost

# Optional (credential store)
WORKSPACE_MCP_CREDENTIALS_DIR=~/.google_workspace_mcp/credentials

# Optional (dev only — allows http:// callback for local OAuth)
OAUTHLIB_INSECURE_TRANSPORT=1
```

- [ ] **Step 2: Replace `README.md`**

```markdown
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
```

- [ ] **Step 3: Commit**

```bash
cd ~/projects/go_gws_mcp
git add README.md .env.example
git commit -m "docs: add README and .env.example"
```

---

## Task 24: Smoke test

**Files:** None modified — this is a manual verification.

- [ ] **Step 1: Run all unit tests**

```bash
cd ~/projects/go_gws_mcp
go test ./... -v
```

Expected: every test passes; no failures.

- [ ] **Step 2: Run `go vet`**

```bash
cd ~/projects/go_gws_mcp
go vet ./...
```

Expected: no diagnostics.

- [ ] **Step 3: Build the binary fresh**

```bash
cd ~/projects/go_gws_mcp
rm -f workspace-mcp
go build -o workspace-mcp ./cmd/workspace-mcp
./workspace-mcp --help
```

Expected: usage output listing all flags (`--transport`, `--single-user`, `--tools`, `--tool-tier`, `--permissions`, `--read-only`).

- [ ] **Step 4: Run with missing required env, expect descriptive error**

```bash
cd ~/projects/go_gws_mcp
unset GOOGLE_OAUTH_CLIENT_ID
./workspace-mcp 2>&1 || true
```

Expected: stderr "error: GOOGLE_OAUTH_CLIENT_ID is required (set env var)" and exit code 1.

- [ ] **Step 5: Run with invalid transport, expect descriptive error**

```bash
cd ~/projects/go_gws_mcp
GOOGLE_OAUTH_CLIENT_ID=test ./workspace-mcp --transport=foo 2>&1 || true
```

Expected: error mentioning unknown transport.

- [ ] **Step 6: (Optional, requires real Google OAuth client) End-to-end test**

```bash
cd ~/projects/go_gws_mcp
export GOOGLE_OAUTH_CLIENT_ID=<your real client id>
export GOOGLE_OAUTH_CLIENT_SECRET=<your real client secret>
export USER_GOOGLE_EMAIL=<your email>
export OAUTHLIB_INSECURE_TRANSPORT=1
./workspace-mcp
```

Then in another terminal, send an MCP `tools/list` request via stdio (or use an MCP client like Claude Desktop with this binary configured). Expected: server lists 5 Gmail tools. On first `search_gmail_messages` call, the server returns the OAuth ACTION REQUIRED message; visit the URL, authorize, retry the call — should now return search results.

- [ ] **Step 7: Final commit (if any cleanups needed)**

If the smoke test surfaced any issues that needed fixing, commit them now. If nothing to commit, skip this step. If it all worked first try:

```bash
cd ~/projects/go_gws_mcp
git log --oneline | head -25
```

Expected: clean commit history showing all 23 prior commits in order.

---

## Done

You should now have:

- A working Go MCP server with 5 Gmail tools
- Both stdio and streamable-HTTP transports
- OAuth 2.0 with PKCE and persistent local credentials
- Tool tier and granular permissions infrastructure (the latter is no-op as designed)
- A test suite that passes without any Google credentials
- ~24 git commits, each atomic and conventional

To extend with another service (e.g., Drive):

1. Re-run the design questions for that service's tool inventory
2. Add a new `internal/drive/` package mirroring `internal/gmail/`
3. Update `tiers.yaml`, `scopes.go`, `permissions.go`
4. Call `drive.RegisterTools(...)` from `main.go`

Each addition is local to its own package; no cross-cutting changes to auth, transport, or core.

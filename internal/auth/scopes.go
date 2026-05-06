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

// Drive scopes.
const (
	DriveReadonlyScope = "https://www.googleapis.com/auth/drive.readonly"
	DriveFileScope     = "https://www.googleapis.com/auth/drive.file"
	DriveScope         = "https://www.googleapis.com/auth/drive"
)

// Docs scopes.
const (
	DocsReadonlyScope = "https://www.googleapis.com/auth/documents.readonly"
	DocsScope         = "https://www.googleapis.com/auth/documents"
)

// BaseScopes are required for user identification on every OAuth flow.
var BaseScopes = []string{UserinfoEmailScope, UserinfoProfileScope, OpenIDScope}

// ScopeHierarchy maps broader scopes to the narrower scopes they cover.
// See https://developers.google.com/gmail/api/auth/scopes.
var ScopeHierarchy = map[string][]string{
	GmailModifyScope: {GmailReadonlyScope, GmailSendScope, GmailComposeScope, GmailLabelsScope},
	DriveScope:       {DriveReadonlyScope, DriveFileScope},
	DocsScope:        {DocsReadonlyScope},
}

// ToolScopesMap is the full scope set per service.
var ToolScopesMap = map[string][]string{
	"gmail": {
		GmailReadonlyScope, GmailSendScope, GmailComposeScope,
		GmailModifyScope, GmailLabelsScope, GmailSettingsScope,
	},
	"drive": {DriveReadonlyScope, DriveFileScope, DriveScope},
	"docs":  {DocsReadonlyScope, DocsScope, DriveReadonlyScope},
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

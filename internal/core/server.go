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

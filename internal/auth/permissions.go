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
	"drive": {
		{"readonly", []string{DriveReadonlyScope}},
		{"file", []string{DriveFileScope}},
		{"full", []string{DriveScope}},
	},
	"calendar": {{"readonly", nil}, {"full", nil}},
	"docs": {
		{"readonly", []string{DocsReadonlyScope, DriveReadonlyScope}},
		{"full", []string{DocsScope}},
	},
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

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

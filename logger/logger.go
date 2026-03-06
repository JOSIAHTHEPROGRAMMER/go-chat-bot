package logger

import "context"

// contextKey is unexported to avoid collisions with other packages using context.
type contextKey struct{}

// RequestLog holds observability data collected during a single request.
// Each layer writes to it as the request flows through the stack.
type RequestLog struct {
	Provider string
	PlanType string
	DocCount int
}

// NewContext attaches a RequestLog pointer to the context.
// Using a pointer so each layer can mutate it without replacing the context.
func NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &RequestLog{})
}

// FromContext retrieves the RequestLog from the context.
// Returns nil if none was set, callers should guard against this.
func FromContext(ctx context.Context) *RequestLog {
	val, _ := ctx.Value(contextKey{}).(*RequestLog)
	return val
}

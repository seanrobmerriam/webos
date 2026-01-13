package router

import (
	"context"
)

// ContextKey is the type used for context keys to avoid collisions.
type ContextKey string

const (
	// ParamsKey is the context key for URL parameters.
	ParamsKey ContextKey = "router.params"
)

// Params holds URL parameter values extracted from the route pattern.
type Params map[string]string

// Get returns the value of the parameter with the given key.
// Returns an empty string if the parameter doesn't exist.
func (p Params) Get(key string) string {
	return p[key]
}

// Has returns true if the parameter with the given key exists.
func (p Params) Has(key string) bool {
	_, ok := p[key]
	return ok
}

// WithParams returns a new context with the given parameters.
func WithParams(ctx context.Context, params Params) context.Context {
	return context.WithValue(ctx, ParamsKey, params)
}

// ParamsFromContext extracts URL parameters from the context.
func ParamsFromContext(ctx context.Context) (Params, bool) {
	params, ok := ctx.Value(ParamsKey).(Params)
	return params, ok
}

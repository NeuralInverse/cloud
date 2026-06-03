package cors

import (
	"context"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type contextKeyBehavior struct{}

// WithBehavior sets the CORS behavior for the given context.
func WithBehavior(ctx context.Context, behavior nicloudsdk.CORSBehavior) context.Context {
	return context.WithValue(ctx, contextKeyBehavior{}, behavior)
}

// HasBehavior returns true if the given context has the specified CORS behavior.
func HasBehavior(ctx context.Context, behavior nicloudsdk.CORSBehavior) bool {
	val := ctx.Value(contextKeyBehavior{})
	b, ok := val.(nicloudsdk.CORSBehavior)
	return ok && b == behavior
}

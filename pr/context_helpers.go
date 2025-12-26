package pr

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey struct{ name string }

var prProviderKey = &contextKey{"pr-provider"}

// ContextWithProvider adds a PR Provider to a context.Context.
// Use ProviderFromContext to retrieve it.
//
// Example:
//
//	provider, _ := pr.ProviderFromEnv(remoteURL)
//	ctx := pr.ContextWithProvider(context.Background(), provider)
//	// Pass ctx to functions that need PR access
func ContextWithProvider(ctx context.Context, p Provider) context.Context {
	return context.WithValue(ctx, prProviderKey, p)
}

// ProviderFromContext retrieves a PR Provider from a context.Context.
// Returns nil if no Provider is present.
//
// Example:
//
//	func createPR(ctx context.Context, title string) (*PullRequest, error) {
//	    provider := pr.ProviderFromContext(ctx)
//	    if provider == nil {
//	        return nil, ErrNoProvider
//	    }
//	    return provider.CreatePR(ctx, Options{Title: title})
//	}
func ProviderFromContext(ctx context.Context) Provider {
	if p, ok := ctx.Value(prProviderKey).(Provider); ok {
		return p
	}
	return nil
}

// MustProviderFromContext retrieves a PR Provider or panics.
// Use in code where provider is required and missing is a programming error.
func MustProviderFromContext(ctx context.Context) Provider {
	p := ProviderFromContext(ctx)
	if p == nil {
		panic("pr.Provider not found in context")
	}
	return p
}

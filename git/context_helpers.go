package git

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey struct{ name string }

var gitContextKey = &contextKey{"git-context"}

// ContextWithGit adds a git Context to a context.Context.
// Use GitFromContext to retrieve it.
//
// Example:
//
//	gitCtx, _ := git.NewContext(".")
//	ctx := git.ContextWithGit(context.Background(), gitCtx)
//	// Pass ctx to functions that need git access
func ContextWithGit(ctx context.Context, gc *Context) context.Context {
	return context.WithValue(ctx, gitContextKey, gc)
}

// GitFromContext retrieves a git Context from a context.Context.
// Returns nil if no git Context is present.
//
// Example:
//
//	func doWork(ctx context.Context) error {
//	    gitCtx := git.GitFromContext(ctx)
//	    if gitCtx == nil {
//	        return errors.New("git context required")
//	    }
//	    return gitCtx.CommitAll("message")
//	}
func GitFromContext(ctx context.Context) *Context {
	if gc, ok := ctx.Value(gitContextKey).(*Context); ok {
		return gc
	}
	return nil
}

// MustGitFromContext retrieves a git Context or panics.
// Use in code where git context is required and missing is a programming error.
func MustGitFromContext(ctx context.Context) *Context {
	gc := GitFromContext(ctx)
	if gc == nil {
		panic("git.Context not found in context")
	}
	return gc
}

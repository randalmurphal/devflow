package testutil

import (
	"context"
	"testing"
	"time"
)

// TestContext returns a context that is canceled when the test ends.
// This ensures any goroutines started during the test are properly cleaned up.
func TestContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return ctx
}

// TestContextWithTimeout returns a context with a timeout.
// The context is also canceled when the test ends.
func TestContextWithTimeout(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx
}

// TestContextWithDeadline returns a context with a deadline.
// The context is also canceled when the test ends.
func TestContextWithDeadline(t *testing.T, deadline time.Time) context.Context {
	t.Helper()

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	t.Cleanup(cancel)

	return ctx
}

// CancelableContext returns a context and cancel function.
// The context is automatically canceled when the test ends if not canceled earlier.
func CancelableContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return ctx, cancel
}

// BackgroundContext returns context.Background() for use in tests
// where cleanup isn't needed.
func BackgroundContext() context.Context {
	return context.Background()
}

// TODO returns context.TODO() - useful for tests that need an explicit TODO context.
func TODO() context.Context {
	return context.TODO()
}

// contextKey is used for storing test values in context.
type contextKey string

// WithValue returns a context with the given key-value pair.
// This is a convenience wrapper around context.WithValue.
func WithValue(ctx context.Context, key, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

// WithTestName adds the test name to the context.
func WithTestName(ctx context.Context, t *testing.T) context.Context {
	return context.WithValue(ctx, contextKey("test_name"), t.Name())
}

// TestNameFromContext retrieves the test name from context.
func TestNameFromContext(ctx context.Context) string {
	if v := ctx.Value(contextKey("test_name")); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

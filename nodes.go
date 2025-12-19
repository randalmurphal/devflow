package devflow

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// =============================================================================
// Node Types
// =============================================================================

// NodeFunc is a function that processes state and returns updated state.
// This signature is compatible with flowgraph's NodeFunc[DevState].
type NodeFunc func(ctx context.Context, state DevState) (DevState, error)

// NodeConfig configures node behavior
type NodeConfig struct {
	MaxReviewAttempts int    // Max review/fix cycles (default: 3)
	TestCommand       string // Test command (default: "go test ./...")
	LintCommand       string // Lint command (default: "go vet ./...")
	BaseBranch        string // Default base branch (default: "main")
}

// DefaultNodeConfig returns sensible defaults
func DefaultNodeConfig() NodeConfig {
	return NodeConfig{
		MaxReviewAttempts: 3,
		TestCommand:       "go test -race ./...",
		LintCommand:       "go vet ./...",
		BaseBranch:        "main",
	}
}

// =============================================================================
// Node Wrappers
// =============================================================================

// WithRetry wraps a node with retry logic
func WithRetry(node NodeFunc, maxRetries int) NodeFunc {
	return func(ctx context.Context, state DevState) (DevState, error) {
		var lastErr error
		for i := 0; i < maxRetries; i++ {
			result, err := node(ctx, state)
			if err == nil {
				return result, nil
			}
			lastErr = err
			// Exponential backoff could go here
		}
		return state, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
	}
}

// WithTranscript wraps a node with transcript recording
func WithTranscript(node NodeFunc, nodeName string) NodeFunc {
	return func(ctx context.Context, state DevState) (DevState, error) {
		mgr := TranscriptManagerFromContext(ctx)

		startTime := time.Now()
		result, err := node(ctx, state)
		duration := time.Since(startTime)

		if mgr != nil {
			turn := Turn{
				Role:      "system",
				Content:   fmt.Sprintf("Node %s completed in %v", nodeName, duration),
				Timestamp: time.Now(),
			}
			if err != nil {
				turn.Content = fmt.Sprintf("Node %s failed: %v", nodeName, err)
			}
			mgr.RecordTurn(state.RunID, turn)
		}

		return result, err
	}
}

// WithTiming wraps a node with timing metrics
func WithTiming(node NodeFunc) NodeFunc {
	return func(ctx context.Context, state DevState) (DevState, error) {
		start := time.Now()
		result, err := node(ctx, state)
		duration := time.Since(start)
		slog.Debug("node execution completed", "runId", state.RunID, "duration", duration)
		return result, err
	}
}

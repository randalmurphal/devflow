package context

import "errors"

// Context building errors
var (
	// ErrContextTooLarge indicates the context exceeds size limits.
	ErrContextTooLarge = errors.New("context too large")
)

package ssh

import "errors"

// SSH key errors.
var (
	// ErrNoSSHAgent is returned when the SSH agent is not available.
	ErrNoSSHAgent = errors.New("ssh-agent not available")

	// ErrNoSSHKeys is returned when no SSH keys are found.
	ErrNoSSHKeys = errors.New("no SSH keys found")

	// ErrKeyNotFound is returned when a specific key is not found in the agent.
	ErrKeyNotFound = errors.New("SSH key not found in agent")

	// ErrInvalidKeyFormat is returned when a public key file has invalid format.
	ErrInvalidKeyFormat = errors.New("invalid SSH public key format")
)

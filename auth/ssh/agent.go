package ssh

import (
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/crypto/ssh/agent"
)

// AgentConnection wraps an SSH agent with its underlying connection
// for proper resource cleanup.
type AgentConnection struct {
	agent.ExtendedAgent
	conn io.Closer
}

// Close closes the underlying connection to the SSH agent.
func (a *AgentConnection) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// GetAgent connects to the SSH agent via SSH_AUTH_SOCK.
// The returned AgentConnection should be closed when done to avoid resource leaks.
func GetAgent() (*AgentConnection, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, ErrNoSSHAgent
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("connect to ssh-agent: %w", err)
	}

	return &AgentConnection{
		ExtendedAgent: agent.NewClient(conn),
		conn:          conn,
	}, nil
}

// ListAgentKeys lists all keys currently in the SSH agent.
func ListAgentKeys(ag agent.Agent) ([]*agent.Key, error) {
	keys, err := ag.List()
	if err != nil {
		return nil, fmt.Errorf("list agent keys: %w", err)
	}
	return keys, nil
}

// FindAgentKeyByFingerprint finds a key in the agent by its fingerprint.
func FindAgentKeyByFingerprint(ag agent.Agent, fingerprint string) (*agent.Key, error) {
	keys, err := ag.List()
	if err != nil {
		return nil, fmt.Errorf("list agent keys: %w", err)
	}

	for _, key := range keys {
		keyFP := ComputeFingerprint(key.Blob)
		if keyFP == fingerprint {
			return key, nil
		}
	}

	return nil, ErrKeyNotFound
}

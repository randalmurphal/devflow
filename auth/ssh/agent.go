package ssh

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh/agent"
)

// GetAgent connects to the SSH agent via SSH_AUTH_SOCK.
func GetAgent() (agent.ExtendedAgent, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, ErrNoSSHAgent
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("connect to ssh-agent: %w", err)
	}

	return agent.NewClient(conn), nil
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

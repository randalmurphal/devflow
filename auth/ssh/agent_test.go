package ssh

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func TestGetAgent_NoSocket(t *testing.T) {
	// Unset SSH_AUTH_SOCK
	original := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if original != "" {
			os.Setenv("SSH_AUTH_SOCK", original)
		}
	}()

	_, err := GetAgent()
	if !errors.Is(err, ErrNoSSHAgent) {
		t.Errorf("GetAgent() error = %v, want ErrNoSSHAgent", err)
	}
}

func TestGetAgent_InvalidSocket(t *testing.T) {
	// Set to non-existent socket
	original := os.Getenv("SSH_AUTH_SOCK")
	os.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket/path")
	defer func() {
		if original != "" {
			os.Setenv("SSH_AUTH_SOCK", original)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	_, err := GetAgent()
	if err == nil {
		t.Error("GetAgent() expected error for invalid socket")
	}
	if errors.Is(err, ErrNoSSHAgent) {
		t.Error("GetAgent() should not return ErrNoSSHAgent for dial failure")
	}
}

func TestAgentConnection_Close(t *testing.T) {
	t.Run("close with nil conn", func(t *testing.T) {
		ac := &AgentConnection{conn: nil}
		err := ac.Close()
		if err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
	})

	t.Run("close with mock conn", func(t *testing.T) {
		mc := &mockCloser{}
		ac := &AgentConnection{conn: mc}
		err := ac.Close()
		if err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
		if !mc.closed {
			t.Error("Close() did not close underlying connection")
		}
	})
}

type mockCloser struct {
	closed bool
}

func (m *mockCloser) Close() error {
	m.closed = true
	return nil
}

// mockAgent implements agent.Agent for testing
type mockAgent struct {
	keys     []*agent.Key
	listErr  error
	signErr  error
	signFunc func(key ssh.PublicKey, data []byte) (*ssh.Signature, error)
}

func (m *mockAgent) List() ([]*agent.Key, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.keys, nil
}

func (m *mockAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	if m.signErr != nil {
		return nil, m.signErr
	}
	if m.signFunc != nil {
		return m.signFunc(key, data)
	}
	return &ssh.Signature{
		Format: "ssh-ed25519",
		Blob:   []byte("mock-signature"),
	}, nil
}

func (m *mockAgent) Add(_ agent.AddedKey) error { return nil }

func (m *mockAgent) Remove(_ ssh.PublicKey) error { return nil }

func (m *mockAgent) RemoveAll() error { return nil }

func (m *mockAgent) Lock(_ []byte) error { return nil }

func (m *mockAgent) Unlock(_ []byte) error { return nil }

func (m *mockAgent) Signers() ([]ssh.Signer, error) { return nil, nil }

// mockExtendedAgent wraps mockAgent to implement ExtendedAgent
type mockExtendedAgent struct {
	*mockAgent
}

func (m *mockExtendedAgent) SignWithFlags(key ssh.PublicKey, data []byte, _ agent.SignatureFlags) (*ssh.Signature, error) {
	return m.Sign(key, data)
}

func (m *mockExtendedAgent) Extension(_ string, _ []byte) ([]byte, error) {
	return nil, agent.ErrExtensionUnsupported
}

func TestListAgentKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockAgent{
			keys: []*agent.Key{
				{Format: "ssh-ed25519", Blob: []byte("key1")},
				{Format: "ssh-rsa", Blob: []byte("key2")},
			},
		}

		keys, err := ListAgentKeys(mock)
		if err != nil {
			t.Fatalf("ListAgentKeys() error = %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("ListAgentKeys() returned %d keys, want 2", len(keys))
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockAgent{
			listErr: errors.New("agent error"),
		}

		_, err := ListAgentKeys(mock)
		if err == nil {
			t.Error("ListAgentKeys() expected error")
		}
	})
}

func TestFindAgentKeyByFingerprint(t *testing.T) {
	// Create a key with known fingerprint
	keyBlob := []byte("test-key-blob-for-fingerprint")
	expectedFP := ComputeFingerprint(keyBlob)

	t.Run("key found", func(t *testing.T) {
		mock := &mockAgent{
			keys: []*agent.Key{
				{Format: "ssh-rsa", Blob: []byte("other-key")},
				{Format: "ssh-ed25519", Blob: keyBlob},
			},
		}

		key, err := FindAgentKeyByFingerprint(mock, expectedFP)
		if err != nil {
			t.Fatalf("FindAgentKeyByFingerprint() error = %v", err)
		}
		if key.Format != "ssh-ed25519" {
			t.Errorf("Format = %q, want ssh-ed25519", key.Format)
		}
	})

	t.Run("key not found", func(t *testing.T) {
		mock := &mockAgent{
			keys: []*agent.Key{
				{Format: "ssh-rsa", Blob: []byte("other-key")},
			},
		}

		_, err := FindAgentKeyByFingerprint(mock, expectedFP)
		if !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("error = %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("list error", func(t *testing.T) {
		mock := &mockAgent{
			listErr: errors.New("agent error"),
		}

		_, err := FindAgentKeyByFingerprint(mock, expectedFP)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("empty agent", func(t *testing.T) {
		mock := &mockAgent{
			keys: []*agent.Key{},
		}

		_, err := FindAgentKeyByFingerprint(mock, expectedFP)
		if !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("error = %v, want ErrKeyNotFound", err)
		}
	})
}

package ssh

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds configuration for SSH key operations.
type Config struct {
	// SSHDir is the SSH directory path.
	// Defaults to ~/.ssh if empty.
	SSHDir string

	// PreferredKeys is the preference order for key types.
	// Defaults to ed25519, ecdsa, rsa if empty.
	PreferredKeys []string
}

// DefaultPreferredKeys is the default key preference order.
var DefaultPreferredKeys = []string{
	"id_ed25519.pub",
	"id_ecdsa.pub",
	"id_rsa.pub",
}

func (c Config) sshDir() (string, error) {
	if c.SSHDir != "" {
		return c.SSHDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".ssh"), nil
}

func (c Config) preferredKeys() []string {
	if len(c.PreferredKeys) > 0 {
		return c.PreferredKeys
	}
	return DefaultPreferredKeys
}

// KeyInfo holds information about an SSH key.
type KeyInfo struct {
	// Path is the path to the public key file.
	Path string

	// PublicKey is the full public key in authorized_keys format.
	PublicKey string

	// KeyType is the key algorithm (e.g., "ssh-ed25519", "ssh-rsa").
	KeyType string

	// Fingerprint is the SHA256 fingerprint of the key.
	Fingerprint string

	// Comment is the optional key comment.
	Comment string
}

// FindDefaultKey finds the default SSH key using default configuration.
func FindDefaultKey() (*KeyInfo, error) {
	return FindDefaultKeyWithConfig(Config{})
}

// FindDefaultKeyWithConfig finds the default SSH key using custom configuration.
func FindDefaultKeyWithConfig(cfg Config) (*KeyInfo, error) {
	sshDir, err := cfg.sshDir()
	if err != nil {
		return nil, err
	}

	for _, name := range cfg.preferredKeys() {
		path := filepath.Join(sshDir, name)
		if info, err := ReadPublicKey(path); err == nil {
			return info, nil
		}
	}

	return nil, ErrNoSSHKeys
}

// ReadPublicKey reads and parses an SSH public key file.
func ReadPublicKey(path string) (*KeyInfo, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-provided path expected
	if err != nil {
		return nil, err
	}

	return ParsePublicKey(path, string(data))
}

// ParsePublicKey parses an SSH public key string.
func ParsePublicKey(path, keyData string) (*KeyInfo, error) {
	keyData = strings.TrimSpace(keyData)
	parts := strings.SplitN(keyData, " ", 3)
	if len(parts) < 2 {
		return nil, ErrInvalidKeyFormat
	}

	keyType := parts[0]
	keyBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid key data: %w", err)
	}

	fingerprint := ComputeFingerprint(keyBytes)

	comment := ""
	if len(parts) == 3 {
		comment = parts[2]
	}

	return &KeyInfo{
		Path:        path,
		PublicKey:   keyData,
		KeyType:     keyType,
		Fingerprint: fingerprint,
		Comment:     comment,
	}, nil
}

// ListLocalKeys lists all SSH public keys in the SSH directory.
func ListLocalKeys() ([]*KeyInfo, error) {
	return ListLocalKeysWithConfig(Config{})
}

// ListLocalKeysWithConfig lists all SSH public keys using custom configuration.
func ListLocalKeysWithConfig(cfg Config) ([]*KeyInfo, error) {
	sshDir, err := cfg.sshDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSSHKeys
		}
		return nil, fmt.Errorf("read ssh directory: %w", err)
	}

	var keys []*KeyInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}

		path := filepath.Join(sshDir, entry.Name())
		info, err := ReadPublicKey(path)
		if err != nil {
			continue // Skip invalid key files
		}
		keys = append(keys, info)
	}

	if len(keys) == 0 {
		return nil, ErrNoSSHKeys
	}

	return keys, nil
}

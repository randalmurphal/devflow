package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

// Sample SSH public keys for testing
const (
	sampleED25519Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
	sampleRSAKey     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... user@host"
)

func TestParsePublicKey(t *testing.T) {
	t.Run("valid ed25519 key", func(t *testing.T) {
		info, err := ParsePublicKey("/test/path", sampleED25519Key)
		if err != nil {
			t.Fatalf("ParsePublicKey() error = %v", err)
		}
		if info.KeyType != "ssh-ed25519" {
			t.Errorf("KeyType = %q, want %q", info.KeyType, "ssh-ed25519")
		}
		if info.Comment != "test@example.com" {
			t.Errorf("Comment = %q, want %q", info.Comment, "test@example.com")
		}
		if info.Fingerprint == "" {
			t.Error("Fingerprint is empty")
		}
		if info.Path != "/test/path" {
			t.Errorf("Path = %q, want %q", info.Path, "/test/path")
		}
	})

	t.Run("key without comment", func(t *testing.T) {
		keyData := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"
		info, err := ParsePublicKey("/test", keyData)
		if err != nil {
			t.Fatalf("ParsePublicKey() error = %v", err)
		}
		if info.Comment != "" {
			t.Errorf("Comment = %q, want empty", info.Comment)
		}
	})

	t.Run("invalid format - too few parts", func(t *testing.T) {
		_, err := ParsePublicKey("/test", "ssh-ed25519")
		if err != ErrInvalidKeyFormat {
			t.Errorf("error = %v, want ErrInvalidKeyFormat", err)
		}
	})

	t.Run("invalid format - bad base64", func(t *testing.T) {
		_, err := ParsePublicKey("/test", "ssh-ed25519 not-valid-base64")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		keyData := "  " + sampleED25519Key + "  \n"
		info, err := ParsePublicKey("/test", keyData)
		if err != nil {
			t.Fatalf("ParsePublicKey() error = %v", err)
		}
		if info.KeyType != "ssh-ed25519" {
			t.Errorf("KeyType = %q, want %q", info.KeyType, "ssh-ed25519")
		}
	})
}

func TestComputeFingerprint(t *testing.T) {
	blob := []byte("test-key-blob")
	fp := ComputeFingerprint(blob)

	if fp == "" {
		t.Error("fingerprint is empty")
	}
	if fp[:7] != "SHA256:" {
		t.Errorf("fingerprint should start with 'SHA256:', got %q", fp)
	}

	// Verify deterministic
	fp2 := ComputeFingerprint(blob)
	if fp != fp2 {
		t.Error("fingerprint should be deterministic")
	}

	// Different input should give different fingerprint
	fp3 := ComputeFingerprint([]byte("different-blob"))
	if fp == fp3 {
		t.Error("different inputs should give different fingerprints")
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}

	preferredKeys := cfg.preferredKeys()
	if len(preferredKeys) == 0 {
		t.Error("preferredKeys should have defaults")
	}
	if preferredKeys[0] != "id_ed25519.pub" {
		t.Errorf("first preferred key = %q, want %q", preferredKeys[0], "id_ed25519.pub")
	}
}

func TestConfig_CustomValues(t *testing.T) {
	cfg := Config{
		SSHDir:        "/custom/ssh",
		PreferredKeys: []string{"my_key.pub"},
	}

	sshDir, err := cfg.sshDir()
	if err != nil {
		t.Fatalf("sshDir() error = %v", err)
	}
	if sshDir != "/custom/ssh" {
		t.Errorf("sshDir = %q, want %q", sshDir, "/custom/ssh")
	}

	preferredKeys := cfg.preferredKeys()
	if len(preferredKeys) != 1 || preferredKeys[0] != "my_key.pub" {
		t.Errorf("preferredKeys = %v, want [my_key.pub]", preferredKeys)
	}
}

func TestReadPublicKey(t *testing.T) {
	// Create temp directory with test key
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.pub")

	err := os.WriteFile(keyPath, []byte(sampleED25519Key), 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	info, err := ReadPublicKey(keyPath)
	if err != nil {
		t.Fatalf("ReadPublicKey() error = %v", err)
	}
	if info.Path != keyPath {
		t.Errorf("Path = %q, want %q", info.Path, keyPath)
	}
	if info.KeyType != "ssh-ed25519" {
		t.Errorf("KeyType = %q, want %q", info.KeyType, "ssh-ed25519")
	}
}

func TestReadPublicKey_NotFound(t *testing.T) {
	_, err := ReadPublicKey("/nonexistent/path/key.pub")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFindDefaultKeyWithConfig(t *testing.T) {
	// Create temp directory with test keys
	tmpDir := t.TempDir()

	// Create an ed25519 key
	ed25519Path := filepath.Join(tmpDir, "id_ed25519.pub")
	err := os.WriteFile(ed25519Path, []byte(sampleED25519Key), 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := Config{SSHDir: tmpDir}
	info, err := FindDefaultKeyWithConfig(cfg)
	if err != nil {
		t.Fatalf("FindDefaultKeyWithConfig() error = %v", err)
	}
	if info.Path != ed25519Path {
		t.Errorf("Path = %q, want %q", info.Path, ed25519Path)
	}
}

func TestFindDefaultKeyWithConfig_NoKeys(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{SSHDir: tmpDir}

	_, err := FindDefaultKeyWithConfig(cfg)
	if err != ErrNoSSHKeys {
		t.Errorf("error = %v, want ErrNoSSHKeys", err)
	}
}

func TestListLocalKeysWithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple keys
	err := os.WriteFile(
		filepath.Join(tmpDir, "id_ed25519.pub"),
		[]byte(sampleED25519Key),
		0600,
	)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create another key with different content
	anotherKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB another@host"
	err = os.WriteFile(
		filepath.Join(tmpDir, "other_key.pub"),
		[]byte(anotherKey),
		0600,
	)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := Config{SSHDir: tmpDir}
	keys, err := ListLocalKeysWithConfig(cfg)
	if err != nil {
		t.Fatalf("ListLocalKeysWithConfig() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("got %d keys, want 2", len(keys))
	}
}

func TestListLocalKeysWithConfig_NoDirectory(t *testing.T) {
	cfg := Config{SSHDir: "/nonexistent/directory"}

	_, err := ListLocalKeysWithConfig(cfg)
	if err != ErrNoSSHKeys {
		t.Errorf("error = %v, want ErrNoSSHKeys", err)
	}
}

func TestListLocalKeysWithConfig_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{SSHDir: tmpDir}

	_, err := ListLocalKeysWithConfig(cfg)
	if err != ErrNoSSHKeys {
		t.Errorf("error = %v, want ErrNoSSHKeys", err)
	}
}

func TestListLocalKeysWithConfig_SkipsInvalidKeys(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid key
	err := os.WriteFile(
		filepath.Join(tmpDir, "valid.pub"),
		[]byte(sampleED25519Key),
		0600,
	)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create invalid key
	err = os.WriteFile(
		filepath.Join(tmpDir, "invalid.pub"),
		[]byte("not a valid key"),
		0600,
	)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := Config{SSHDir: tmpDir}
	keys, err := ListLocalKeysWithConfig(cfg)
	if err != nil {
		t.Fatalf("ListLocalKeysWithConfig() error = %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("got %d keys, want 1 (should skip invalid)", len(keys))
	}
}

func TestListLocalKeysWithConfig_SkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid key
	err := os.WriteFile(
		filepath.Join(tmpDir, "valid.pub"),
		[]byte(sampleED25519Key),
		0600,
	)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create a directory ending in .pub
	err = os.Mkdir(filepath.Join(tmpDir, "subdir.pub"), 0755)
	if err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	cfg := Config{SSHDir: tmpDir}
	keys, err := ListLocalKeysWithConfig(cfg)
	if err != nil {
		t.Fatalf("ListLocalKeysWithConfig() error = %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("got %d keys, want 1 (should skip directories)", len(keys))
	}
}

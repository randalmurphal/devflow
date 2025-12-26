package ssh

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Test ed25519 private key (unencrypted, for testing only)
// Generated with: ssh-keygen -t ed25519 -N "" -f test_key
const testED25519PrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBk1/7jOTEXnxzGBhMDhINv5cP674zQN3Y240uellpH3AAAAJjFc4JRxXOC
UQAAAAtzc2gtZWQyNTUxOQAAACBk1/7jOTEXnxzGBhMDhINv5cP674zQN3Y240uellpH3A
AAAECvIQlgj5pI4bTMsBA/6hTJMv65Bf3UnMH6GsNMKNIxP2TX/uM5MRefHMYGEwOEg2/l
w/rvjNA3djbjS56WWkfcAAAAEHRlc3RAZXhhbXBsZS5jb20BAgMEBQ==
-----END OPENSSH PRIVATE KEY-----`

// Corresponding public key
const testED25519PublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGTX/uM5MRefHMYGEwOEg2/lw/rvjNA3djbjS56WWkfc test@example.com"

func TestSignWithAgent(t *testing.T) {
	// Create a key with known fingerprint
	keyBlob := []byte("test-key-blob")
	expectedFP := ComputeFingerprint(keyBlob)

	t.Run("success", func(t *testing.T) {
		mock := &mockExtendedAgent{
			mockAgent: &mockAgent{
				keys: []*agent.Key{
					{Format: "ssh-ed25519", Blob: keyBlob},
				},
			},
		}

		sig, err := SignWithAgent(mock, expectedFP, []byte("test data"))
		if err != nil {
			t.Fatalf("SignWithAgent() error = %v", err)
		}
		if sig == "" {
			t.Error("SignWithAgent() returned empty signature")
		}

		// Verify it's valid base64
		_, err = base64.StdEncoding.DecodeString(sig)
		if err != nil {
			t.Errorf("signature is not valid base64: %v", err)
		}
	})

	t.Run("key not found", func(t *testing.T) {
		mock := &mockExtendedAgent{
			mockAgent: &mockAgent{
				keys: []*agent.Key{},
			},
		}

		_, err := SignWithAgent(mock, expectedFP, []byte("test data"))
		if !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("error = %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("sign error", func(t *testing.T) {
		mock := &mockExtendedAgent{
			mockAgent: &mockAgent{
				keys: []*agent.Key{
					{Format: "ssh-ed25519", Blob: keyBlob},
				},
				signErr: errors.New("signing failed"),
			},
		}

		_, err := SignWithAgent(mock, expectedFP, []byte("test data"))
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestSignChallengeWithAgent(t *testing.T) {
	keyBlob := []byte("test-key-blob")
	expectedFP := ComputeFingerprint(keyBlob)

	t.Run("success", func(t *testing.T) {
		mock := &mockExtendedAgent{
			mockAgent: &mockAgent{
				keys: []*agent.Key{
					{Format: "ssh-ed25519", Blob: keyBlob},
				},
			},
		}

		// Create a valid base64 challenge (RawStdEncoding = no padding)
		challenge := base64.RawStdEncoding.EncodeToString([]byte("test challenge"))

		sig, err := SignChallengeWithAgent(mock, expectedFP, challenge)
		if err != nil {
			t.Fatalf("SignChallengeWithAgent() error = %v", err)
		}
		if sig == "" {
			t.Error("signature is empty")
		}
	})

	t.Run("invalid base64 challenge", func(t *testing.T) {
		mock := &mockExtendedAgent{
			mockAgent: &mockAgent{
				keys: []*agent.Key{
					{Format: "ssh-ed25519", Blob: keyBlob},
				},
			},
		}

		_, err := SignChallengeWithAgent(mock, expectedFP, "not-valid-base64!!!")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}

func TestSignWithKeyFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "test_key")
		err := os.WriteFile(keyPath, []byte(testED25519PrivateKey), 0600)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		sig, err := SignWithKeyFile(keyPath, []byte("test data to sign"))
		if err != nil {
			t.Fatalf("SignWithKeyFile() error = %v", err)
		}
		if sig == "" {
			t.Error("signature is empty")
		}

		// Verify it's valid base64
		decoded, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			t.Errorf("signature is not valid base64: %v", err)
		}
		if len(decoded) == 0 {
			t.Error("decoded signature is empty")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := SignWithKeyFile("/nonexistent/key", []byte("test data"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("invalid key format", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "bad_key")
		err := os.WriteFile(keyPath, []byte("not a valid key"), 0600)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err = SignWithKeyFile(keyPath, []byte("test data"))
		if err == nil {
			t.Error("expected error for invalid key")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "test_key")
		err := os.WriteFile(keyPath, []byte(testED25519PrivateKey), 0600)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		// Empty data should still produce a valid signature
		sig, err := SignWithKeyFile(keyPath, []byte{})
		if err != nil {
			t.Fatalf("SignWithKeyFile() error = %v", err)
		}
		if sig == "" {
			t.Error("signature is empty")
		}
	})
}

func TestSignChallengeWithKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	err := os.WriteFile(keyPath, []byte(testED25519PrivateKey), 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Run("success", func(t *testing.T) {
		challenge := base64.RawStdEncoding.EncodeToString([]byte("test challenge"))

		sig, err := SignChallengeWithKeyFile(keyPath, challenge)
		if err != nil {
			t.Fatalf("SignChallengeWithKeyFile() error = %v", err)
		}
		if sig == "" {
			t.Error("signature is empty")
		}
	})

	t.Run("invalid base64 challenge", func(t *testing.T) {
		_, err := SignChallengeWithKeyFile(keyPath, "not-valid-base64!!!")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}

func TestSignatureVerification(t *testing.T) {
	// Test that signatures produced can be verified with the public key
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	err := os.WriteFile(keyPath, []byte(testED25519PrivateKey), 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data := []byte("data to sign and verify")
	sigB64, err := SignWithKeyFile(keyPath, data)
	if err != nil {
		t.Fatalf("SignWithKeyFile() error = %v", err)
	}

	// Parse public key for verification
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testED25519PublicKey))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey() error = %v", err)
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}

	// Unmarshal SSH signature
	sig := &ssh.Signature{}
	if err := ssh.Unmarshal(sigBytes, sig); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify
	if err := pubKey.Verify(data, sig); err != nil {
		t.Errorf("signature verification failed: %v", err)
	}
}

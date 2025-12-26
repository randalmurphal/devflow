package ssh

import (
	"encoding/base64"
	"fmt"
	"os"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SignWithAgent signs data using the SSH agent.
// The fingerprint is used to find the correct key in the agent.
// Returns the signature encoded as base64.
func SignWithAgent(ag agent.ExtendedAgent, fingerprint string, data []byte) (string, error) {
	// Find matching key
	key, err := FindAgentKeyByFingerprint(ag, fingerprint)
	if err != nil {
		return "", err
	}

	// Sign
	sig, err := ag.Sign(key, data)
	if err != nil {
		return "", fmt.Errorf("sign data: %w", err)
	}

	// Encode signature (using SSH wire format)
	return base64.StdEncoding.EncodeToString(gossh.Marshal(sig)), nil
}

// SignChallengeWithAgent signs a base64-encoded challenge using the SSH agent.
// This is a convenience wrapper for challenge-response authentication.
func SignChallengeWithAgent(ag agent.ExtendedAgent, fingerprint, challenge string) (string, error) {
	challengeBytes, err := base64.RawStdEncoding.DecodeString(challenge)
	if err != nil {
		return "", fmt.Errorf("decode challenge: %w", err)
	}

	return SignWithAgent(ag, fingerprint, challengeBytes)
}

// SignWithKeyFile signs data using a private key file.
// Note: Only unencrypted keys are supported. For encrypted keys, use ssh-agent.
func SignWithKeyFile(keyPath string, data []byte) (string, error) {
	keyData, err := os.ReadFile(keyPath) //nolint:gosec // user-provided path expected
	if err != nil {
		return "", fmt.Errorf("read private key: %w", err)
	}

	signer, err := gossh.ParsePrivateKey(keyData)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w (encrypted keys require ssh-agent)", err)
	}

	sig, err := signer.Sign(nil, data)
	if err != nil {
		return "", fmt.Errorf("sign data: %w", err)
	}

	return base64.StdEncoding.EncodeToString(gossh.Marshal(sig)), nil
}

// SignChallengeWithKeyFile signs a base64-encoded challenge using a key file.
func SignChallengeWithKeyFile(keyPath, challenge string) (string, error) {
	challengeBytes, err := base64.RawStdEncoding.DecodeString(challenge)
	if err != nil {
		return "", fmt.Errorf("decode challenge: %w", err)
	}

	return SignWithKeyFile(keyPath, challengeBytes)
}

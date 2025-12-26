// Package ssh provides SSH key utilities for CLI applications.
//
// This package includes:
//   - SSH key discovery (find default key, list all keys)
//   - Public key parsing and fingerprint computation
//   - SSH agent connection and signing
//   - Direct key file signing
//
// # Finding SSH Keys
//
// Find the default SSH key:
//
//	info, err := ssh.FindDefaultKey()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(info.Fingerprint) // SHA256:...
//
// List all SSH keys in ~/.ssh:
//
//	keys, err := ssh.ListLocalKeys()
//	for _, key := range keys {
//	    fmt.Printf("%s: %s\n", key.KeyType, key.Fingerprint)
//	}
//
// # SSH Agent Signing
//
// Sign a challenge using the SSH agent:
//
//	agent, err := ssh.GetAgent()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	sig, err := ssh.SignWithAgent(agent, fingerprint, challengeBytes)
//
// # Direct Key Signing
//
// Sign using a private key file (unencrypted keys only):
//
//	sig, err := ssh.SignWithKeyFile(keyPath, challengeBytes)
//
// # Custom Configuration
//
// Use Config for custom SSH directory or key preferences:
//
//	cfg := ssh.Config{
//	    SSHDir:        "/custom/path/.ssh",
//	    PreferredKeys: []string{"id_ed25519.pub", "id_ecdsa.pub"},
//	}
//
//	info, err := ssh.FindDefaultKeyWithConfig(cfg)
package ssh

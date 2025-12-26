# auth/ssh package

SSH key utilities for CLI authentication.

## Quick Reference

| Type | Purpose |
|------|---------|
| `Config` | SSH directory and key preferences |
| `KeyInfo` | SSH key metadata (path, type, fingerprint, comment) |

## Key Discovery

| Function | Purpose |
|----------|---------|
| `FindDefaultKey()` | Find default SSH key (~/.ssh/id_ed25519, etc.) |
| `FindDefaultKeyWithConfig(cfg)` | Find default key with custom config |
| `ReadPublicKey(path)` | Read and parse a public key file |
| `ParsePublicKey(path, data)` | Parse public key from string |
| `ListLocalKeys()` | List all SSH keys in ~/.ssh |
| `ListLocalKeysWithConfig(cfg)` | List keys with custom config |

## Fingerprinting

| Function | Purpose |
|----------|---------|
| `ComputeFingerprint(blob)` | Compute SHA256 fingerprint from key blob |

## SSH Agent

| Function | Purpose |
|----------|---------|
| `GetAgent()` | Connect to SSH agent via SSH_AUTH_SOCK |
| `ListAgentKeys(agent)` | List keys in agent |
| `FindAgentKeyByFingerprint(agent, fp)` | Find key in agent by fingerprint |

## Signing

| Function | Purpose |
|----------|---------|
| `SignWithAgent(agent, fp, data)` | Sign data with agent key |
| `SignChallengeWithAgent(agent, fp, challenge)` | Sign base64 challenge |
| `SignWithKeyFile(path, data)` | Sign with unencrypted private key |
| `SignChallengeWithKeyFile(path, challenge)` | Sign base64 challenge with key file |

## Errors

| Error | When |
|-------|------|
| `ErrNoSSHAgent` | SSH_AUTH_SOCK not set or agent unavailable |
| `ErrNoSSHKeys` | No SSH keys found in directory |
| `ErrKeyNotFound` | Fingerprint not found in agent |
| `ErrInvalidKeyFormat` | Public key file has invalid format |

## Usage Example

```go
// Find default key
info, err := ssh.FindDefaultKey()
if err != nil {
    log.Fatal(err)
}

// Connect to agent and sign challenge
agent, err := ssh.GetAgent()
if err != nil {
    log.Fatal(err)
}

sig, err := ssh.SignChallengeWithAgent(agent, info.Fingerprint, challenge)
if err != nil {
    log.Fatal(err)
}
```

## Application-Specific Wrappers

Applications using SSH authentication should keep the authentication flow in their own code,
using this package only for key discovery and signing:

```go
// In your application's auth package
func AuthenticateWithSSHKey(ctx context.Context, fingerprint string) (*Credentials, error) {
    // 1. Get challenge from your server (application-specific)
    challenge, err := client.GetSSHChallenge(ctx, fingerprint)

    // 2. Sign challenge using devflow/auth/ssh
    agent, err := ssh.GetAgent()
    signature, err := ssh.SignChallengeWithAgent(agent, fingerprint, challenge)

    // 3. Authenticate with your server (application-specific)
    return client.AuthenticateWithSSHKey(ctx, fingerprint, challenge, signature)
}
```

## File Structure

```
auth/ssh/
├── doc.go           # Package documentation
├── errors.go        # Sentinel errors
├── keys.go          # Key discovery and parsing
├── fingerprint.go   # Fingerprint computation
├── agent.go         # SSH agent connection
├── sign.go          # Signing utilities
└── keys_test.go     # Tests
```

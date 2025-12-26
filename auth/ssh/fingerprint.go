package ssh

import (
	"crypto/sha256"
	"encoding/base64"
)

// ComputeFingerprint computes the SHA256 fingerprint of a key blob.
func ComputeFingerprint(keyBlob []byte) string {
	hash := sha256.Sum256(keyBlob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])
}

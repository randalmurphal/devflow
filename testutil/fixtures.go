// Package testutil provides utilities for testing.
package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// LoadFixture loads a fixture file from the testdata directory.
// The path is relative to the testdata directory.
func LoadFixture(t *testing.T, path string) []byte {
	t.Helper()

	fullPath := filepath.Join("testdata", path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", path, err)
	}

	return data
}

// LoadFixtureString loads a fixture file as a string.
func LoadFixtureString(t *testing.T, path string) string {
	t.Helper()
	return string(LoadFixture(t, path))
}

// LoadJSONFixture loads a fixture file and unmarshals it as JSON.
func LoadJSONFixture[T any](t *testing.T, path string) T {
	t.Helper()

	data := LoadFixture(t, path)

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse JSON fixture %s: %v", path, err)
	}

	return result
}

// MustLoadFixture loads a fixture file, panicking on error.
// Use this for test setup outside of test functions.
func MustLoadFixture(path string) []byte {
	fullPath := filepath.Join("testdata", path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		panic("failed to load fixture " + path + ": " + err.Error())
	}
	return data
}

// WriteFixture writes data to a fixture file.
// Useful for generating test fixtures from real API responses.
func WriteFixture(t *testing.T, path string, data []byte) {
	t.Helper()

	fullPath := filepath.Join("testdata", path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create fixture directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", path, err)
	}
}

// WriteJSONFixture marshals data to JSON and writes it to a fixture file.
func WriteJSONFixture(t *testing.T, path string, v any) {
	t.Helper()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON for fixture %s: %v", path, err)
	}

	WriteFixture(t, path, data)
}

// TempDir creates a temporary directory for the test.
// It returns the directory path and is automatically cleaned up when the test ends.
func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// TempFile creates a temporary file with the given content.
// Returns the file path. File is automatically cleaned up when the test ends.
func TempFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to create temp file %s: %v", name, err)
	}

	return path
}

// TempFileString creates a temporary file with string content.
func TempFileString(t *testing.T, name, content string) string {
	return TempFile(t, name, []byte(content))
}

// CopyFixture copies a fixture file to a temporary location.
// Returns the path to the copy.
func CopyFixture(t *testing.T, fixturePath string) string {
	t.Helper()

	data := LoadFixture(t, fixturePath)
	return TempFile(t, filepath.Base(fixturePath), data)
}

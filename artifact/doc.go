// Package artifact provides storage and lifecycle management for workflow artifacts.
//
// Core types:
//   - Manager: Saves and loads artifacts for workflow runs
//   - LifecycleManager: Handles cleanup, archival, and retention
//   - ReviewResult: Code review findings artifact
//   - TestOutput: Test execution results artifact
//   - LintOutput: Linting results artifact
//   - Specification: Feature specification artifact
//
// Example usage:
//
//	mgr := artifact.NewManager(artifact.Config{
//	    BaseDir:       ".devflow/runs",
//	    CompressAbove: 1024,
//	})
//	err := mgr.SaveArtifact("run-123", "output.json", data)
//	data, err := mgr.LoadArtifact("run-123", "output.json")
package artifact

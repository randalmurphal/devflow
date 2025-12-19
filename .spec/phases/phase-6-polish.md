# Phase 6: Polish & Integration

## Overview

Final polish, documentation, and integration testing.

**Duration**: Week 6
**Dependencies**: Phases 1-5
**Deliverables**: Production-ready library with docs and examples

---

## Goals

1. Complete documentation
2. Create example applications
3. Full integration testing with flowgraph
4. Performance optimization
5. Release preparation

---

## Documentation Tasks

### Task 6.1: API Documentation

Document all public types and functions:

```go
// GitContext manages git operations for a repository.
// It provides worktree management, commit operations, and PR creation.
//
// Example:
//
//     git, err := devflow.NewGitContext("/path/to/repo",
//         devflow.WithGitHub(token),
//     )
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     worktree, err := git.CreateWorktree("feature/my-branch")
//     defer git.CleanupWorktree(worktree)
type GitContext struct {
    // ...
}
```

**Deliverables**:
- [ ] All exported types documented
- [ ] All exported functions documented
- [ ] Examples for common operations
- [ ] godoc generates clean output

### Task 6.2: README.md

Comprehensive README:

```markdown
# devflow

Dev workflow primitives for AI-powered automation.

## Quick Start
## Installation
## Core Concepts
## API Overview
## Examples
## Contributing
```

**Deliverables**:
- [ ] Quick start guide
- [ ] Installation instructions
- [ ] Conceptual overview
- [ ] Links to detailed docs

### Task 6.3: docs/ Directory

```
docs/
├── getting-started.md
├── git-operations.md
├── claude-integration.md
├── transcripts.md
├── artifacts.md
├── flowgraph-integration.md
└── examples/
    ├── ticket-to-pr.md
    ├── code-review.md
    └── custom-workflow.md
```

### Task 6.4: CLAUDE.md Updates

Update project CLAUDE.md with:
- Accurate API reference
- Current examples
- Correct file paths

---

## Example Applications

### Example 1: Ticket-to-PR CLI

```
examples/ticket-to-pr/
├── main.go
├── README.md
└── prompts/
    ├── generate-spec.txt
    └── implement.txt
```

Demonstrates:
- Full workflow execution
- Context injection
- Checkpointing
- Error handling

### Example 2: Code Review Bot

```
examples/code-review/
├── main.go
├── README.md
└── prompts/
    └── review.txt
```

Demonstrates:
- PR diff retrieval
- Review generation
- Comment posting

### Example 3: Custom Workflow

```
examples/custom-workflow/
├── main.go
├── README.md
└── state.go
```

Demonstrates:
- Custom state types
- Custom nodes
- Mixing devflow and custom nodes

---

## Integration Testing

### Task 6.5: Full Stack Integration Tests

```go
func TestFullWorkflow_WithRealGit(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Real git repo
    // Mock Claude (or real with low token limit)
    // Full workflow execution
    // Verify artifacts created
    // Verify transcript recorded
}
```

### Task 6.6: flowgraph Integration Tests

```go
func TestFlowgraphIntegration(t *testing.T) {
    // Verify devflow nodes work with flowgraph
    // Verify checkpointing works
    // Verify resume after crash works
}
```

### Task 6.7: Cross-Platform Testing

- [ ] Test on Linux
- [ ] Test on macOS
- [ ] Test git worktrees work correctly
- [ ] Test path handling

---

## Performance Optimization

### Task 6.8: Profiling

Profile key operations:
- Transcript save/load
- Artifact compression
- Git operations

### Task 6.9: Benchmarks

```go
func BenchmarkTranscriptSave(b *testing.B) { ... }
func BenchmarkArtifactLoad(b *testing.B) { ... }
func BenchmarkContextBuild(b *testing.B) { ... }
```

### Task 6.10: Optimization

Based on profiling:
- [ ] Optimize hot paths
- [ ] Reduce allocations
- [ ] Consider caching where appropriate

---

## Release Preparation

### Task 6.11: Version Tagging

```bash
git tag v0.1.0
git push origin v0.1.0
```

### Task 6.12: CHANGELOG.md

```markdown
# Changelog

## [0.1.0] - 2025-XX-XX

### Added
- GitContext for git operations
- ClaudeCLI wrapper
- TranscriptManager
- ArtifactManager
- Pre-built workflow nodes

### Notes
- Initial release
- Requires Go 1.22+
- Requires Claude CLI installed
```

### Task 6.13: LICENSE

MIT license file.

### Task 6.14: CI/CD

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test -race ./...
      - run: go vet ./...
```

### Task 6.15: goreleaser

```yaml
# .goreleaser.yml
project_name: devflow
builds:
  - skip: true # Library only
```

---

## Quality Checklist

### Code Quality

- [ ] All tests passing
- [ ] Test coverage > 80%
- [ ] No golangci-lint warnings
- [ ] No race conditions (tested with -race)
- [ ] No security vulnerabilities

### Documentation Quality

- [ ] All public APIs documented
- [ ] Examples compile and work
- [ ] README accurate
- [ ] CLAUDE.md up to date

### Release Quality

- [ ] Version tagged
- [ ] CHANGELOG complete
- [ ] LICENSE present
- [ ] CI passing
- [ ] Examples working

---

## File Structure

Final project structure:

```
devflow/
├── .github/
│   └── workflows/
│       └── ci.yml
├── .spec/              # Specifications (this session)
├── docs/
│   ├── getting-started.md
│   ├── git-operations.md
│   └── ...
├── examples/
│   ├── ticket-to-pr/
│   ├── code-review/
│   └── custom-workflow/
├── prompts/            # Embedded default prompts
│   ├── generate-spec.txt
│   ├── implement.txt
│   └── review-code.txt
├── CHANGELOG.md
├── CLAUDE.md
├── LICENSE
├── README.md
├── go.mod
├── git.go
├── claude.go
├── transcript.go
├── artifact.go
├── nodes.go
├── state.go
├── errors.go
└── *_test.go
```

---

## Completion Criteria

- [ ] All documentation complete
- [ ] All examples working
- [ ] Full integration tests passing
- [ ] Performance acceptable
- [ ] CI/CD configured
- [ ] Ready for v0.1.0 release

---

## References

- All ADRs (001-020)
- Phases 1-5
- flowgraph documentation

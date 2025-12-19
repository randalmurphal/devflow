# devflow Specification Planning

## Purpose

This document outlines the complete specification roadmap for devflow - the dev workflow primitives layer for AI-powered development automation.

## Specification Scope

devflow provides four core capabilities:

| Component | Purpose | Key Types |
|-----------|---------|-----------|
| **Git Operations** | Worktrees, branches, commits, PRs | `GitContext`, `PROptions`, `WorktreeInfo` |
| **Claude CLI** | Structured LLM invocation with dev context | `ClaudeCLI`, `RunResult`, `RunOption` |
| **Transcripts** | Recording and storing conversations | `TranscriptManager`, `Transcript`, `Turn` |
| **Artifacts** | Managing run outputs and files | `ArtifactManager`, `ArtifactInfo` |

## Design Philosophy

### Simplicity First

Every decision favors the simplest solution:

1. **Shell out vs library** - Prefer `exec.Command("git", ...)` over go-git unless we need programmatic access
2. **Files vs database** - Transcripts and artifacts are JSON files on disk, not database records
3. **Grep vs search engine** - Search is `grep`, not Elasticsearch
4. **Cron vs scheduler** - Cleanup is a cron job, not a background worker

### Composition Over Complexity

devflow composes with flowgraph rather than reimplementing:

```
User Code
    │
    ▼
flowgraph.Graph[DevState]  ←── Graph orchestration from flowgraph
    │
    ├── devflow.CreateWorktreeNode
    ├── devflow.ImplementNode
    ├── devflow.ReviewNode
    └── devflow.CreatePRNode
```

### Error Handling

Follow the CLAUDE.md philosophy:

| Impact | Action | Example |
|--------|--------|---------|
| Affects output quality | Crash | Missing worktree, context not passing |
| Should fix, output OK | Error log | Artifact cleanup failed |
| Shouldn't happen, OK if not fixed | Warning | Display data unavailable |
| Expected/designed | Debug | Fallback strategy taken |

---

## ADR Index

### Git Operations (001-005)

| ADR | Topic | Status |
|-----|-------|--------|
| 001 | Worktree strategy | Planned |
| 002 | Git operations interface | Planned |
| 003 | Branch naming conventions | Planned |
| 004 | Commit formatting | Planned |
| 005 | PR/MR creation patterns | Planned |

### Claude CLI Integration (006-010)

| ADR | Topic | Status |
|-----|-------|--------|
| 006 | Claude CLI wrapper design | Planned |
| 007 | Prompt management | Planned |
| 008 | Context file handling | Planned |
| 009 | Output parsing | Planned |
| 010 | Session management | Planned |

### Transcript Management (011-014)

| ADR | Topic | Status |
|-----|-------|--------|
| 011 | Transcript format | Planned |
| 012 | Transcript storage | Planned |
| 013 | Transcript replay | Planned |
| 014 | Transcript search | Planned |

### Artifact Management (015-017)

| ADR | Topic | Status |
|-----|-------|--------|
| 015 | Artifact directory structure | Planned |
| 016 | Artifact lifecycle | Planned |
| 017 | Artifact types | Planned |

### Integration (018-020)

| ADR | Topic | Status |
|-----|-------|--------|
| 018 | flowgraph integration | Planned |
| 019 | State design patterns | Planned |
| 020 | Error handling strategy | Planned |

---

## Implementation Phases

| Phase | Focus | Dependencies |
|-------|-------|--------------|
| 1 | Git Primitives | None |
| 2 | Claude CLI Wrapper | None |
| 3 | Transcript Management | Phase 2 |
| 4 | Artifact Management | Phase 3 |
| 5 | Dev Workflow Nodes | Phases 1-4, flowgraph |
| 6 | Polish & Integration | All phases |

---

## Feature Specifications

| Feature | File | Status |
|---------|------|--------|
| Worktree Management | `features/worktree-management.md` | Planned |
| Git Operations | `features/git-operations.md` | Planned |
| Claude CLI | `features/claude-cli.md` | Planned |
| Prompt Loading | `features/prompt-loading.md` | Planned |
| Transcript Recording | `features/transcript-recording.md` | Planned |
| Transcript Replay | `features/transcript-replay.md` | Planned |
| Artifact Storage | `features/artifact-storage.md` | Planned |
| Dev Workflow Nodes | `features/dev-workflow-nodes.md` | Planned |

---

## Node Specifications

Pre-built nodes for common dev workflow operations:

| Node | Purpose | File |
|------|---------|------|
| generate-spec | Generate specs from tickets | `features/nodes/generate-spec.md` |
| implement | Implement code from spec | `features/nodes/implement.md` |
| review-code | Review implementation | `features/nodes/review-code.md` |
| fix-findings | Fix review findings | `features/nodes/fix-findings.md` |
| create-pr | Create pull request | `features/nodes/create-pr.md` |
| run-tests | Execute test suite | `features/nodes/run-tests.md` |
| check-lint | Run linting/type checks | `features/nodes/check-lint.md` |

---

## Key Design Decisions

### 1. Git via exec.Command

**Decision**: Shell out to git binary rather than use go-git library.

**Rationale**:
- Simpler to maintain
- Handles edge cases the same way users do
- No additional dependencies
- Easy to debug (just run the command)

### 2. JSON Files for Storage

**Decision**: Transcripts and artifacts stored as JSON files, not database.

**Rationale**:
- No database dependency for open source use
- Easy to inspect and debug
- grep for search
- Compression for large files

### 3. flowgraph for Orchestration

**Decision**: Build on flowgraph for graph orchestration, don't reimplement.

**Rationale**:
- Single responsibility - devflow handles dev primitives
- flowgraph handles graph execution, checkpointing
- Clean separation of concerns
- Easier to test each layer

### 4. Pre-built Nodes as Functions

**Decision**: Nodes are functions, not types with methods.

**Rationale**:
- Matches flowgraph's `NodeFunc[S]` signature
- Simpler to compose
- Easy to wrap with middleware (transcripts, timing)
- Consistent with Go idioms

---

## Success Criteria

This specification is complete when:

- [ ] All 20 ADRs written and approved
- [ ] All 6 phase specs written
- [ ] All 8 feature specs written
- [ ] All 7 node specs written
- [ ] Integration patterns documented
- [ ] Someone could implement devflow from specs alone
- [ ] Clear integration path with flowgraph documented

---

## Related Documents

| Document | Location | Purpose |
|----------|----------|---------|
| flowgraph spec | flowgraph repo | Foundation layer |
| MASTER_SPEC.md | /tmp/notes/ | Full ecosystem vision |
| IMPLEMENTATION_CHECKLIST.md | /tmp/notes/ | Phase 4-6 cover devflow |
| OVERVIEW.md | docs/ | High-level vision |
| ARCHITECTURE.md | docs/ | Component structure |
| API_REFERENCE.md | docs/ | Initial API sketches |

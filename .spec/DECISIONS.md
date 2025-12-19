# devflow Architecture Decision Records

## Overview

This document indexes all architecture decision records (ADRs) for devflow. Each ADR documents a significant design choice, its context, alternatives considered, and consequences.

## ADR Template

All ADRs follow this structure:

```markdown
# ADR-XXX: Title

## Status
Proposed | Accepted | Deprecated | Superseded

## Context
What is the issue that we're seeing that is motivating this decision?

## Decision
What is the change that we're proposing and/or doing?

## Alternatives Considered
What other options were evaluated?

## Consequences
What becomes easier or more difficult?

## Code Example
Concrete example showing the decision in practice.
```

---

## Decision Index

### Git Operations

| ADR | Title | Status | Summary |
|-----|-------|--------|---------|
| [001](decisions/001-worktree-strategy.md) | Worktree Strategy | Proposed | How to manage parallel worktrees for isolated work |
| [002](decisions/002-git-operations-interface.md) | Git Operations Interface | Proposed | GitContext interface design |
| [003](decisions/003-branch-naming.md) | Branch Naming Conventions | Proposed | Conventions for generated branches |
| [004](decisions/004-commit-formatting.md) | Commit Formatting | Proposed | Commit message structure |
| [005](decisions/005-pr-creation.md) | PR Creation Patterns | Proposed | PR/MR creation patterns |

### Claude CLI Integration

| ADR | Title | Status | Summary |
|-----|-------|--------|---------|
| [006](decisions/006-claude-cli-wrapper.md) | Claude CLI Wrapper | Proposed | How to wrap claude CLI binary |
| [007](decisions/007-prompt-management.md) | Prompt Management | Proposed | How prompts are stored and loaded |
| [008](decisions/008-context-files.md) | Context File Handling | Proposed | How to pass files to Claude |
| [009](decisions/009-output-parsing.md) | Output Parsing | Proposed | Parsing Claude's output |
| [010](decisions/010-session-management.md) | Session Management | Proposed | Multi-turn conversations |

### Transcript Management

| ADR | Title | Status | Summary |
|-----|-------|--------|---------|
| [011](decisions/011-transcript-format.md) | Transcript Format | Proposed | Structure of saved transcripts |
| [012](decisions/012-transcript-storage.md) | Transcript Storage | Proposed | Where/how transcripts are saved |
| [013](decisions/013-transcript-replay.md) | Transcript Replay | Proposed | Replaying for debugging |
| [014](decisions/014-transcript-search.md) | Transcript Search | Proposed | Finding past conversations |

### Artifact Management

| ADR | Title | Status | Summary |
|-----|-------|--------|---------|
| [015](decisions/015-artifact-structure.md) | Artifact Structure | Proposed | Directory layout for artifacts |
| [016](decisions/016-artifact-lifecycle.md) | Artifact Lifecycle | Proposed | Creation, retention, cleanup |
| [017](decisions/017-artifact-types.md) | Artifact Types | Proposed | What types of artifacts we store |

### Integration

| ADR | Title | Status | Summary |
|-----|-------|--------|---------|
| [018](decisions/018-flowgraph-integration.md) | flowgraph Integration | Proposed | How devflow uses flowgraph |
| [019](decisions/019-state-design.md) | State Design Patterns | Proposed | State types for dev workflows |
| [020](decisions/020-error-handling.md) | Error Handling Strategy | Proposed | Error strategy (inherits from flowgraph) |

---

## Key Principles

These principles guide all decisions:

### 1. Simplicity Over Cleverness

> The simplest solution that solves the problem is the right solution.

- Shell out to `git` rather than use go-git
- JSON files rather than database
- grep rather than search engine
- Functions rather than complex types

### 2. Composition Over Reimplementation

> Use flowgraph for what flowgraph does. Add dev-specific value.

- flowgraph handles graph execution
- devflow handles git, Claude, transcripts, artifacts
- Clean boundaries, clear responsibilities

### 3. Explicit Over Implicit

> No magic. Make data flow visible.

- State types are explicit structs
- Context is passed explicitly
- Errors are handled explicitly

### 4. Files Over Services

> For open source, dependencies should be minimal.

- No database required for basic use
- No external services required
- Everything works with just files

---

## Decision Process

1. **Identify** - Recognize a decision needs to be made
2. **Document** - Write the ADR with context and alternatives
3. **Review** - Get feedback on the decision
4. **Decide** - Mark as accepted
5. **Implement** - Reference ADR in code

---

## Superseded Decisions

| ADR | Superseded By | Reason |
|-----|---------------|--------|
| (none yet) | | |

---

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [Michael Nygard's ADR Template](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- flowgraph ADRs (foundation layer decisions)

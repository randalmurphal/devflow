# devflow Specification Changelog

All notable changes to the devflow specification.

## Format

```markdown
## [Version] - YYYY-MM-DD

### Added
- New features or documents

### Changed
- Changes to existing documents

### Deprecated
- Features or patterns being phased out

### Removed
- Removed features or documents

### Fixed
- Corrections to specifications
```

---

## [1.0.0] - 2025-12-19

### Added
- **Foundation Documents**
  - PLANNING.md - Overall spec roadmap with design philosophy
  - DECISIONS.md - ADR index with decision summaries
  - tracking/PROGRESS.md - Progress tracking (42/42 complete)
  - tracking/CHANGELOG.md - Change history
  - SESSION_PROMPT.md - Implementation handoff document

- **Architecture Decision Records** (20 total)
  - Git Operations: 001-005 (worktrees, interface, naming, commits, PRs)
  - Claude CLI: 006-010 (wrapper, prompts, context, parsing, sessions)
  - Transcripts: 011-014 (format, storage, replay, search)
  - Artifacts: 015-017 (structure, lifecycle, types)
  - Integration: 018-020 (flowgraph, state, errors)

- **Phase Specifications** (6 total)
  - Phase 1: Git Primitives (Week 1)
  - Phase 2: Claude CLI Wrapper (Week 2)
  - Phase 3: Transcript Management (Week 3)
  - Phase 4: Artifact Management (Week 4)
  - Phase 5: Workflow Nodes (Week 5)
  - Phase 6: Polish & Integration (Week 6)

- **Feature Specifications** (8 total)
  - worktree-management.md
  - git-operations.md
  - claude-cli.md
  - prompt-loading.md
  - transcript-recording.md
  - transcript-replay.md
  - artifact-storage.md
  - dev-workflow-nodes.md

- **Node Specifications** (7 total in features/nodes/)
  - generate-spec.md - Ticket to specification
  - implement.md - Spec to code
  - review-code.md - Code review
  - fix-findings.md - Address review findings
  - create-pr.md - PR/MR creation
  - run-tests.md - Test execution
  - check-lint.md - Linting and static analysis

- **Knowledge Documents**
  - INTEGRATION_PATTERNS.md - flowgraph integration patterns

### Status
- **Specification Phase**: Complete
- **Total Documents**: 47
- **Ready for**: Implementation Phase 1

---

## Session Log

### 2025-12-19 - Specification Session Start

**Goal**: Create complete specification for devflow

**Context**:
- devflow is the middle layer of three-tier ecosystem
- flowgraph (foundation) - graph orchestration
- devflow (this repo) - dev workflow primitives
- task-keeper (product) - commercial SaaS

**Reference Documents**:
- docs/OVERVIEW.md - High-level vision
- docs/ARCHITECTURE.md - Component structure
- docs/API_REFERENCE.md - Initial API sketches
- /tmp/notes/MASTER_SPEC.md - Full ecosystem vision
- /tmp/notes/IMPLEMENTATION_CHECKLIST.md - Implementation phases

**Design Philosophy**:
- Simplicity is elegance
- Shell out to git, not go-git
- JSON files, not database
- grep, not search engine
- Compose with flowgraph, don't reimplement

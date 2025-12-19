# devflow Implementation Session

## Status: Ready for Implementation

Specification phase complete. All design decisions documented. Ready to write code.

---

## Quick Context

**devflow** = Go library providing dev workflow primitives for AI agents:
- Git operations (worktrees, commits, PRs)
- Claude CLI wrapper
- Transcript management
- Artifact storage
- Pre-built flowgraph nodes

**Depends on**: flowgraph (graph orchestration engine)

---

## What's Been Done

✅ **47 specification documents created:**
- 20 ADRs covering all architectural decisions
- 6 phase specifications with detailed task breakdowns
- 8 feature specifications with API designs
- 7 node specifications with prompts and test cases
- Integration patterns documentation

All specs are in `.spec/`. Read `PLANNING.md` first for philosophy.

---

## Implementation Phases

### Phase 1: Git Primitives (Week 1)
**Start here.** Foundation for everything else.

| Task | File | ADRs |
|------|------|------|
| GitContext interface | `git.go` | 001, 002 |
| CreateWorktree | `git.go` | 001, 003 |
| CleanupWorktree | `git.go` | 001 |
| Commit, Push | `git.go` | 004 |
| GitHub PR creation | `github.go` | 005 |
| GitLab MR creation | `gitlab.go` | 005 |

**Read first**: `.spec/phases/phase-1-git-primitives.md`

### Phase 2: Claude CLI Wrapper (Week 2)
Can run parallel with Phase 1.

| Task | File | ADRs |
|------|------|------|
| ClaudeCLI interface | `claude.go` | 006 |
| Run with prompt | `claude.go` | 006 |
| Prompt template loading | `prompt.go` | 007 |
| Context file handling | `claude.go` | 008 |
| Output parsing | `claude.go` | 009 |
| Session/multi-turn | `claude.go` | 010 |

**Read first**: `.spec/phases/phase-2-claude-cli.md`

### Phase 3: Transcript Management (Week 3)
Can run parallel.

| Task | File | ADRs |
|------|------|------|
| TranscriptManager | `transcript.go` | 011, 012 |
| StartRun, RecordTurn | `transcript.go` | 011 |
| EndRun | `transcript.go` | 011 |
| View/Export | `transcript.go` | 013 |
| Search | `transcript.go` | 014 |

**Read first**: `.spec/phases/phase-3-transcripts.md`

### Phase 4: Artifact Management (Week 4)
Can run parallel.

| Task | File | ADRs |
|------|------|------|
| ArtifactManager | `artifact.go` | 015 |
| SaveArtifact | `artifact.go` | 015, 017 |
| LoadArtifact | `artifact.go` | 015 |
| ListArtifacts | `artifact.go` | 015 |
| Cleanup/retention | `artifact.go` | 016 |

**Read first**: `.spec/phases/phase-4-artifacts.md`

### Phase 5: Workflow Nodes (Week 5)
Requires Phases 1-4 complete.

| Node | File | Spec |
|------|------|------|
| CreateWorktreeNode | `nodes.go` | features/nodes/generate-spec.md |
| GenerateSpecNode | `nodes.go` | features/nodes/generate-spec.md |
| ImplementNode | `nodes.go` | features/nodes/implement.md |
| ReviewNode | `nodes.go` | features/nodes/review-code.md |
| FixFindingsNode | `nodes.go` | features/nodes/fix-findings.md |
| CreatePRNode | `nodes.go` | features/nodes/create-pr.md |
| RunTestsNode | `nodes.go` | features/nodes/run-tests.md |
| CheckLintNode | `nodes.go` | features/nodes/check-lint.md |

**Read first**: `.spec/phases/phase-5-workflow-nodes.md`

### Phase 6: Polish (Week 6)
Final integration and documentation.

- Integration tests
- Documentation
- Examples
- Release prep

**Read first**: `.spec/phases/phase-6-polish.md`

---

## Key Design Decisions

**Read these ADRs before coding:**

| Decision | ADR | Summary |
|----------|-----|---------|
| Shell out to git | 001 | Use `exec.Command("git", ...)` not go-git |
| Shell out to claude | 006 | Use `exec.Command("claude", ...)` not API |
| JSON file storage | 012 | No database, just JSON files |
| grep for search | 014 | No search engine, shell out to grep |
| Context injection | 018 | Pass services via `context.Context` |
| State composition | 019 | Embed standard state structs |

---

## File Structure to Create

```
devflow/
├── git.go              # GitContext implementation
├── github.go           # GitHub PR provider
├── gitlab.go           # GitLab MR provider
├── claude.go           # ClaudeCLI implementation
├── prompt.go           # Prompt template loading
├── transcript.go       # TranscriptManager
├── artifact.go         # ArtifactManager
├── nodes.go            # Pre-built workflow nodes
├── state.go            # DevState and embedded types
├── errors.go           # Error types
├── context.go          # Context key helpers
├── options.go          # Functional options
├── git_test.go         # Unit tests
├── claude_test.go
├── transcript_test.go
├── artifact_test.go
├── nodes_test.go
└── integration/
    ├── git_test.go     # Integration tests
    ├── claude_test.go
    └── workflow_test.go
```

---

## Tracking Progress

Update `.spec/tracking/PROGRESS.md` as you complete items.

Format:
```markdown
| Phase 1 Task | Status | Notes |
|--------------|--------|-------|
| GitContext interface | Complete | git.go:15-45 |
| CreateWorktree | In Progress | |
```

---

## Success Criteria

Phase is complete when:
- [ ] All tasks implemented per spec
- [ ] Unit tests written and passing
- [ ] `go test -race ./...` passes
- [ ] Code follows Go conventions
- [ ] No `TODO` or `FIXME` left without tracking issue

---

## Getting Started

1. **Read** `.spec/PLANNING.md` for design philosophy
2. **Read** `.spec/phases/phase-1-git-primitives.md`
3. **Read** ADRs 001-005 in `.spec/decisions/`
4. **Create** `git.go` with GitContext interface
5. **Implement** CreateWorktree first (foundational)
6. **Test** with real git repo
7. **Update** PROGRESS.md

---

## Questions?

If something in the specs is unclear:
1. Check related ADRs for rationale
2. Check `knowledge/INTEGRATION_PATTERNS.md` for examples
3. Default to simplicity (shell out, JSON files, grep)
4. Flag in PROGRESS.md if blocked

---

## Previous Session Summary

**Completed**: Full specification session
- Created directory structure
- Wrote 20 ADRs with context, decision, alternatives, consequences, code examples
- Wrote 6 phase specs with task breakdowns and acceptance criteria
- Wrote 8 feature specs with API documentation
- Wrote 7 node specs with signatures, prompts, error cases, tests
- Wrote integration patterns document
- All tracked in PROGRESS.md (42/42 complete)

**Next**: Implementation Phase 1 - Git Primitives

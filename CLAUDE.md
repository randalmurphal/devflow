# devflow

**Go library for AI-powered development workflows.** Git operations, Claude CLI integration, transcript management, and artifact storage.

---

## Vision

Dev workflow primitives for AI agents. Builds on flowgraph to provide development-specific functionality. Part of a three-layer ecosystem:

| Layer | Purpose | Repo |
|-------|---------|------|
| flowgraph | Graph orchestration engine | Open source |
| **devflow** | Dev workflow primitives (this repo) | Open source |
| task-keeper | Commercial SaaS product | Commercial |

**Depends on**: flowgraph (for graph orchestration)

---

## Core Components

| Component | Description | Key Type |
|-----------|-------------|----------|
| **GitContext** | Git operations (worktrees, commits, branches) | `GitContext` |
| **ClaudeCLI** | Claude CLI wrapper with devflow conventions | `ClaudeCLI` |
| **TranscriptManager** | Recording and storing conversation transcripts | `TranscriptManager` |
| **ArtifactManager** | Storing run artifacts (files, outputs) | `ArtifactManager` |

---

## Quick Reference

### Git Operations

```go
git := devflow.NewGitContext(repoPath)

// Create worktree for isolated work
worktree, err := git.CreateWorktree("feature/my-branch")
defer git.CleanupWorktree(worktree)

// Commit changes
err = git.Commit("Add feature", "file1.go", "file2.go")

// Create PR (GitHub)
pr, err := git.CreatePR(devflow.PROptions{
    Title: "Add feature",
    Body:  "Description",
    Base:  "main",
})
```

### Claude CLI

```go
claude := devflow.NewClaudeCLI(devflow.ClaudeConfig{
    Timeout:   5 * time.Minute,
    MaxTurns:  10,
})

result, err := claude.Run(ctx, "Implement the feature",
    devflow.WithSystemPrompt("You are an expert Go developer"),
    devflow.WithContext("main.go", "utils.go"),
    devflow.WithWorkDir(worktreePath),
)

// Result contains output, token usage, transcript, created files
fmt.Println(result.Output)
fmt.Println(result.TokensIn, result.TokensOut)
```

### Transcripts

```go
mgr := devflow.NewTranscriptManager(devflow.TranscriptConfig{
    BaseDir: ".devflow/runs",
})

err := mgr.StartRun("run-123", devflow.RunMetadata{
    FlowID: "ticket-to-pr",
    Input:  map[string]any{"ticket": "TK-421"},
})

err = mgr.RecordTurn("run-123", devflow.Turn{
    Role:    "assistant",
    Content: "I'll implement this feature...",
    Tokens:  1500,
})

err = mgr.EndRun("run-123", devflow.RunStatusCompleted)
```

### Artifacts

```go
artifacts := devflow.NewArtifactManager(devflow.ArtifactConfig{
    BaseDir:       ".devflow/runs",
    CompressAbove: 1024, // Compress files > 1KB
})

// Save artifact
err := artifacts.SaveArtifact("run-123", "output.json", data)

// Load artifact
data, err := artifacts.LoadArtifact("run-123", "output.json")
```

---

## Project Structure

```
devflow/
├── git.go              # GitContext interface and implementation
├── github.go           # GitHub PR operations
├── gitlab.go           # GitLab MR operations
├── claude.go           # Claude CLI wrapper
├── transcript.go       # Transcript management
├── artifact.go         # Artifact storage
├── errors.go           # Error types
└── tests/
    └── integration/    # Integration tests (require git, Claude)
```

---

## Integration with flowgraph

devflow provides nodes for flowgraph graphs:

```go
import (
    "github.com/yourorg/flowgraph"
    "github.com/yourorg/devflow"
)

type DevState struct {
    TicketID string
    Spec     *devflow.Spec
    Worktree string
    PR       *devflow.PullRequest
}

graph := flowgraph.NewGraph[DevState]().
    AddNode("create-worktree", devflow.CreateWorktreeNode).
    AddNode("generate-spec", devflow.GenerateSpecNode).
    AddNode("implement", devflow.ImplementNode).
    AddNode("create-pr", devflow.CreatePRNode).
    AddEdge("create-worktree", "generate-spec").
    AddEdge("generate-spec", "implement").
    AddEdge("implement", "create-pr").
    AddEdge("create-pr", flowgraph.END).
    SetEntry("create-worktree")
```

---

## Directory Conventions

```
.devflow/
├── runs/
│   └── 2025-01-15-ticket-to-pr-TK421/
│       ├── transcript.json      # Conversation log
│       ├── metadata.json        # Run metadata
│       ├── artifacts/           # Generated files
│       │   ├── spec.md
│       │   └── output.json
│       └── state-checkpoints/   # flowgraph checkpoints
│           ├── generate-spec.json
│           └── implement.json
├── prompts/                     # Prompt templates
│   ├── spec-generation.txt
│   └── implementation.txt
└── config.json                  # devflow configuration
```

---

## Error Handling

| Error | When | Handling |
|-------|------|----------|
| `ErrWorktreeExists` | Worktree already exists | Clean up or use existing |
| `ErrGitDirty` | Uncommitted changes | Stash or abort |
| `ErrClaudeTimeout` | Claude CLI timed out | Retry with longer timeout |
| `ErrTranscriptNotFound` | Run ID doesn't exist | Check run ID |

---

## Testing

```bash
go test -race ./...                    # Unit tests
go test -race -tags=integration ./...  # With real git/Claude
```

**Coverage targets**: 85% overall

---

## Dependencies

```bash
# Core
go get github.com/yourorg/flowgraph  # Graph orchestration

# Git operations
go get github.com/go-git/go-git/v5   # Pure Go git

# GitHub/GitLab
go get github.com/google/go-github/v57
go get github.com/xanzy/go-gitlab
```

---

## References

| Doc | Purpose |
|-----|---------|
| `docs/OVERVIEW.md` | Detailed vision and concepts |
| `docs/ARCHITECTURE.md` | Component design, data flow |
| `docs/API_REFERENCE.md` | Complete public API |
| `docs/GIT_OPERATIONS.md` | Git workflow patterns |
| `docs/TRANSCRIPT_FORMAT.md` | Transcript storage format |

---

## Related Repos

- **flowgraph**: Foundation layer (graph orchestration)
- **task-keeper**: Product layer (commercial SaaS)
- **ai-devtools/ensemble**: Python reference implementation

---

## Specification Documents

Complete specifications are in `.spec/`. **Read these before implementing.**

```
.spec/
├── PLANNING.md              # Overall roadmap and design philosophy
├── DECISIONS.md             # ADR index with decision summaries
├── SESSION_PROMPT.md        # Current session handoff/instructions
├── decisions/               # 20 Architecture Decision Records
│   ├── 001-020              # Git, Claude CLI, Transcripts, Artifacts, Integration
├── phases/                  # 6 implementation phases (6 weeks)
│   ├── phase-1-git-primitives.md
│   ├── phase-2-claude-cli.md
│   ├── phase-3-transcripts.md
│   ├── phase-4-artifacts.md
│   ├── phase-5-workflow-nodes.md
│   └── phase-6-polish.md
├── features/                # 8 feature specifications
│   ├── worktree-management.md
│   ├── git-operations.md
│   ├── claude-cli.md
│   ├── prompt-loading.md
│   ├── transcript-recording.md
│   ├── transcript-replay.md
│   ├── artifact-storage.md
│   ├── dev-workflow-nodes.md
│   └── nodes/               # 7 node specifications
│       ├── generate-spec.md
│       ├── implement.md
│       ├── review-code.md
│       ├── fix-findings.md
│       ├── create-pr.md
│       ├── run-tests.md
│       └── check-lint.md
├── knowledge/
│   └── INTEGRATION_PATTERNS.md  # flowgraph integration patterns
└── tracking/
    ├── PROGRESS.md          # Implementation progress
    └── CHANGELOG.md         # Change history
```

### Implementation Order

Follow phases in order. Each phase builds on the previous:

| Phase | Focus | Dependencies |
|-------|-------|--------------|
| 1 | Git Primitives | None (foundation) |
| 2 | Claude CLI | None (parallel with Phase 1) |
| 3 | Transcripts | None (parallel) |
| 4 | Artifacts | None (parallel) |
| 5 | Workflow Nodes | Phases 1-4 complete |
| 6 | Polish | Phase 5 complete |

### Key Design Decisions

- **Shell out to git** (ADR-001): Don't use go-git for worktrees, shell out to git binary
- **Shell out to claude** (ADR-006): Wrap the claude CLI, don't use API directly
- **JSON files for storage** (ADR-012): Simple file-based storage, no database
- **grep for search** (ADR-014): Use grep for transcript search, not a search engine
- **Context injection** (ADR-018): Pass services via context.Context, not state

### Before Implementing

1. Read the relevant ADR in `decisions/`
2. Read the feature spec in `features/`
3. Read the phase spec in `phases/`
4. Check `PLANNING.md` for design philosophy
5. Update `tracking/PROGRESS.md` as you complete items

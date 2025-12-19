# devflow Integration Requirements

## Overview

devflow depends on flowgraph for all LLM-related functionality. This document defines the contract between the two projects and what must be completed before devflow can ship v1.0.

**Key Principle**: If it's LLM-related, it belongs in flowgraph. devflow imports, never implements.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Application                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         devflow                                  │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ Git Ops      │ │ Transcripts  │ │ Artifacts    │             │
│  │ (own impl)   │ │ (own impl)   │ │ (own impl)   │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ Workflow     │ │ Notifications│ │ State Types  │             │
│  │ Nodes        │ │ (own impl)   │ │ (own impl)   │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│                              │                                   │
│                              │ IMPORTS                           │
│                              ▼                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        flowgraph                                 │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ LLM Client   │ │ Context      │ │ Prompt       │             │
│  │ Interface    │ │ Builder      │ │ Templates    │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ ClaudeCLI    │ │ Graph Engine │ │ Checkpointing│             │
│  │ Implementation│ │              │ │              │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

---

## flowgraph Contract: Required Features

The following features MUST exist in flowgraph before devflow can complete its integration. If any are missing, **STOP AND ESCALATE** - do not implement in devflow.

### LLM Client Interface (`pkg/flowgraph/llm/`)

| Feature | Required | flowgraph Status | Notes |
|---------|----------|-----------------|-------|
| `llm.Client` interface | ✅ Yes | ✅ Complete | `Complete()` and `Stream()` methods |
| `CompletionRequest` type | ✅ Yes | ✅ Complete | Messages, SystemPrompt, Model, etc. |
| `CompletionResponse` type | ✅ Yes | ✅ Complete | Content, Usage, FinishReason, etc. |
| `TokenUsage` type | ✅ Yes | ✅ Complete | InputTokens, OutputTokens, TotalTokens |
| `MockClient` for testing | ✅ Yes | ✅ Complete | For devflow node tests |

### Claude CLI Implementation (`pkg/flowgraph/llm/claude_cli.go`)

| Feature | Required | flowgraph Status | Notes |
|---------|----------|-----------------|-------|
| JSON output parsing | ✅ Yes | ✅ **Complete** | `--output-format json` default |
| Token/cost extraction | ✅ Yes | ✅ **Complete** | Full JSON response parsing |
| `SessionID` in response | ✅ Yes | ✅ **Complete** | For multi-turn tracking |
| `CostUSD` in response | ✅ Yes | ✅ **Complete** | For budget tracking |
| `WithSessionID(id)` | ✅ Yes | ✅ **Complete** | Use specific session |
| `WithContinue()` | ✅ Yes | ✅ **Complete** | Continue last session |
| `WithResume(id)` | ✅ Yes | ✅ **Complete** | Resume specific session |
| `WithMaxTurns(n)` | ✅ Yes | ✅ **Complete** | Limit agentic turns |
| `WithSystemPrompt(s)` | ✅ Yes | ✅ **Complete** | Set system prompt |
| `WithAppendSystemPrompt(s)` | ✅ Yes | ✅ **Complete** | Append to system prompt |
| `WithAllowedTools(tools)` | ✅ Yes | ✅ **Complete** | Whitelist tools |
| `WithDisallowedTools(tools)` | ✅ Yes | ✅ **Complete** | Blacklist tools |
| `WithDangerouslySkipPermissions()` | ✅ Yes | ✅ **Complete** | Non-interactive mode |
| `WithMaxBudgetUSD(amount)` | ✅ Yes | ✅ **Complete** | Cap spending |
| `WithWorkdir(dir)` | ✅ Yes | ✅ **Complete** | Working directory |
| `WithTimeout(d)` | ✅ Yes | ✅ **Complete** | Command timeout |
| `WithAddDirs(dirs)` | ⚪ Optional | ✅ **Complete** | Additional directories |
| `WithFallbackModel(model)` | ⚪ Optional | ✅ **Complete** | Fallback on overload |

### Context Building (`pkg/flowgraph/llm/context.go` - NEW)

**These features currently exist in devflow but SHOULD move to flowgraph:**

| Feature | Required | flowgraph Status | devflow Has | Notes |
|---------|----------|-----------------|-------------|-------|
| `ContextBuilder` | ✅ Yes | ❌ Missing | ✅ Yes | File context aggregation |
| `ContextLimits` | ✅ Yes | ❌ Missing | ✅ Yes | MaxFileSize, MaxTotalSize, MaxFileCount |
| `AddFile(path)` | ✅ Yes | ❌ Missing | ✅ Yes | Add single file |
| `AddGlob(pattern)` | ✅ Yes | ❌ Missing | ✅ Yes | Add files matching pattern |
| `AddContent(path, content)` | ✅ Yes | ❌ Missing | ✅ Yes | Add in-memory content |
| `Build()` | ✅ Yes | ❌ Missing | ✅ Yes | Generate formatted context |
| Binary file detection | ✅ Yes | ❌ Missing | ✅ Yes | Skip/summarize binary files |
| MIME type detection | ⚪ Optional | ❌ Missing | ✅ Yes | For binary file description |
| `FileSelector` | ⚪ Optional | ❌ Missing | ✅ Yes | Include/exclude patterns |

### Prompt Templates (`pkg/flowgraph/llm/prompt.go` - NEW)

**These features currently exist in devflow but SHOULD move to flowgraph:**

| Feature | Required | flowgraph Status | devflow Has | Notes |
|---------|----------|-----------------|-------------|-------|
| `PromptLoader` | ✅ Yes | ❌ Missing | ✅ Yes | Load prompts from files |
| Go template support | ✅ Yes | ❌ Missing | ✅ Yes | Variable substitution |
| Embedded prompts | ✅ Yes | ❌ Missing | ✅ Yes | `//go:embed` support |
| Search directories | ✅ Yes | ❌ Missing | ✅ Yes | Multiple search paths |
| Template caching | ✅ Yes | ❌ Missing | ✅ Yes | Performance |
| Custom template funcs | ⚪ Optional | ❌ Missing | ✅ Yes | `AddFunc()` |
| `PromptBuilder` | ⚪ Optional | ❌ Missing | ✅ Yes | Programmatic construction |

---

## ESCALATION PROTOCOL

When working on devflow integration, follow this protocol:

### If flowgraph is missing a required feature:

1. **DO NOT implement it in devflow**
2. **Document the blocker** in `.spec/tracking/BLOCKERS.md`
3. **Create a task** for flowgraph to add the feature
4. **Inform the user** with this message:

```
BLOCKER: devflow cannot proceed with [feature] because flowgraph is missing:
- [specific missing feature]

Required action:
1. Complete flowgraph Phase 6 LLM enhancements
2. Add [specific feature] to flowgraph
3. Return to devflow integration

Do not implement LLM functionality in devflow.
```

### If flowgraph feature exists but API doesn't match:

1. **Prefer adapting devflow** to flowgraph's API
2. **If flowgraph's API is insufficient**, escalate to add features to flowgraph
3. **Document the adapter pattern** if needed

---

## devflow Refactoring Plan

Once flowgraph has all required features, devflow must refactor:

### Phase A: Remove Duplicate LLM Code

| File | Action | Notes |
|------|--------|-------|
| `claude.go` | **DELETE** | Replace with flowgraph import |
| `claude_test.go` | **DELETE** | Tests move to flowgraph |
| `prompt.go` | **MIGRATE** | Move to flowgraph, delete from devflow |
| `prompt_test.go` | **MIGRATE** | Move to flowgraph, delete from devflow |
| `context.go` (ContextBuilder) | **MIGRATE** | Move to flowgraph, delete from devflow |

### Phase B: Update Context Injection

| Current | New |
|---------|-----|
| `WithClaudeCLI(ctx, *ClaudeCLI)` | `WithLLMClient(ctx, llm.Client)` |
| `ClaudeFromContext(ctx)` | `LLMFromContext(ctx)` |
| `MustClaudeFromContext(ctx)` | `MustLLMFromContext(ctx)` |
| `WithPromptLoader(ctx, *PromptLoader)` | Use flowgraph's `llm.WithPromptLoader` |

### Phase C: Update Nodes

All nodes that call Claude must be updated:

```go
// BEFORE (devflow's ClaudeCLI)
claude := ClaudeFromContext(ctx)
result, err := claude.Run(ctx, prompt, WithSystemPrompt(...))

// AFTER (flowgraph's llm.Client)
client := llm.ClientFromContext(ctx)
resp, err := client.Complete(ctx, llm.CompletionRequest{
    SystemPrompt: "...",
    Messages: []llm.Message{{Role: llm.RoleUser, Content: prompt}},
})
```

### Phase D: Update DevServices

```go
// BEFORE
type DevServices struct {
    Git         *GitContext
    Claude      *ClaudeCLI      // devflow type
    Transcripts TranscriptManager
    Artifacts   *ArtifactManager
    Prompts     *PromptLoader   // devflow type
}

// AFTER
type DevServices struct {
    Git         *GitContext
    LLM         llm.Client      // flowgraph type
    Transcripts TranscriptManager
    Artifacts   *ArtifactManager
    // PromptLoader removed - use flowgraph's
}
```

---

## Notification Node Design

Add a notification system for workflow events:

### Interface

```go
// Notifier sends notifications about workflow events
type Notifier interface {
    Notify(ctx context.Context, event NotificationEvent) error
}

// NotificationEvent describes what happened
type NotificationEvent struct {
    Type      EventType
    RunID     string
    FlowID    string
    NodeID    string
    Message   string
    Severity  Severity
    Timestamp time.Time
    Metadata  map[string]any
}

type EventType string
const (
    EventRunStarted    EventType = "run_started"
    EventRunCompleted  EventType = "run_completed"
    EventRunFailed     EventType = "run_failed"
    EventNodeStarted   EventType = "node_started"
    EventNodeCompleted EventType = "node_completed"
    EventNodeFailed    EventType = "node_failed"
    EventReviewNeeded  EventType = "review_needed"
    EventPRCreated     EventType = "pr_created"
)

type Severity string
const (
    SeverityInfo    Severity = "info"
    SeverityWarning Severity = "warning"
    SeverityError   Severity = "error"
)
```

### Implementations

```go
// SlackNotifier sends to Slack webhook
type SlackNotifier struct {
    WebhookURL string
    Channel    string
}

// WebhookNotifier sends to generic webhook
type WebhookNotifier struct {
    URL     string
    Headers map[string]string
}

// LogNotifier logs notifications (for testing/debugging)
type LogNotifier struct {
    Logger *slog.Logger
}

// MultiNotifier sends to multiple notifiers
type MultiNotifier struct {
    Notifiers []Notifier
}
```

### Context Injection

```go
func WithNotifier(ctx context.Context, n Notifier) context.Context
func NotifierFromContext(ctx context.Context) Notifier
```

### Node Implementation

```go
// NotifyNode sends a notification based on current state
func NotifyNode(ctx context.Context, state DevState) (DevState, error) {
    notifier := NotifierFromContext(ctx)
    if notifier == nil {
        return state, nil // No-op if no notifier
    }

    event := NotificationEvent{
        Type:      determineEventType(state),
        RunID:     state.RunID,
        FlowID:    state.FlowID,
        Timestamp: time.Now(),
        Metadata:  buildMetadata(state),
    }

    if err := notifier.Notify(ctx, event); err != nil {
        // Log but don't fail the workflow
        slog.Warn("notification failed", "error", err, "event", event.Type)
    }

    return state, nil
}
```

---

## Test Coverage Requirements

### Current State

| Package | Coverage | Target | Gap |
|---------|----------|--------|-----|
| Overall | 52.3% | 80% | 27.7% |

### Coverage Plan by File

| File | Current | Target | Priority | Notes |
|------|---------|--------|----------|-------|
| `git.go` | ~60% | 85% | High | Core functionality |
| `branch.go` | ~70% | 90% | Medium | Simple logic |
| `commit.go` | ~70% | 90% | Medium | Simple logic |
| `pr.go` | ~50% | 80% | High | Interface + builder |
| `github.go` | ~40% | 70% | Medium | External API |
| `gitlab.go` | ~40% | 70% | Medium | External API |
| `claude.go` | ~50% | **DELETE** | - | Moving to flowgraph |
| `prompt.go` | ~60% | **DELETE** | - | Moving to flowgraph |
| `context.go` | ~40% | 85% | High | Core injection |
| `transcript.go` | ~60% | 85% | High | Core types |
| `transcript_store.go` | ~60% | 85% | High | Core storage |
| `transcript_search.go` | ~50% | 80% | Medium | Search functionality |
| `transcript_view.go` | ~50% | 80% | Medium | View/export |
| `artifact.go` | ~60% | 85% | High | Core storage |
| `artifact_types.go` | ~70% | 90% | Medium | Simple types |
| `artifact_lifecycle.go` | ~50% | 80% | Medium | Lifecycle mgmt |
| `state.go` | ~60% | 90% | High | Core state |
| `nodes.go` | ~50% | 80% | High | Core nodes |

### Test Types Needed

1. **Unit Tests** - All exported functions
2. **Integration Tests** - Git operations with real repo (skip in CI)
3. **Mock Tests** - Nodes with mock LLM client
4. **Error Path Tests** - All error conditions
5. **Edge Case Tests** - Empty inputs, max limits, etc.

### Missing Test Scenarios

| Component | Missing Tests |
|-----------|---------------|
| GitContext | Worktree cleanup failure, concurrent operations |
| Nodes | All nodes with mock LLM, retry logic, timeout |
| Transcripts | Compression, concurrent access, large files |
| Artifacts | Archive/restore, lifecycle cleanup |
| State | All validation combinations |

---

## Documentation Requirements

### Files to Update

| File | Status | Action |
|------|--------|--------|
| `README.md` | Exists | Update with flowgraph dependency |
| `CLAUDE.md` | Exists | Update integration section |
| `docs/ARCHITECTURE.md` | Exists | Add flowgraph integration diagram |
| `docs/API_REFERENCE.md` | Exists | Update for new API |
| `.spec/SESSION_PROMPT.md` | Exists | Full rewrite for Phase 6 |

### New Files to Create

| File | Purpose |
|------|---------|
| `docs/FLOWGRAPH_INTEGRATION.md` | How devflow uses flowgraph |
| `docs/NOTIFICATIONS.md` | Notification system guide |
| `examples/with-flowgraph/` | Example using flowgraph LLM |
| `examples/notifications/` | Example with notifications |

---

## Completion Checklist

### Before Integration Can Start

- [ ] flowgraph Phase 6 complete
- [ ] flowgraph has JSON output parsing
- [ ] flowgraph has session management
- [ ] flowgraph has all ClaudeOptions
- [ ] flowgraph has ContextBuilder (or we migrate it)
- [ ] flowgraph has PromptLoader (or we migrate it)

### devflow Refactoring

- [ ] Remove claude.go
- [ ] Remove prompt.go (after migration)
- [ ] Remove ContextBuilder from context.go (after migration)
- [ ] Update context injection to use llm.Client
- [ ] Update all nodes to use flowgraph LLM
- [ ] Update DevServices struct
- [ ] Update all tests

### New Features

- [ ] Implement Notifier interface
- [ ] Implement SlackNotifier
- [ ] Implement WebhookNotifier
- [ ] Implement LogNotifier
- [ ] Implement MultiNotifier
- [ ] Implement NotifyNode
- [ ] Add notification context injection
- [ ] Add notification tests

### Test Coverage

- [ ] Achieve 80% overall coverage
- [ ] All nodes tested with mock LLM
- [ ] All error paths tested
- [ ] Integration tests for git ops
- [ ] Notification tests

### Documentation

- [ ] Update README.md
- [ ] Update CLAUDE.md
- [ ] Update ARCHITECTURE.md
- [ ] Create FLOWGRAPH_INTEGRATION.md
- [ ] Create NOTIFICATIONS.md
- [ ] Create examples

---

## Timeline Estimate

| Phase | Effort | Blocked By |
|-------|--------|------------|
| Wait for flowgraph | - | flowgraph Phase 6 |
| Migrate ContextBuilder to flowgraph | 2 hours | flowgraph Phase 6 |
| Migrate PromptLoader to flowgraph | 2 hours | flowgraph Phase 6 |
| Remove devflow duplicates | 1 hour | Migrations complete |
| Update context injection | 2 hours | Duplicates removed |
| Update all nodes | 4 hours | Injection updated |
| Implement notifications | 4 hours | None |
| Test coverage improvements | 8 hours | None |
| Documentation | 4 hours | All code complete |
| **Total** | **~27 hours** | flowgraph Phase 6 |

---

## Questions for User

1. Should ContextBuilder/PromptLoader migrate to flowgraph, or stay in devflow and just use flowgraph's LLM client?
2. What notification backends are highest priority? (Slack, webhook, email, etc.)
3. Should transcripts also move to flowgraph as a generic observability feature?

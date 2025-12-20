# workflow package

Workflow state and node implementations for AI-powered development automation.

## Quick Reference

| Type | Purpose |
|------|---------|
| `State` | Complete workflow execution state |
| `Ticket` | External ticket reference (Jira, GitHub) |
| `NodeFunc` | Function signature for workflow nodes |
| `NodeConfig` | Configuration for node behavior |

## State Components (embedded in State)

| Component | Purpose |
|-----------|---------|
| `GitState` | Worktree, branch, base branch |
| `SpecState` | Generated specification |
| `ImplementState` | Implementation, file changes |
| `ReviewState` | Review result, attempts |
| `PullRequestState` | Created PR info |
| `TestState` | Test execution results |
| `LintState` | Lint check results |
| `MetricsState` | Token usage, cost, duration |

## Workflow Nodes

| Node | Purpose | Requires |
|------|---------|----------|
| `CreateWorktreeNode` | Create isolated worktree | git context |
| `CleanupNode` | Remove worktree | git context |
| `GenerateSpecNode` | Generate spec from ticket | LLM client |
| `ImplementNode` | Implement from spec | LLM client, worktree |
| `ReviewNode` | Review implementation | LLM client |
| `FixFindingsNode` | Fix review issues | LLM client |
| `RunTestsNode` | Execute tests | runner |
| `CheckLintNode` | Run linting | runner |
| `CreatePRNode` | Create pull request | git, pr provider |
| `NotifyNode` | Send notification | notifier |

## Node Wrappers

```go
// Add retry logic
workflow.WithRetry(node, maxAttempts)

// Record to transcript
workflow.WithTranscript(node, "step-name")

// Track execution time
workflow.WithTiming(node)
```

## State Validation

```go
err := state.Validate(
    workflow.RequireTicket,
    workflow.RequireWorktree,
    workflow.RequireSpec,
)
```

## Review Routing

```go
if state.NeedsReviewFix() && state.CanRetryReview(3) {
    return "fix"
}
return flowgraph.END
```

## File Structure

```
workflow/
├── state.go      # State, Ticket, state components
├── node.go       # NodeFunc, NodeConfig, wrappers
├── worktree.go   # CreateWorktreeNode, CleanupNode
├── spec.go       # GenerateSpecNode
├── implement.go  # ImplementNode
├── review.go     # ReviewNode, FixFindingsNode
├── testing.go    # RunTestsNode
├── lint.go       # CheckLintNode
├── pr.go         # CreatePRNode
└── notify.go     # NotifyNode
```

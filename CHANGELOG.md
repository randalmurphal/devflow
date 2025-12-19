# Changelog

All notable changes to devflow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-01-15

### Added

#### Git Operations (Phase 1)
- `GitContext` for managing git repositories
- Worktree management: `CreateWorktree`, `CleanupWorktree`, `ListWorktrees`
- Branch operations: `CreateBranch`, `SwitchBranch`, `DeleteBranch`
- Commit operations: `Commit`, `CommitAll`
- Remote operations: `Push`, `Pull`, `Fetch`, `GetRemoteURL`
- Diff operations: `Diff`, `DiffStaged`, `HasStagedChanges`
- Git status: `IsClean`, `IsBranchPushed`, `HeadCommit`
- PR providers: GitHub and GitLab support via interfaces
- `PRBuilder` for fluent PR configuration
- `BranchNamer` for consistent branch naming conventions
- `CommitMessage` for conventional commit formatting

#### LLM Integration (Phase 2)
- Integration with flowgraph's `llm.Client` interface
- Context injection: `WithLLMClient`, `LLMFromContext`, `MustLLMFromContext`
- `ContextBuilder` for assembling file context for LLM calls
- `PromptLoader` for template-based prompts
- Support for `llm.MockClient` in tests

#### Transcript Management (Phase 3)
- `TranscriptManager` interface for recording conversations
- `FileTranscriptStore` implementation with JSON storage
- Turn recording with token tracking
- Tool call recording
- Cost tracking
- `TranscriptSearcher` with grep-based content search
- `TranscriptViewer` for display and export
- Markdown export functionality
- Transcript diff comparison

#### Artifact Storage (Phase 4)
- `ArtifactManager` for storing workflow artifacts
- Automatic compression for large artifacts
- Type-specific save/load: `SaveSpec`, `SaveReview`, `SaveTestOutput`, `SaveLintOutput`, `SaveDiff`, `SaveJSON`
- `LifecycleManager` for archival and cleanup
- Retention policies with configurable days
- Archive/restore functionality
- Disk usage statistics

#### Workflow Nodes (Phase 5)
- `DevState` for workflow state management
- State validation with `StateRequirement`
- Embeddable state components: `GitState`, `SpecState`, `ImplementState`, `ReviewState`, etc.
- 9 workflow nodes:
  - `CreateWorktreeNode` - Set up isolated workspace
  - `GenerateSpecNode` - Generate specifications from tickets
  - `ImplementNode` - Implement code from specs
  - `ReviewNode` - Review implementation
  - `FixFindingsNode` - Fix review findings
  - `RunTestsNode` - Execute tests
  - `CheckLintNode` - Run linters
  - `CreatePRNode` - Create pull requests
  - `CleanupNode` - Clean up worktrees
  - `NotifyNode` - Send notifications
- Node wrappers: `WithRetry`, `WithTiming`
- `ReviewRouter` for review-based branching
- Prompt formatters for each node type

#### Notification System (Phase 6)
- `Notifier` interface for workflow events
- `SlackNotifier` for Slack webhook notifications
- `WebhookNotifier` for generic webhook support
- `LogNotifier` for logging notifications
- `MultiNotifier` for combining notifiers
- `NopNotifier` for no-op notifications
- Event types: `RunStarted`, `RunCompleted`, `RunFailed`, `PRCreated`, `ReviewNeeded`, `NodeStarted`, `NodeCompleted`
- Context injection: `WithNotifier`, `NotifierFromContext`, `MustNotifierFromContext`

#### Service Infrastructure
- `DevServices` for convenient service bundling
- Context injection pattern for all services
- `InjectAll` for bulk service injection

### Changed

- Migrated from custom `ClaudeCLI` to flowgraph's `llm.Client` interface
- Old `WithClaudeCLI`/`ClaudeFromContext` helpers deprecated (still functional)

### Deprecated

- `ClaudeCLI` struct - use flowgraph's `llm.NewClaudeCLI` instead
- `WithClaudeCLI` - use `WithLLMClient` instead
- `ClaudeFromContext` - use `LLMFromContext` instead

### Technical Details

- Go 1.21+ required
- Dependencies: flowgraph, go-github, go-gitlab, oauth2
- Test coverage: ~60%
- All tests pass with race detection

[Unreleased]: https://github.com/anthropic/devflow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/anthropic/devflow/releases/tag/v0.1.0

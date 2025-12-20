# context package

Dependency injection for workflow services via context.Context.

## Quick Reference

| Type | Purpose |
|------|---------|
| `Services` | Collection of all devflow services |
| `ContextBuilder` | Builds LLM context from files |
| `FileSelector` | Selects files for context |
| `ContextLimits` | Token and size limits |

## Injection Functions

| Function Pair | Service |
|---------------|---------|
| `WithGit` / `Git` / `MustGit` | Git context |
| `WithLLM` / `LLM` / `MustLLM` | LLM client (flowgraph) |
| `WithTranscript` / `Transcript` / `MustTranscript` | Transcript manager |
| `WithArtifact` / `Artifact` / `MustArtifact` | Artifact manager |
| `WithPrompt` / `Prompt` / `MustPrompt` | Prompt loader |
| `WithRunner` / `Runner` / `GetRunner` | Command runner (testing) |
| `WithPR` / `PR` / `MustPR` | PR provider |

**Note:** Notifier uses `notify.WithNotifier` / `notify.NotifierFromContext` from the notify package.

## Services Struct

```go
services := &context.Services{
    Git:         gitCtx,
    LLM:         llmClient,
    Transcripts: transcriptMgr,
    Artifacts:   artifactMgr,
    Notifier:    notifier,
}

// Inject all at once
ctx := services.InjectAll(ctx)
```

## Individual Injection

```go
ctx = context.WithGit(ctx, gitCtx)
ctx = context.WithLLM(ctx, llmClient)

// Retrieve
git := context.Git(ctx)
llm := context.LLM(ctx)
```

## Context Builder

```go
builder := context.NewBuilder(context.BuilderConfig{
    Limits: context.ContextLimits{
        MaxTokens:   100000,
        MaxFileSize: 1 << 20, // 1MB
    },
})

// Add files
builder.AddFile("main.go", content)
builder.AddDirectory(repoPath, selector)

// Build context string
result, err := builder.Build()
```

## File Structure

```
context/
├── context.go   # Injection functions (With*/Get*/Must*)
├── services.go  # Services struct, InjectAll, NewServices
├── builder.go   # ContextBuilder, FileSelector, ContextLimits
└── doc.go       # Package documentation
```

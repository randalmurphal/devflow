// Package context provides dependency injection for workflow services.
//
// Core types:
//   - Services: Collection of all devflow services for injection
//   - ContextBuilder: Builds LLM context from files with size limits
//   - FileSelector: Selects files for context based on patterns
//   - ContextLimits: Token and size limits for context building
//
// Context injection functions:
//   - WithGit/Git: Git context injection
//   - WithLLM/LLM: LLM client injection (flowgraph claude.Client)
//   - WithTranscript/Transcript: Transcript manager injection
//   - WithArtifact/Artifact: Artifact manager injection
//   - WithNotifier/Notifier: Notifier injection
//   - WithRunner/Runner: Command runner injection (for testing)
//
// Example usage:
//
//	services := &context.Services{
//	    Git:      gitCtx,
//	    LLM:      llmClient,
//	    Notifier: slackNotifier,
//	}
//	ctx := services.InjectAll(ctx)
//
//	// Later, retrieve services
//	git := context.Git(ctx)
//	llm := context.LLM(ctx)
package context

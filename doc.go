// Package devflow provides development workflow primitives for AI-powered automation.
//
// The package is organized into subpackages by domain:
//
//   - git: Git repository operations, worktrees, branches, commits
//   - pr: Pull request creation for GitHub and GitLab
//   - transcript: AI conversation transcript recording and search
//   - artifact: Workflow artifact storage and lifecycle management
//   - workflow: Workflow state and node implementations
//   - notify: Notification services (Slack, webhook)
//   - context: Service dependency injection
//   - prompt: Prompt template loading
//   - task: Task-based model selection
//   - http: HTTP client utilities
//   - testutil: Test utilities and fixtures
//
// # Quick Start
//
//	import (
//	    "github.com/randalmurphal/devflow/git"
//	    "github.com/randalmurphal/devflow/workflow"
//	    "github.com/randalmurphal/devflow/context"
//	)
//
//	// Create git context
//	gitCtx, _ := git.NewContext("/path/to/repo")
//
//	// Create workflow state
//	state := workflow.NewState("my-flow")
//
//	// Inject services
//	services := &context.Services{Git: gitCtx}
//	ctx := services.InjectAll(ctx)
//
// See individual package documentation for detailed usage.
package devflow

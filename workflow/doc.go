// Package workflow provides workflow state management and node implementations
// for AI-powered development workflows.
//
// Core types:
//   - State: Workflow execution state with git, spec, implementation, and review data
//   - NodeFunc: Function signature for workflow nodes
//   - NodeConfig: Configuration for node behavior (retries, transcripts, etc.)
//   - Ticket: External ticket reference (Jira, GitHub issue, etc.)
//
// Workflow nodes:
//   - CreateWorktreeNode: Creates git worktree for isolated work
//   - GenerateSpecNode: Generates feature specification from ticket
//   - ImplementNode: Implements code based on specification
//   - ReviewNode: Reviews implementation for issues
//   - FixFindingsNode: Fixes issues found during review
//   - RunTestsNode: Executes test suite
//   - CheckLintNode: Runs linting checks
//   - CreatePRNode: Creates pull request
//   - NotifyNode: Sends workflow notifications
//
// Example usage:
//
//	state := workflow.NewState("ticket-to-pr")
//	state.SetTicket(workflow.Ticket{ID: "TK-421", Title: "Add feature"})
//	result, err := workflow.CreateWorktreeNode(ctx, state)
package workflow

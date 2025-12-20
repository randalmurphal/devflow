// Package task provides task-based model selection for LLM operations.
//
// Core types:
//   - TaskType: Type of task (investigation, implementation, review, etc.)
//   - Selector: Selects appropriate model based on task type
//
// Task types:
//   - TaskInvestigation: Code exploration, impact analysis
//   - TaskImplementation: Writing code, making changes
//   - TaskReview: Code review, validation
//   - TaskArchitecture: Design decisions, high-stakes reasoning
//   - TaskSimple: Quick searches, formatting
//
// Example usage:
//
//	selector := task.NewSelector(task.Config{
//	    Investigation: "claude-opus-4-20250514",
//	    Implementation: "claude-sonnet-4-20250514",
//	    Review:         "claude-opus-4-20250514",
//	    Simple:         "claude-haiku-3-5-20241022",
//	})
//	model := selector.ModelFor(task.TaskReview)
package task

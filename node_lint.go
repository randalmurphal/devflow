package devflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
)

// DefaultLintCommand is the default command used to run linting.
const DefaultLintCommand = "go vet ./..."

// CheckLintNode runs linting and type checks.
//
// Prerequisites: state.Worktree must be set
// Updates: state.LintOutput, state.LintPassed, state.LintRunAt
//
// The node uses CommandRunner from context if available, otherwise falls back
// to ExecRunner. This allows for easy testing with MockRunner.
func CheckLintNode(ctx flowgraph.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireWorktree); err != nil {
		return state, err
	}

	// Get command runner (uses ExecRunner if none in context)
	runner := GetCommandRunner(ctx)

	// Run linter using the runner
	output, err := runner.Run(state.Worktree, "sh", "-c", DefaultLintCommand)
	passed := err == nil

	// Parse lint output
	lintOutput := parseLintOutput(output, passed)

	state.LintOutput = lintOutput
	state.LintPassed = passed
	state.LintRunAt = time.Now()

	// Save lint output artifact
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		artifacts.SaveLintOutput(state.RunID, lintOutput)
	}

	return state, nil
}

// parseLintOutput parses linter output
func parseLintOutput(output string, passed bool) *LintOutput {
	result := &LintOutput{
		Passed: passed,
		Tool:   "go vet",
	}

	if passed {
		return result
	}

	// Parse issues from output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Go vet format: file.go:line:col: message
		if parts := strings.SplitN(line, ":", 4); len(parts) >= 3 {
			result.Issues = append(result.Issues, LintIssue{
				File:     parts[0],
				Line:     parseIntSafe(parts[1]),
				Message:  strings.TrimSpace(strings.Join(parts[2:], ":")),
				Severity: "warning",
			})
			result.Summary.TotalIssues++
			result.Summary.Warnings++
		}
	}

	return result
}

// parseIntSafe safely parses int, returns 0 on error
func parseIntSafe(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

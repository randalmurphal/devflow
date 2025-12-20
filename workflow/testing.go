package workflow

import (
	"strings"
	"time"

	"github.com/randalmurphal/devflow/artifact"
	devcontext "github.com/randalmurphal/devflow/context"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
)

// DefaultTestCommand is the default command used to run tests.
const DefaultTestCommand = "go test -race ./..."

// RunTestsNode runs the test suite.
//
// Prerequisites: state.Worktree must be set
// Updates: state.TestOutput, state.TestPassed, state.TestRunAt
//
// The node uses CommandRunner from context if available, otherwise falls back
// to ExecRunner. This allows for easy testing with MockRunner.
func RunTestsNode(ctx flowgraph.Context, state State) (State, error) {
	if err := state.Validate(RequireWorktree); err != nil {
		return state, err
	}

	// Get command runner from context
	runner := getCommandRunner(ctx)

	// Run tests using the runner
	output, err := runner.Run(state.Worktree, "sh", "-c", DefaultTestCommand)
	passed := err == nil

	// Parse test output
	testOutput := parseTestOutput(output, passed)

	state.TestOutput = testOutput
	state.TestPassed = passed
	state.TestRunAt = time.Now()

	// Save test output artifact
	if artifacts := devcontext.Artifact(ctx); artifacts != nil {
		artifacts.SaveTestOutput(state.RunID, testOutput)
	}

	// Don't return error for test failures - let the graph handle routing
	return state, nil
}

// parseTestOutput parses test command output
func parseTestOutput(output string, passed bool) *artifact.TestOutput {
	result := &artifact.TestOutput{
		Passed: passed,
	}

	// Count test results from output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ok ") {
			result.PassedTests++
			result.TotalTests++
		} else if strings.HasPrefix(line, "FAIL") {
			result.FailedTests++
			result.TotalTests++
		} else if strings.HasPrefix(line, "--- FAIL:") {
			// Extract failure details
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				result.Failures = append(result.Failures, artifact.TestFailure{
					Name:   parts[2],
					Output: output, // Full output for context
				})
			}
		}
	}

	return result
}

// getCommandRunner returns the CommandRunner from context, or a default runner.
func getCommandRunner(ctx flowgraph.Context) commandRunner {
	// GetRunner handles nil check and fallback to ExecRunner
	return devcontext.GetRunner(ctx)
}

// commandRunner is a minimal interface for running commands
type commandRunner interface {
	Run(workDir string, name string, args ...string) (string, error)
}

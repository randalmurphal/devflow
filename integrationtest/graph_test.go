package integrationtest

import (
	"testing"

	"github.com/randalmurphal/devflow/workflow"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraphConstruction verifies that devflow nodes can be used to build a flowgraph.
func TestGraphConstruction(t *testing.T) {
	// Build a simple linear graph with devflow nodes
	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("create-worktree", workflow.CreateWorktreeNode).
		AddNode("cleanup", workflow.CleanupNode).
		AddEdge("create-worktree", "cleanup").
		AddEdge("cleanup", flowgraph.END).
		SetEntry("create-worktree")

	// Verify the graph compiles
	compiled, err := graph.Compile()
	require.NoError(t, err, "graph should compile")
	assert.NotNil(t, compiled, "compiled graph should not be nil")
}

// TestGraphWithAllNodes verifies that all devflow nodes compile together.
func TestGraphWithAllNodes(t *testing.T) {
	// Build a comprehensive graph with all node types
	graph := flowgraph.NewGraph[workflow.State]().
		// Worktree management
		AddNode("create-worktree", workflow.CreateWorktreeNode).
		// Spec generation
		AddNode("generate-spec", workflow.GenerateSpecNode).
		// Implementation
		AddNode("implement", workflow.ImplementNode).
		// Quality checks
		AddNode("lint", workflow.CheckLintNode).
		AddNode("test", workflow.RunTestsNode).
		// Review
		AddNode("review", workflow.ReviewNode).
		AddNode("fix-findings", workflow.FixFindingsNode).
		// PR creation
		AddNode("create-pr", workflow.CreatePRNode).
		// Notification
		AddNode("notify", workflow.NotifyNode).
		// Cleanup
		AddNode("cleanup", workflow.CleanupNode).
		// Define edges
		AddEdge("create-worktree", "generate-spec").
		AddEdge("generate-spec", "implement").
		AddEdge("implement", "lint").
		AddEdge("lint", "test").
		AddEdge("test", "review").
		AddEdge("review", "fix-findings").
		AddEdge("fix-findings", "create-pr").
		AddEdge("create-pr", "notify").
		AddEdge("notify", "cleanup").
		AddEdge("cleanup", flowgraph.END).
		SetEntry("create-worktree")

	compiled, err := graph.Compile()
	require.NoError(t, err, "comprehensive graph should compile")
	assert.NotNil(t, compiled)
}

// TestNodeWrappers verifies that wrapped nodes compile correctly.
// Note: workflow.NodeFunc needs to be converted to flowgraph.NodeFunc[State]
func TestNodeWrappers(t *testing.T) {
	// Create wrapped nodes and convert to flowgraph type
	specWithRetry := flowgraph.NodeFunc[workflow.State](
		workflow.WithRetry(workflow.GenerateSpecNode, 3),
	)
	specWithTiming := flowgraph.NodeFunc[workflow.State](
		workflow.WithTiming(workflow.GenerateSpecNode),
	)
	specWithTranscript := flowgraph.NodeFunc[workflow.State](
		workflow.WithTranscript(workflow.GenerateSpecNode, "generate-spec"),
	)

	// Use in a graph
	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("spec-retry", specWithRetry).
		AddNode("spec-timing", specWithTiming).
		AddNode("spec-transcript", specWithTranscript).
		AddEdge("spec-retry", "spec-timing").
		AddEdge("spec-timing", "spec-transcript").
		AddEdge("spec-transcript", flowgraph.END).
		SetEntry("spec-retry")

	compiled, err := graph.Compile()
	require.NoError(t, err, "wrapped nodes should compile")
	assert.NotNil(t, compiled)
}

// TestDevStatePassthrough verifies that State passes through nodes correctly.
func TestDevStatePassthrough(t *testing.T) {
	repoPath := setupTempRepo(t)

	// Create a simple node that just passes state through
	passthrough := func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
		// Modify state to prove it passes through
		state.TicketID = "TK-PASSTHROUGH"
		return state, nil
	}

	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("passthrough", passthrough).
		AddEdge("passthrough", flowgraph.END).
		SetEntry("passthrough")

	compiled, err := graph.Compile()
	require.NoError(t, err)

	// Setup context
	ctx := setupContext(t, repoPath, nil)

	// Execute
	state := workflow.NewState("test-flow")
	result, err := compiled.Run(ctx, state)
	require.NoError(t, err)

	assert.Equal(t, "TK-PASSTHROUGH", result.TicketID, "state should be modified by passthrough")
	assert.Equal(t, "test-flow", result.FlowID, "original FlowID should be preserved")
}

// TestMultiNodeExecution verifies state flows through multiple nodes.
func TestMultiNodeExecution(t *testing.T) {
	repoPath := setupTempRepo(t)

	// Create nodes that track execution order
	order := []string{}

	nodeA := func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
		order = append(order, "A")
		state.TicketID = "FROM_A"
		return state, nil
	}

	nodeB := func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
		order = append(order, "B")
		// Verify state from A
		if state.TicketID != "FROM_A" {
			t.Error("nodeB should see state from nodeA")
		}
		state.Branch = "FROM_B"
		return state, nil
	}

	nodeC := func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
		order = append(order, "C")
		// Verify state from B
		if state.Branch != "FROM_B" {
			t.Error("nodeC should see state from nodeB")
		}
		state.Spec = "FROM_C"
		return state, nil
	}

	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("a", nodeA).
		AddNode("b", nodeB).
		AddNode("c", nodeC).
		AddEdge("a", "b").
		AddEdge("b", "c").
		AddEdge("c", flowgraph.END).
		SetEntry("a")

	compiled, err := graph.Compile()
	require.NoError(t, err)

	ctx := setupContext(t, repoPath, nil)
	state := workflow.NewState("test")

	result, err := compiled.Run(ctx, state)
	require.NoError(t, err)

	// Verify execution order
	assert.Equal(t, []string{"A", "B", "C"}, order, "nodes should execute in order")

	// Verify final state
	assert.Equal(t, "FROM_A", result.TicketID)
	assert.Equal(t, "FROM_B", result.Branch)
	assert.Equal(t, "FROM_C", result.Spec)
}

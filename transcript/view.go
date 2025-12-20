package transcript

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Viewer displays transcripts
type Viewer struct {
	colorEnabled bool
}

// NewViewer creates a viewer
func NewViewer(colorEnabled bool) *Viewer {
	return &Viewer{colorEnabled: colorEnabled}
}

// ViewFull displays the complete transcript
func (v *Viewer) ViewFull(w io.Writer, t *Transcript) error {
	v.writeHeader(w, t)

	for _, turn := range t.Turns {
		v.writeTurn(w, turn)
	}

	return nil
}

// ViewSummary displays a brief summary
func (v *Viewer) ViewSummary(w io.Writer, t *Transcript) error {
	v.writeHeader(w, t)

	fmt.Fprintln(w, "\nTurn Summary:")
	for _, turn := range t.Turns {
		preview := turn.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " ")
		fmt.Fprintf(w, "  [%d] %s: %s\n", turn.ID, turn.Role, preview)
	}

	return nil
}

// ViewTurn displays a single turn
func (v *Viewer) ViewTurn(w io.Writer, turn Turn) error {
	v.writeTurn(w, turn)
	return nil
}

// ViewAssistantOnly displays only assistant turns
func (v *Viewer) ViewAssistantOnly(w io.Writer, t *Transcript) error {
	v.writeHeader(w, t)

	for _, turn := range t.Turns {
		if turn.Role == "assistant" {
			v.writeTurn(w, turn)
		}
	}

	return nil
}

func (v *Viewer) writeHeader(w io.Writer, t *Transcript) {
	sep := strings.Repeat("=", 60)

	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "Run: %s\n", t.RunID)
	fmt.Fprintf(w, "Flow: %s | Status: %s\n", t.Metadata.FlowID, t.Metadata.Status)

	duration := t.Duration()
	fmt.Fprintf(w, "Started: %s | Duration: %s\n",
		t.Metadata.StartedAt.Format("2006-01-02 15:04:05"),
		duration.Round(time.Second))

	fmt.Fprintf(w, "Tokens: %d in / %d out | Cost: $%.2f\n",
		t.Metadata.TotalTokensIn,
		t.Metadata.TotalTokensOut,
		t.Metadata.TotalCost)

	if t.Metadata.Error != "" {
		fmt.Fprintf(w, "Error: %s\n", t.Metadata.Error)
	}

	fmt.Fprintln(w, sep)
}

func (v *Viewer) writeTurn(w io.Writer, turn Turn) {
	fmt.Fprintln(w)

	// Turn header
	header := fmt.Sprintf("[%d] %s (%s)",
		turn.ID,
		strings.ToUpper(turn.Role),
		turn.Timestamp.Format("15:04:05"))

	if turn.TokensIn > 0 {
		header += fmt.Sprintf(" [%d tokens in]", turn.TokensIn)
	}
	if turn.TokensOut > 0 {
		header += fmt.Sprintf(" [%d tokens out]", turn.TokensOut)
	}
	if turn.DurationMs > 0 {
		header += fmt.Sprintf(" [%dms]", turn.DurationMs)
	}

	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("-", 60))

	// Content
	fmt.Fprintln(w, turn.Content)

	// Tool calls
	for _, tc := range turn.ToolCalls {
		fmt.Fprintf(w, "\n  Tool: %s\n", tc.Name)
		if tc.Input != nil {
			inputJSON, _ := json.MarshalIndent(tc.Input, "     ", "  ")
			fmt.Fprintf(w, "     Input: %s\n", string(inputJSON))
		}
		if tc.Output != "" {
			output := tc.Output
			if len(output) > 200 {
				output = output[:200] + "..."
			}
			fmt.Fprintf(w, "     Output: %s\n", output)
		}
		if tc.Error != "" {
			fmt.Fprintf(w, "     Error: %s\n", tc.Error)
		}
	}
}

// ExportMarkdown exports to markdown format
func (v *Viewer) ExportMarkdown(w io.Writer, t *Transcript) error {
	fmt.Fprintf(w, "# Transcript: %s\n\n", t.RunID)

	// Metadata
	fmt.Fprintf(w, "## Metadata\n\n")
	fmt.Fprintf(w, "| Field | Value |\n")
	fmt.Fprintf(w, "|-------|-------|\n")
	fmt.Fprintf(w, "| Flow | %s |\n", t.Metadata.FlowID)
	fmt.Fprintf(w, "| Status | %s |\n", t.Metadata.Status)
	fmt.Fprintf(w, "| Started | %s |\n", t.Metadata.StartedAt.Format(time.RFC3339))
	if !t.Metadata.EndedAt.IsZero() {
		fmt.Fprintf(w, "| Ended | %s |\n", t.Metadata.EndedAt.Format(time.RFC3339))
	}
	fmt.Fprintf(w, "| Duration | %s |\n", t.Duration().Round(time.Second))
	fmt.Fprintf(w, "| Tokens In | %d |\n", t.Metadata.TotalTokensIn)
	fmt.Fprintf(w, "| Tokens Out | %d |\n", t.Metadata.TotalTokensOut)
	fmt.Fprintf(w, "| Cost | $%.2f |\n", t.Metadata.TotalCost)
	if t.Metadata.Error != "" {
		fmt.Fprintf(w, "| Error | %s |\n", t.Metadata.Error)
	}
	fmt.Fprintln(w)

	// Conversation
	fmt.Fprintf(w, "## Conversation\n\n")

	for _, turn := range t.Turns {
		fmt.Fprintf(w, "### %s (Turn %d)\n\n", title(turn.Role), turn.ID)

		if turn.TokensIn > 0 {
			fmt.Fprintf(w, "*%d tokens in*\n\n", turn.TokensIn)
		}
		if turn.TokensOut > 0 {
			fmt.Fprintf(w, "*%d tokens out*\n\n", turn.TokensOut)
		}

		fmt.Fprintf(w, "%s\n\n", turn.Content)

		for _, tc := range turn.ToolCalls {
			fmt.Fprintf(w, "#### Tool Call: `%s`\n\n", tc.Name)
			if tc.Input != nil {
				inputJSON, _ := json.MarshalIndent(tc.Input, "", "  ")
				fmt.Fprintf(w, "**Input:**\n```json\n%s\n```\n\n", string(inputJSON))
			}
			if tc.Output != "" {
				fmt.Fprintf(w, "**Output:**\n```\n%s\n```\n\n", tc.Output)
			}
			if tc.Error != "" {
				fmt.Fprintf(w, "**Error:** %s\n\n", tc.Error)
			}
		}
	}

	return nil
}

// ExportJSON exports to JSON format
func (v *Viewer) ExportJSON(w io.Writer, t *Transcript) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(t)
}

// Diff compares two transcripts
func (v *Viewer) Diff(w io.Writer, a, b *Transcript) error {
	fmt.Fprintln(w, "Comparing transcripts:")
	fmt.Fprintf(w, "  A: %s (%s)\n", a.RunID, a.Metadata.Status)
	fmt.Fprintf(w, "  B: %s (%s)\n", b.RunID, b.Metadata.Status)
	fmt.Fprintln(w)

	// Compare metadata
	fmt.Fprintln(w, "Metadata Comparison:")
	fmt.Fprintf(w, "  Turns:      %d vs %d\n", len(a.Turns), len(b.Turns))
	fmt.Fprintf(w, "  Tokens In:  %d vs %d\n", a.Metadata.TotalTokensIn, b.Metadata.TotalTokensIn)
	fmt.Fprintf(w, "  Tokens Out: %d vs %d\n", a.Metadata.TotalTokensOut, b.Metadata.TotalTokensOut)
	fmt.Fprintf(w, "  Cost:       $%.2f vs $%.2f\n", a.Metadata.TotalCost, b.Metadata.TotalCost)
	fmt.Fprintf(w, "  Duration:   %s vs %s\n", a.Duration().Round(time.Second), b.Duration().Round(time.Second))
	fmt.Fprintln(w)

	// Compare assistant outputs
	fmt.Fprintln(w, "Turn Comparison:")

	maxTurns := len(a.Turns)
	if len(b.Turns) > maxTurns {
		maxTurns = len(b.Turns)
	}

	for i := 0; i < maxTurns; i++ {
		var turnA, turnB *Turn
		if i < len(a.Turns) {
			turnA = &a.Turns[i]
		}
		if i < len(b.Turns) {
			turnB = &b.Turns[i]
		}

		if turnA == nil {
			fmt.Fprintf(w, "  Turn %d: [missing] vs %s (%d chars)\n", i+1, turnB.Role, len(turnB.Content))
		} else if turnB == nil {
			fmt.Fprintf(w, "  Turn %d: %s (%d chars) vs [missing]\n", i+1, turnA.Role, len(turnA.Content))
		} else if turnA.Role != turnB.Role {
			fmt.Fprintf(w, "  Turn %d: %s vs %s (different roles)\n", i+1, turnA.Role, turnB.Role)
		} else if turnA.Content == turnB.Content {
			fmt.Fprintf(w, "  Turn %d: %s - identical (%d chars)\n", i+1, turnA.Role, len(turnA.Content))
		} else {
			fmt.Fprintf(w, "  Turn %d: %s - different (%d vs %d chars)\n", i+1, turnA.Role, len(turnA.Content), len(turnB.Content))
		}
	}

	return nil
}

// FormatMetaList formats a list of metadata for display
func (v *Viewer) FormatMetaList(w io.Writer, metas []Meta) error {
	if len(metas) == 0 {
		fmt.Fprintln(w, "No runs found.")
		return nil
	}

	// Header
	fmt.Fprintf(w, "%-40s %-12s %-20s %8s %8s %8s\n",
		"RUN ID", "STATUS", "STARTED", "TOKENS", "COST", "TURNS")
	fmt.Fprintln(w, strings.Repeat("-", 100))

	for _, m := range metas {
		tokens := fmt.Sprintf("%d/%d", m.TotalTokensIn, m.TotalTokensOut)
		cost := fmt.Sprintf("$%.2f", m.TotalCost)

		fmt.Fprintf(w, "%-40s %-12s %-20s %8s %8s %8d\n",
			truncate(m.RunID, 40),
			m.Status,
			m.StartedAt.Format("2006-01-02 15:04"),
			tokens,
			cost,
			m.TurnCount)
	}

	fmt.Fprintf(w, "\nTotal: %d runs\n", len(metas))
	return nil
}

// FormatStats formats statistics for display
func (v *Viewer) FormatStats(w io.Writer, stats *Statistics) error {
	fmt.Fprintln(w, "Run Statistics:")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	fmt.Fprintf(w, "Total Runs:      %d\n", stats.TotalRuns)
	fmt.Fprintf(w, "  Completed:     %d\n", stats.CompletedRuns)
	fmt.Fprintf(w, "  Failed:        %d\n", stats.FailedRuns)
	fmt.Fprintf(w, "  Canceled:      %d\n", stats.CanceledRuns)
	fmt.Fprintf(w, "  Active:        %d\n", stats.ActiveRuns)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Total Tokens:    %d in / %d out\n", stats.TotalTokensIn, stats.TotalTokensOut)
	fmt.Fprintf(w, "Avg Tokens/Run:  %d in / %d out\n", stats.AvgTokensIn, stats.AvgTokensOut)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Total Cost:      $%.2f\n", stats.TotalCost)
	fmt.Fprintf(w, "Avg Cost/Run:    $%.2f\n", stats.AvgCost)

	return nil
}

// title capitalizes the first letter
func title(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// truncate shortens a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

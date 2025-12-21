package transcript

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Searcher provides search capabilities over transcripts
type Searcher struct {
	baseDir string
}

// NewSearcher creates a searcher
func NewSearcher(baseDir string) *Searcher {
	return &Searcher{baseDir: baseDir}
}

// SearchOptions configures content search
type SearchOptions struct {
	CaseSensitive bool
	MaxResults    int
	Context       int // Lines of context around match
}

// SearchResult represents a search match
type SearchResult struct {
	RunID     string `json:"runId"`
	TurnID    int    `json:"turnId,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content"`
	MatchLine int    `json:"matchLine,omitempty"`
	Match     string `json:"match,omitempty"`
}

// SearchContent searches transcript content using ripgrep or grep
func (s *Searcher) SearchContent(query string, opts SearchOptions) ([]SearchResult, error) {
	runsDir := filepath.Join(s.baseDir, "runs")

	// Try ripgrep first
	if _, err := exec.LookPath("rg"); err == nil {
		return s.searchWithRipgrep(runsDir, query, opts)
	}

	// Fall back to grep
	return s.searchWithGrep(runsDir, query, opts)
}

func (s *Searcher) searchWithRipgrep(runsDir, query string, opts SearchOptions) ([]SearchResult, error) {
	args := []string{
		"--json",
		"-g", "transcript.json",
		"-g", "transcript.json.gz",
	}

	if !opts.CaseSensitive {
		args = append(args, "-i")
	}

	if opts.MaxResults > 0 {
		args = append(args, "-m", strconv.Itoa(opts.MaxResults))
	}

	args = append(args, query, runsDir)

	cmd := exec.Command("rg", args...)
	output, err := cmd.Output()
	if err != nil {
		// rg returns exit code 1 for no matches
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}

	return s.parseRipgrepOutput(output)
}

func (s *Searcher) parseRipgrepOutput(output []byte) ([]SearchResult, error) {
	var results []SearchResult

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		var msg struct {
			Type string `json:"type"`
			Data struct {
				Path struct {
					Text string `json:"text"`
				} `json:"path"`
				Lines struct {
					Text string `json:"text"`
				} `json:"lines"`
				LineNumber int `json:"line_number"`
			} `json:"data"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			slog.Debug("skipping malformed ripgrep output",
				slog.String("error", err.Error()))
			continue
		}

		if msg.Type != "match" {
			continue
		}

		runID := extractRunID(msg.Data.Path.Text)
		if runID == "" {
			continue
		}

		results = append(results, SearchResult{
			RunID:     runID,
			Content:   strings.TrimSpace(msg.Data.Lines.Text),
			MatchLine: msg.Data.LineNumber,
		})
	}

	return results, nil
}

func (s *Searcher) searchWithGrep(runsDir, query string, opts SearchOptions) ([]SearchResult, error) {
	args := []string{"-r", "-l"}
	if !opts.CaseSensitive {
		args = append(args, "-i")
	}
	args = append(args, query, runsDir)

	cmd := exec.Command("grep", args...)
	output, err := cmd.Output()
	if err != nil {
		// grep returns 1 for no matches
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}

	var results []SearchResult
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		runID := extractRunID(line)
		if runID == "" || seen[runID] {
			continue
		}
		seen[runID] = true

		results = append(results, SearchResult{
			RunID: runID,
		})

		if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
			break
		}
	}

	return results, nil
}

// FindByFlow returns transcripts for a flow
func (s *Searcher) FindByFlow(flowID string) ([]Meta, error) {
	return s.findByMetadata(func(m *Meta) bool {
		return m.FlowID == flowID
	})
}

// FindByStatus returns transcripts with status
func (s *Searcher) FindByStatus(status RunStatus) ([]Meta, error) {
	return s.findByMetadata(func(m *Meta) bool {
		return m.Status == status
	})
}

// FindByDateRange returns transcripts in date range
func (s *Searcher) FindByDateRange(start, end time.Time) ([]Meta, error) {
	return s.findByMetadata(func(m *Meta) bool {
		return m.StartedAt.After(start) && m.StartedAt.Before(end)
	})
}

// FindByTokenRange returns transcripts within token ranges
func (s *Searcher) FindByTokenRange(minIn, maxIn, minOut, maxOut int) ([]Meta, error) {
	return s.findByMetadata(func(m *Meta) bool {
		if minIn > 0 && m.TotalTokensIn < minIn {
			return false
		}
		if maxIn > 0 && m.TotalTokensIn > maxIn {
			return false
		}
		if minOut > 0 && m.TotalTokensOut < minOut {
			return false
		}
		if maxOut > 0 && m.TotalTokensOut > maxOut {
			return false
		}
		return true
	})
}

func (s *Searcher) findByMetadata(predicate func(*Meta) bool) ([]Meta, error) {
	runsDir := filepath.Join(s.baseDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []Meta

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaPath := filepath.Join(runsDir, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			slog.Debug("skipping transcript with unreadable metadata",
				slog.String("path", metaPath),
				slog.String("error", err.Error()))
			continue
		}

		var meta Meta
		if err := json.Unmarshal(data, &meta); err != nil {
			slog.Debug("skipping transcript with malformed metadata",
				slog.String("path", metaPath),
				slog.String("error", err.Error()))
			continue
		}

		if predicate(&meta) {
			results = append(results, meta)
		}
	}

	return results, nil
}

// TotalCost calculates total cost for matching runs
func (s *Searcher) TotalCost(filter ListFilter) (float64, error) {
	store, err := NewFileStore(StoreConfig{BaseDir: s.baseDir})
	if err != nil {
		return 0, err
	}

	runs, err := store.List(filter)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, run := range runs {
		total += run.TotalCost
	}

	return total, nil
}

// TotalTokens calculates total tokens for matching runs
func (s *Searcher) TotalTokens(filter ListFilter) (int, int, error) {
	store, err := NewFileStore(StoreConfig{BaseDir: s.baseDir})
	if err != nil {
		return 0, 0, err
	}

	runs, err := store.List(filter)
	if err != nil {
		return 0, 0, err
	}

	var totalIn, totalOut int
	for _, run := range runs {
		totalIn += run.TotalTokensIn
		totalOut += run.TotalTokensOut
	}

	return totalIn, totalOut, nil
}

// RunStats returns statistics for matching runs
func (s *Searcher) RunStats(filter ListFilter) (*Statistics, error) {
	store, err := NewFileStore(StoreConfig{BaseDir: s.baseDir})
	if err != nil {
		return nil, err
	}

	runs, err := store.List(filter)
	if err != nil {
		return nil, err
	}

	stats := &Statistics{}
	for _, run := range runs {
		stats.TotalRuns++
		stats.TotalTokensIn += run.TotalTokensIn
		stats.TotalTokensOut += run.TotalTokensOut
		stats.TotalCost += run.TotalCost

		switch run.Status {
		case RunStatusCompleted:
			stats.CompletedRuns++
		case RunStatusFailed:
			stats.FailedRuns++
		case RunStatusCanceled:
			stats.CanceledRuns++
		case RunStatusRunning:
			stats.ActiveRuns++
		}
	}

	if stats.TotalRuns > 0 {
		stats.AvgTokensIn = stats.TotalTokensIn / stats.TotalRuns
		stats.AvgTokensOut = stats.TotalTokensOut / stats.TotalRuns
		stats.AvgCost = stats.TotalCost / float64(stats.TotalRuns)
	}

	return stats, nil
}

// Statistics holds aggregated run statistics
type Statistics struct {
	TotalRuns      int
	CompletedRuns  int
	FailedRuns     int
	CanceledRuns   int
	ActiveRuns     int
	TotalTokensIn  int
	TotalTokensOut int
	TotalCost      float64
	AvgTokensIn    int
	AvgTokensOut   int
	AvgCost        float64
}

func extractRunID(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	for i, p := range parts {
		if p == "runs" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

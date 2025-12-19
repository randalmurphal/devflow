package devflow

import (
	"encoding/json"
)

// ReviewResult represents the result of a code review
type ReviewResult struct {
	Approved bool            `json:"approved"`
	Verdict  string          `json:"verdict,omitempty"` // APPROVE, REQUEST_CHANGES, NEEDS_DISCUSSION
	Summary  string          `json:"summary"`
	Findings []ReviewFinding `json:"findings,omitempty"`
	Metrics  ReviewMetrics   `json:"metrics,omitempty"`
}

// ReviewFinding represents a single review finding
type ReviewFinding struct {
	File       string `json:"file"`
	Line       int    `json:"line,omitempty"`
	EndLine    int    `json:"endLine,omitempty"`
	Severity   string `json:"severity"` // critical, error, warning, info
	Category   string `json:"category"` // security, performance, style, logic, test
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Code       string `json:"code,omitempty"` // Code snippet
}

// ReviewMetrics contains metrics about the review
type ReviewMetrics struct {
	LinesReviewed int     `json:"linesReviewed"`
	FilesReviewed int     `json:"filesReviewed"`
	TokensUsed    int     `json:"tokensUsed"`
	Duration      float64 `json:"durationSeconds,omitempty"`
}

// FindingSeverity constants
const (
	SeverityCritical = "critical"
	SeverityError    = "error"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// FindingCategory constants
const (
	CategorySecurity    = "security"
	CategoryPerformance = "performance"
	CategoryStyle       = "style"
	CategoryLogic       = "logic"
	CategoryTest        = "test"
)

// ReviewVerdict constants
const (
	VerdictApprove        = "APPROVE"
	VerdictRequestChanges = "REQUEST_CHANGES"
	VerdictNeedsDiscussion = "NEEDS_DISCUSSION"
)

// HasCriticalFindings returns true if any finding is critical
func (r *ReviewResult) HasCriticalFindings() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// HasErrors returns true if any finding is an error or higher
func (r *ReviewResult) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical || f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// FindingsByFile groups findings by file
func (r *ReviewResult) FindingsByFile() map[string][]ReviewFinding {
	result := make(map[string][]ReviewFinding)
	for _, f := range r.Findings {
		result[f.File] = append(result[f.File], f)
	}
	return result
}

// FindingsBySeverity groups findings by severity
func (r *ReviewResult) FindingsBySeverity() map[string][]ReviewFinding {
	result := make(map[string][]ReviewFinding)
	for _, f := range r.Findings {
		result[f.Severity] = append(result[f.Severity], f)
	}
	return result
}

// TestOutput represents test execution results
type TestOutput struct {
	Passed       bool          `json:"passed"`
	TotalTests   int           `json:"totalTests"`
	PassedTests  int           `json:"passedTests"`
	FailedTests  int           `json:"failedTests"`
	SkippedTests int           `json:"skippedTests"`
	Duration     string        `json:"duration"`
	Failures     []TestFailure `json:"failures,omitempty"`
	Coverage     *TestCoverage `json:"coverage,omitempty"`
}

// TestFailure represents a single test failure
type TestFailure struct {
	Name      string `json:"name"`
	Package   string `json:"package,omitempty"`
	Message   string `json:"message"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Output    string `json:"output,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
}

// TestCoverage represents code coverage data
type TestCoverage struct {
	Percentage float64              `json:"percentage"`
	Lines      int                  `json:"lines"`
	Covered    int                  `json:"covered"`
	ByPackage  map[string]float64   `json:"byPackage,omitempty"`
}

// SuccessRate returns the percentage of tests that passed
func (t *TestOutput) SuccessRate() float64 {
	if t.TotalTests == 0 {
		return 0
	}
	return float64(t.PassedTests) / float64(t.TotalTests) * 100
}

// LintOutput represents linting results
type LintOutput struct {
	Passed   bool          `json:"passed"`
	Tool     string        `json:"tool"` // ruff, eslint, golint, etc.
	Issues   []LintIssue   `json:"issues,omitempty"`
	Summary  LintSummary   `json:"summary"`
}

// LintIssue represents a single lint issue
type LintIssue struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column,omitempty"`
	Rule       string `json:"rule"`
	Severity   string `json:"severity"` // error, warning
	Message    string `json:"message"`
	Fixable    bool   `json:"fixable,omitempty"`
}

// LintSummary contains summary statistics
type LintSummary struct {
	TotalIssues   int `json:"totalIssues"`
	Errors        int `json:"errors"`
	Warnings      int `json:"warnings"`
	FixableCount  int `json:"fixableCount"`
	FilesChecked  int `json:"filesChecked"`
}

// Specification represents a generated specification
type Specification struct {
	Title        string            `json:"title"`
	Overview     string            `json:"overview"`
	Requirements []string          `json:"requirements,omitempty"`
	Design       string            `json:"design,omitempty"`
	APIChanges   string            `json:"apiChanges,omitempty"`
	DBChanges    string            `json:"dbChanges,omitempty"`
	TestPlan     string            `json:"testPlan,omitempty"`
	Risks        []string          `json:"risks,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Raw          string            `json:"raw,omitempty"` // Original markdown
}

// SaveSpec saves a specification artifact
func (m *ArtifactManager) SaveSpec(runID string, spec string) error {
	return m.SaveArtifact(runID, ArtifactSpec, []byte(spec))
}

// LoadSpec loads a specification artifact
func (m *ArtifactManager) LoadSpec(runID string) (string, error) {
	data, err := m.LoadArtifact(runID, ArtifactSpec)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveReview saves a review result artifact
func (m *ArtifactManager) SaveReview(runID string, review *ReviewResult) error {
	data, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return err
	}
	return m.SaveArtifact(runID, ArtifactReview, data)
}

// LoadReview loads a review result artifact
func (m *ArtifactManager) LoadReview(runID string) (*ReviewResult, error) {
	data, err := m.LoadArtifact(runID, ArtifactReview)
	if err != nil {
		return nil, err
	}

	var review ReviewResult
	if err := json.Unmarshal(data, &review); err != nil {
		return nil, err
	}

	return &review, nil
}

// SaveTestOutput saves test output artifact
func (m *ArtifactManager) SaveTestOutput(runID string, output *TestOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	return m.SaveArtifact(runID, ArtifactTestOutput, data)
}

// LoadTestOutput loads test output artifact
func (m *ArtifactManager) LoadTestOutput(runID string) (*TestOutput, error) {
	data, err := m.LoadArtifact(runID, ArtifactTestOutput)
	if err != nil {
		return nil, err
	}

	var output TestOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

// SaveLintOutput saves lint output artifact
func (m *ArtifactManager) SaveLintOutput(runID string, output *LintOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	return m.SaveArtifact(runID, ArtifactLintOutput, data)
}

// LoadLintOutput loads lint output artifact
func (m *ArtifactManager) LoadLintOutput(runID string) (*LintOutput, error) {
	data, err := m.LoadArtifact(runID, ArtifactLintOutput)
	if err != nil {
		return nil, err
	}

	var output LintOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

// SaveDiff saves an implementation diff artifact
func (m *ArtifactManager) SaveDiff(runID string, diff string) error {
	return m.SaveArtifact(runID, ArtifactImplementation, []byte(diff))
}

// LoadDiff loads an implementation diff artifact
func (m *ArtifactManager) LoadDiff(runID string) (string, error) {
	data, err := m.LoadArtifact(runID, ArtifactImplementation)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveJSON saves arbitrary JSON data as an artifact
func (m *ArtifactManager) SaveJSON(runID, name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return m.SaveArtifact(runID, name, data)
}

// LoadJSON loads and unmarshals a JSON artifact
func (m *ArtifactManager) LoadJSON(runID, name string, v any) error {
	data, err := m.LoadArtifact(runID, name)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

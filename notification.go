package devflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rmurphy/flowgraph/pkg/flowgraph"
)

// =============================================================================
// Notification Types
// =============================================================================

// EventType represents the type of workflow event.
type EventType string

// Event type constants.
const (
	EventRunStarted    EventType = "run_started"
	EventRunCompleted  EventType = "run_completed"
	EventRunFailed     EventType = "run_failed"
	EventNodeStarted   EventType = "node_started"
	EventNodeCompleted EventType = "node_completed"
	EventNodeFailed    EventType = "node_failed"
	EventReviewNeeded  EventType = "review_needed"
	EventPRCreated     EventType = "pr_created"
)

// NotificationSeverity is an alias for notification severity levels.
// Uses the same constants as artifact_types.go: SeverityInfo, SeverityWarning, SeverityError

// NotificationEvent describes a workflow event for notification.
type NotificationEvent struct {
	Type      EventType      `json:"type"`
	RunID     string         `json:"run_id"`
	FlowID    string         `json:"flow_id"`
	NodeID    string         `json:"node_id,omitempty"`
	Message   string         `json:"message"`
	Severity  string         `json:"severity"` // SeverityInfo, SeverityWarning, SeverityError
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// =============================================================================
// Notifier Interface
// =============================================================================

// Notifier sends notifications about workflow events.
type Notifier interface {
	// Notify sends a notification. Implementations should be non-blocking
	// and handle errors gracefully (log, don't crash).
	Notify(ctx context.Context, event NotificationEvent) error
}

// =============================================================================
// LogNotifier
// =============================================================================

// LogNotifier logs notifications using slog (for testing/debugging).
type LogNotifier struct {
	Logger *slog.Logger
}

// NewLogNotifier creates a notifier that logs to the given logger.
// If logger is nil, uses the default slog logger.
func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogNotifier{Logger: logger}
}

// Notify implements Notifier.
func (n *LogNotifier) Notify(ctx context.Context, event NotificationEvent) error {
	level := slog.LevelInfo
	switch event.Severity {
	case SeverityWarning:
		level = slog.LevelWarn
	case SeverityError:
		level = slog.LevelError
	}

	n.Logger.Log(ctx, level, event.Message,
		"type", event.Type,
		"run_id", event.RunID,
		"flow_id", event.FlowID,
		"node_id", event.NodeID,
		"metadata", event.Metadata,
	)
	return nil
}

// =============================================================================
// WebhookNotifier
// =============================================================================

// WebhookNotifier sends notifications to a generic HTTP webhook.
type WebhookNotifier struct {
	URL     string
	Headers map[string]string
	Client  *http.Client
}

// NewWebhookNotifier creates a webhook notifier.
func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier {
	return &WebhookNotifier{
		URL:     url,
		Headers: headers,
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Notify implements Notifier.
func (n *WebhookNotifier) Notify(ctx context.Context, event NotificationEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}

	resp, err := n.Client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}

	return nil
}

// =============================================================================
// SlackNotifier
// =============================================================================

// SlackNotifier sends notifications to a Slack webhook.
type SlackNotifier struct {
	WebhookURL string
	Channel    string
	Username   string
	Client     *http.Client
}

// NewSlackNotifier creates a Slack webhook notifier.
func NewSlackNotifier(webhookURL string, opts ...SlackOption) *SlackNotifier {
	n := &SlackNotifier{
		WebhookURL: webhookURL,
		Username:   "devflow",
		Client:     &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// SlackOption configures SlackNotifier.
type SlackOption func(*SlackNotifier)

// WithSlackChannel sets the channel to post to.
func WithSlackChannel(channel string) SlackOption {
	return func(n *SlackNotifier) { n.Channel = channel }
}

// WithSlackUsername sets the bot username.
func WithSlackUsername(username string) SlackOption {
	return func(n *SlackNotifier) { n.Username = username }
}

// Notify implements Notifier.
func (n *SlackNotifier) Notify(ctx context.Context, event NotificationEvent) error {
	// Format message for Slack
	emoji := n.emojiForEvent(event)
	color := n.colorForSeverity(event.Severity)

	payload := slackPayload{
		Username: n.Username,
		Attachments: []slackAttachment{
			{
				Color:      color,
				Title:      fmt.Sprintf("%s %s", emoji, event.Type),
				Text:       event.Message,
				Footer:     fmt.Sprintf("Flow: %s | Run: %s", event.FlowID, event.RunID),
				FooterIcon: "https://cdn.anthropic.com/claude-logo-32.png",
				Timestamp:  event.Timestamp.Unix(),
				Fields:     n.fieldsFromMetadata(event.Metadata),
			},
		},
	}

	if n.Channel != "" {
		payload.Channel = n.Channel
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.Client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned %d", resp.StatusCode)
	}

	return nil
}

func (n *SlackNotifier) emojiForEvent(event NotificationEvent) string {
	switch event.Type {
	case EventRunStarted:
		return "🚀"
	case EventRunCompleted:
		return "✅"
	case EventRunFailed:
		return "❌"
	case EventPRCreated:
		return "🔗"
	case EventReviewNeeded:
		return "👀"
	case EventNodeStarted:
		return "▶️"
	case EventNodeCompleted:
		return "✓"
	case EventNodeFailed:
		return "⚠️"
	default:
		return "📢"
	}
}

func (n *SlackNotifier) colorForSeverity(severity string) string {
	switch severity {
	case SeverityError:
		return "danger"
	case SeverityWarning:
		return "warning"
	default:
		return "good"
	}
}

func (n *SlackNotifier) fieldsFromMetadata(metadata map[string]any) []slackField {
	if len(metadata) == 0 {
		return nil
	}

	var fields []slackField
	for k, v := range metadata {
		fields = append(fields, slackField{
			Title: k,
			Value: fmt.Sprintf("%v", v),
			Short: true,
		})
	}
	return fields
}

// Slack webhook payload types
type slackPayload struct {
	Username    string            `json:"username,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title"`
	Text       string       `json:"text"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	Fields     []slackField `json:"fields,omitempty"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// =============================================================================
// MultiNotifier
// =============================================================================

// MultiNotifier sends notifications to multiple notifiers.
type MultiNotifier struct {
	Notifiers []Notifier
	Logger    *slog.Logger
}

// NewMultiNotifier creates a notifier that fans out to multiple notifiers.
// Errors from individual notifiers are logged but don't stop other notifications.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{
		Notifiers: notifiers,
		Logger:    slog.Default(),
	}
}

// Notify implements Notifier.
func (n *MultiNotifier) Notify(ctx context.Context, event NotificationEvent) error {
	var lastErr error
	for _, notifier := range n.Notifiers {
		if err := notifier.Notify(ctx, event); err != nil {
			lastErr = err
			if n.Logger != nil {
				n.Logger.Warn("notifier failed",
					"error", err,
					"event_type", event.Type,
				)
			}
		}
	}
	return lastErr // Return last error, if any
}

// =============================================================================
// NopNotifier
// =============================================================================

// NopNotifier is a no-op notifier that discards all notifications.
// Useful for testing or when notifications are disabled.
type NopNotifier struct{}

// Notify implements Notifier.
func (NopNotifier) Notify(ctx context.Context, event NotificationEvent) error {
	return nil
}

// =============================================================================
// Context Injection
// =============================================================================

const notifierServiceKey serviceContextKey = "devflow.notifier"

// WithNotifier adds a Notifier to the context.
func WithNotifier(ctx context.Context, n Notifier) context.Context {
	return context.WithValue(ctx, notifierServiceKey, n)
}

// NotifierFromContext extracts the Notifier from context.
// Returns nil if no notifier is configured.
func NotifierFromContext(ctx context.Context) Notifier {
	if n, ok := ctx.Value(notifierServiceKey).(Notifier); ok {
		return n
	}
	return nil
}

// MustNotifierFromContext extracts the Notifier or panics.
func MustNotifierFromContext(ctx context.Context) Notifier {
	n := NotifierFromContext(ctx)
	if n == nil {
		panic("devflow: Notifier not found in context")
	}
	return n
}

// =============================================================================
// NotifyNode
// =============================================================================

// NotifyNode sends a notification based on current state.
// If no notifier is configured in context, this is a no-op.
func NotifyNode(ctx flowgraph.Context, state DevState) (DevState, error) {
	notifier := NotifierFromContext(ctx)
	if notifier == nil {
		return state, nil // No-op if no notifier
	}

	event := NotificationEvent{
		Type:      determineEventType(state),
		RunID:     state.RunID,
		FlowID:    state.FlowID,
		Timestamp: time.Now(),
		Severity:  determineSeverity(state),
		Message:   buildNotificationMessage(state),
		Metadata:  buildNotificationMetadata(state),
	}

	if err := notifier.Notify(ctx, event); err != nil {
		// Log but don't fail the workflow
		slog.Warn("notification failed", "error", err, "event", event.Type)
	}

	return state, nil
}

// determineEventType infers the event type from state.
func determineEventType(state DevState) EventType {
	// Check for PR creation
	if state.PR != nil && !state.PRCreated.IsZero() {
		return EventPRCreated
	}

	// Check for review needed
	if state.Review != nil && !state.Review.Approved {
		return EventReviewNeeded
	}

	// Check for failure
	if state.Error != "" {
		return EventRunFailed
	}

	// Default to completion if we have results
	if state.Spec != "" || state.Implementation != "" || state.PR != nil {
		return EventRunCompleted
	}

	return EventRunStarted
}

// determineSeverity determines notification severity from state.
func determineSeverity(state DevState) string {
	if state.Error != "" {
		return SeverityError
	}
	if state.Review != nil && !state.Review.Approved {
		return SeverityWarning
	}
	return SeverityInfo
}

// buildNotificationMessage creates a human-readable message.
func buildNotificationMessage(state DevState) string {
	switch determineEventType(state) {
	case EventPRCreated:
		if state.PR != nil {
			return fmt.Sprintf("PR created: %s", state.PR.URL)
		}
		return "PR created"
	case EventReviewNeeded:
		return fmt.Sprintf("Review needed: %d findings", len(state.Review.Findings))
	case EventRunFailed:
		return fmt.Sprintf("Run failed: %s", state.Error)
	case EventRunCompleted:
		return "Run completed successfully"
	default:
		return fmt.Sprintf("Workflow %s started", state.FlowID)
	}
}

// buildNotificationMetadata extracts relevant metadata from state.
func buildNotificationMetadata(state DevState) map[string]any {
	meta := make(map[string]any)

	if state.TicketID != "" {
		meta["ticket_id"] = state.TicketID
	}
	if state.Branch != "" {
		meta["branch"] = state.Branch
	}
	if state.TotalTokensIn > 0 || state.TotalTokensOut > 0 {
		meta["tokens_in"] = state.TotalTokensIn
		meta["tokens_out"] = state.TotalTokensOut
	}
	if state.TotalCost > 0 {
		meta["cost_usd"] = fmt.Sprintf("%.4f", state.TotalCost)
	}
	if state.TestOutput != nil {
		meta["tests_passed"] = state.TestOutput.PassedTests
		meta["tests_failed"] = state.TestOutput.FailedTests
	}

	if len(meta) == 0 {
		return nil
	}
	return meta
}

// =============================================================================
// Convenience Functions
// =============================================================================

// NotifyRunStarted sends a run started notification.
func NotifyRunStarted(ctx context.Context, state DevState) error {
	notifier := NotifierFromContext(ctx)
	if notifier == nil {
		return nil
	}

	return notifier.Notify(ctx, NotificationEvent{
		Type:      EventRunStarted,
		RunID:     state.RunID,
		FlowID:    state.FlowID,
		Message:   fmt.Sprintf("Starting workflow: %s", state.FlowID),
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"ticket_id": state.TicketID,
		},
	})
}

// NotifyRunCompleted sends a run completed notification.
func NotifyRunCompleted(ctx context.Context, state DevState) error {
	notifier := NotifierFromContext(ctx)
	if notifier == nil {
		return nil
	}

	return notifier.Notify(ctx, NotificationEvent{
		Type:      EventRunCompleted,
		RunID:     state.RunID,
		FlowID:    state.FlowID,
		Message:   fmt.Sprintf("Workflow completed: %s", state.FlowID),
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
		Metadata:  buildNotificationMetadata(state),
	})
}

// NotifyRunFailed sends a run failed notification.
func NotifyRunFailed(ctx context.Context, state DevState, err error) error {
	notifier := NotifierFromContext(ctx)
	if notifier == nil {
		return nil
	}

	return notifier.Notify(ctx, NotificationEvent{
		Type:      EventRunFailed,
		RunID:     state.RunID,
		FlowID:    state.FlowID,
		Message:   fmt.Sprintf("Workflow failed: %v", err),
		Severity:  SeverityError,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"error": err.Error(),
		},
	})
}

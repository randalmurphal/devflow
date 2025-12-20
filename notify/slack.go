package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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
func (n *SlackNotifier) Notify(ctx context.Context, event Event) error {
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

func (n *SlackNotifier) emojiForEvent(event Event) string {
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

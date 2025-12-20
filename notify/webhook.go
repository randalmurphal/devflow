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
func (n *WebhookNotifier) Notify(ctx context.Context, event Event) error {
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

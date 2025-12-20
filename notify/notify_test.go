package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Event Type Tests
// =============================================================================

func TestEventTypes(t *testing.T) {
	// Verify all event types are unique
	types := []EventType{
		EventRunStarted,
		EventRunCompleted,
		EventRunFailed,
		EventNodeStarted,
		EventNodeCompleted,
		EventNodeFailed,
		EventReviewNeeded,
		EventPRCreated,
	}

	seen := make(map[EventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true
	}
}

func TestSeverityLevels(t *testing.T) {
	// Verify severity levels are unique
	levels := []string{SeverityInfo, SeverityWarning, SeverityError}

	seen := make(map[string]bool)
	for _, s := range levels {
		if seen[s] {
			t.Errorf("duplicate severity: %s", s)
		}
		seen[s] = true
	}
}

// =============================================================================
// NopNotifier Tests
// =============================================================================

func TestNopNotifier(t *testing.T) {
	n := NopNotifier{}
	ctx := context.Background()

	err := n.Notify(ctx, Event{
		Type:    EventRunStarted,
		Message: "test",
	})

	if err != nil {
		t.Errorf("NopNotifier.Notify() error = %v, want nil", err)
	}
}

// =============================================================================
// LogNotifier Tests
// =============================================================================

func TestLogNotifier(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	n := NewLogNotifier(logger)
	ctx := context.Background()

	event := Event{
		Type:      EventRunCompleted,
		RunID:     "run-123",
		FlowID:    "test-flow",
		Message:   "Test completed",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}

	err := n.Notify(ctx, event)
	if err != nil {
		t.Errorf("LogNotifier.Notify() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test completed") {
		t.Errorf("Log output missing message: %s", output)
	}
	if !strings.Contains(output, "run-123") {
		t.Errorf("Log output missing run_id: %s", output)
	}
}

func TestLogNotifier_Severity(t *testing.T) {
	tests := []struct {
		severity string
		wantLog  string
	}{
		{SeverityInfo, "level=INFO"},
		{SeverityWarning, "level=WARN"},
		{SeverityError, "level=ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			n := NewLogNotifier(logger)

			err := n.Notify(context.Background(), Event{
				Type:     EventRunStarted,
				Message:  "test",
				Severity: tt.severity,
			})

			if err != nil {
				t.Errorf("Notify() error = %v", err)
			}

			if !strings.Contains(buf.String(), tt.wantLog) {
				t.Errorf("Log output = %q, want to contain %q", buf.String(), tt.wantLog)
			}
		})
	}
}

func TestLogNotifier_NilLogger(t *testing.T) {
	n := NewLogNotifier(nil)
	if n.Logger == nil {
		t.Error("NewLogNotifier should use default logger when nil")
	}
}

// =============================================================================
// WebhookNotifier Tests
// =============================================================================

func TestWebhookNotifier(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := NewWebhookNotifier(server.URL, nil)
	ctx := context.Background()

	event := Event{
		Type:      EventRunCompleted,
		RunID:     "run-123",
		FlowID:    "test-flow",
		Message:   "Webhook test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}

	err := n.Notify(ctx, event)
	if err != nil {
		t.Errorf("WebhookNotifier.Notify() error = %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(receivedBody, &parsed); err != nil {
		t.Errorf("Failed to parse received body: %v", err)
	}
	if parsed.RunID != "run-123" {
		t.Errorf("Received RunID = %s, want run-123", parsed.RunID)
	}
}

func TestWebhookNotifier_CustomHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"Authorization": "Bearer test-token",
	}
	n := NewWebhookNotifier(server.URL, headers)

	err := n.Notify(context.Background(), Event{Type: EventRunStarted})
	if err != nil {
		t.Errorf("Notify() error = %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want 'Bearer test-token'", receivedAuth)
	}
}

func TestWebhookNotifier_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	n := NewWebhookNotifier(server.URL, nil)
	err := n.Notify(context.Background(), Event{Type: EventRunStarted})

	if err == nil {
		t.Error("Notify() should return error for 500 status")
	}
}

func TestWebhookNotifier_NetworkError(t *testing.T) {
	n := NewWebhookNotifier("http://localhost:99999", nil) // Invalid port
	err := n.Notify(context.Background(), Event{Type: EventRunStarted})

	if err == nil {
		t.Error("Notify() should return error for network failure")
	}
}

// =============================================================================
// SlackNotifier Tests
// =============================================================================

func TestSlackNotifier(t *testing.T) {
	var receivedPayload slackPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := NewSlackNotifier(server.URL,
		WithSlackChannel("#test"),
		WithSlackUsername("testbot"),
	)

	event := Event{
		Type:      EventPRCreated,
		RunID:     "run-123",
		FlowID:    "ticket-to-pr",
		Message:   "PR created: https://github.com/org/repo/pull/1",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"pr_url": "https://github.com/org/repo/pull/1",
		},
	}

	err := n.Notify(context.Background(), event)
	if err != nil {
		t.Errorf("SlackNotifier.Notify() error = %v", err)
	}

	if receivedPayload.Channel != "#test" {
		t.Errorf("Channel = %s, want #test", receivedPayload.Channel)
	}
	if receivedPayload.Username != "testbot" {
		t.Errorf("Username = %s, want testbot", receivedPayload.Username)
	}
	if len(receivedPayload.Attachments) == 0 {
		t.Error("Missing attachments")
	}
}

func TestSlackNotifier_EmojiForEvent(t *testing.T) {
	n := &SlackNotifier{}

	tests := []struct {
		eventType EventType
		wantEmoji string
	}{
		{EventRunStarted, "🚀"},
		{EventRunCompleted, "✅"},
		{EventRunFailed, "❌"},
		{EventPRCreated, "🔗"},
		{EventReviewNeeded, "👀"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			emoji := n.emojiForEvent(Event{Type: tt.eventType})
			if emoji != tt.wantEmoji {
				t.Errorf("emojiForEvent() = %s, want %s", emoji, tt.wantEmoji)
			}
		})
	}
}

func TestSlackNotifier_ColorForSeverity(t *testing.T) {
	n := &SlackNotifier{}

	tests := []struct {
		severity  string
		wantColor string
	}{
		{SeverityInfo, "good"},
		{SeverityWarning, "warning"},
		{SeverityError, "danger"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			color := n.colorForSeverity(tt.severity)
			if color != tt.wantColor {
				t.Errorf("colorForSeverity() = %s, want %s", color, tt.wantColor)
			}
		})
	}
}

// =============================================================================
// MultiNotifier Tests
// =============================================================================

func TestMultiNotifier(t *testing.T) {
	var calls []string

	notifier1 := &mockNotifier{name: "n1", calls: &calls}
	notifier2 := &mockNotifier{name: "n2", calls: &calls}

	multi := NewMultiNotifier(notifier1, notifier2)

	err := multi.Notify(context.Background(), Event{Type: EventRunStarted})
	if err != nil {
		t.Errorf("MultiNotifier.Notify() error = %v", err)
	}

	if len(calls) != 2 {
		t.Errorf("Call count = %d, want 2", len(calls))
	}
	if calls[0] != "n1" || calls[1] != "n2" {
		t.Errorf("Calls = %v, want [n1, n2]", calls)
	}
}

func TestMultiNotifier_ContinuesOnError(t *testing.T) {
	var calls []string

	notifier1 := &mockNotifier{name: "n1", calls: &calls, err: context.DeadlineExceeded}
	notifier2 := &mockNotifier{name: "n2", calls: &calls}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	multi := NewMultiNotifier(notifier1, notifier2)
	multi.Logger = logger

	err := multi.Notify(context.Background(), Event{Type: EventRunStarted})

	// Should return the error but still call both notifiers
	if err == nil {
		t.Error("MultiNotifier should return last error")
	}

	if len(calls) != 2 {
		t.Errorf("Call count = %d, want 2 (both notifiers called)", len(calls))
	}
}

type mockNotifier struct {
	name  string
	calls *[]string
	err   error
}

func (m *mockNotifier) Notify(ctx context.Context, event Event) error {
	*m.calls = append(*m.calls, m.name)
	return m.err
}

// =============================================================================
// Context Injection Tests
// =============================================================================

func TestNotifierContextInjection(t *testing.T) {
	ctx := context.Background()

	// Without injection
	if NotifierFromContext(ctx) != nil {
		t.Error("NotifierFromContext should return nil without injection")
	}

	// With injection
	notifier := NopNotifier{}
	ctx = WithNotifier(ctx, notifier)

	if NotifierFromContext(ctx) == nil {
		t.Error("NotifierFromContext should not return nil after injection")
	}
}

func TestMustNotifierFromContext_Panics(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNotifierFromContext should panic without injection")
		}
	}()

	MustNotifierFromContext(ctx)
}

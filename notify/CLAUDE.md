# notify package

Notification services for workflow events (Slack, webhooks, logging).

## Quick Reference

| Type | Purpose |
|------|---------|
| `Notifier` | Interface for sending notifications |
| `Event` | Notification event with type and message |
| `EventType` | Event type constant |
| `SlackNotifier` | Slack webhook notifications |
| `WebhookNotifier` | Generic webhook notifications |
| `LogNotifier` | Log-based notifications (testing) |
| `MultiNotifier` | Combines multiple notifiers |
| `NopNotifier` | No-op notifier (testing) |

## Event Types

| Constant | When |
|----------|------|
| `EventStarted` | Workflow run started |
| `EventCompleted` | Run completed successfully |
| `EventFailed` | Run failed with error |
| `EventStepCompleted` | Individual step completed |

## Creating Notifiers

```go
// Slack
slack := notify.NewSlack(webhookURL,
    notify.WithChannel("#dev-alerts"),
    notify.WithUsername("devflow-bot"),
)

// Webhook
webhook := notify.NewWebhook(url, headers)

// Log (for testing)
log := notify.NewLogNotifier(logger)

// Combine multiple
multi := notify.NewMulti(slack, webhook)
```

## Sending Notifications

```go
err := notifier.Notify(ctx, notify.Event{
    Type:     notify.EventCompleted,
    Message:  "Workflow completed",
    Metadata: map[string]any{"runID": "run-123"},
})
```

## Context Integration

```go
// Inject into context
ctx = notify.WithNotifier(ctx, notifier)

// Retrieve from context
n := notify.NotifierFromContext(ctx)
```

## File Structure

```
notify/
├── notify.go    # Notifier interface, Event, EventType
├── slack.go     # SlackNotifier
├── webhook.go   # WebhookNotifier
├── log.go       # LogNotifier
└── multi.go     # MultiNotifier, NopNotifier
```

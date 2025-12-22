package jira

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// WebhookEventType represents a Jira webhook event type.
type WebhookEventType string

// Webhook event types for Jira webhooks.
const (
	WebhookEventIssueCreated   WebhookEventType = "jira:issue_created"
	WebhookEventIssueUpdated   WebhookEventType = "jira:issue_updated"
	WebhookEventIssueDeleted   WebhookEventType = "jira:issue_deleted"
	WebhookEventCommentCreated WebhookEventType = "comment_created"
	WebhookEventCommentUpdated WebhookEventType = "comment_updated"
	WebhookEventCommentDeleted WebhookEventType = "comment_deleted"
)

// WebhookPayload represents the common Jira webhook payload.
type WebhookPayload struct {
	Timestamp      int64            `json:"timestamp"`
	WebhookEvent   WebhookEventType `json:"webhookEvent"`
	IssueEventType string           `json:"issue_event_type_name,omitempty"`
	User           *User            `json:"user,omitempty"`
	Issue          *Issue           `json:"issue,omitempty"`
	Comment        *Comment         `json:"comment,omitempty"`
	Changelog      *Changelog       `json:"changelog,omitempty"`
}

// Changelog represents the changelog in a webhook payload.
type Changelog struct {
	ID    string          `json:"id"`
	Items []ChangelogItem `json:"items"`
}

// ChangelogItem represents a single change in the changelog.
type ChangelogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype"`
	FieldID    string `json:"fieldId,omitempty"`
	From       string `json:"from,omitempty"`
	FromString string `json:"fromString,omitempty"`
	To         string `json:"to,omitempty"`
	ToString   string `json:"toString,omitempty"`
}

// HasFieldChange checks if the changelog contains a change for the specified field.
func (c *Changelog) HasFieldChange(fieldName string) bool {
	if c == nil {
		return false
	}
	for _, item := range c.Items {
		if strings.EqualFold(item.Field, fieldName) {
			return true
		}
	}
	return false
}

// GetFieldChange returns the changelog item for a specific field, or nil if not found.
func (c *Changelog) GetFieldChange(fieldName string) *ChangelogItem {
	if c == nil {
		return nil
	}
	for i := range c.Items {
		if strings.EqualFold(c.Items[i].Field, fieldName) {
			return &c.Items[i]
		}
	}
	return nil
}

// ValidateWebhookSignature validates a Jira webhook signature.
// It supports both X-Hub-Signature-256 and X-Atlassian-Webhook-Signature headers.
// The signature should be in the format "sha256=<hex-encoded-signature>" or just
// the hex-encoded signature.
func ValidateWebhookSignature(body []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	// Remove prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// ParseWebhookPayload parses a Jira webhook payload from JSON bytes.
func ParseWebhookPayload(body []byte) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ErrWebhookInvalidPayload
	}
	return &payload, nil
}

// WebhookSignatureHeaders are the possible headers containing the webhook signature.
var WebhookSignatureHeaders = []string{
	"X-Hub-Signature-256",
	"X-Atlassian-Webhook-Signature",
}

// GetChangedFields returns the list of fields that changed in the webhook.
func (p *WebhookPayload) GetChangedFields() []string {
	if p.Changelog == nil {
		return nil
	}

	fields := make([]string, 0, len(p.Changelog.Items))
	for _, item := range p.Changelog.Items {
		fields = append(fields, item.Field)
	}
	return fields
}

// HasFieldChange checks if a specific field changed in the webhook.
func (p *WebhookPayload) HasFieldChange(field string) bool {
	if p.Changelog == nil {
		return false
	}
	return p.Changelog.HasFieldChange(field)
}

// GetFieldChange returns the change details for a specific field.
func (p *WebhookPayload) GetFieldChange(field string) *ChangelogItem {
	if p.Changelog == nil {
		return nil
	}
	return p.Changelog.GetFieldChange(field)
}

// IsStatusChange returns true if the issue status changed.
func (p *WebhookPayload) IsStatusChange() bool {
	return p.HasFieldChange("status")
}

// IsAssigneeChange returns true if the issue assignee changed.
func (p *WebhookPayload) IsAssigneeChange() bool {
	return p.HasFieldChange("assignee")
}

// IsPriorityChange returns true if the issue priority changed.
func (p *WebhookPayload) IsPriorityChange() bool {
	return p.HasFieldChange("priority")
}

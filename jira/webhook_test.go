package jira

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestValidateWebhookSignature(t *testing.T) {
	secret := "webhook-secret"
	body := []byte(`{"webhookEvent":"jira:issue_created"}`)

	// Calculate valid signature
	// Using HMAC-SHA256 with the secret
	validSig := "sha256=e9a8d0b3a5c7f9e2d1b4a6c8e0f2d4a6b8c0e2f4a6b8c0e2f4a6b8c0e2f4a6b8"

	tests := []struct {
		name      string
		body      []byte
		signature string
		secret    string
		want      bool
	}{
		{
			name:      "empty signature",
			body:      body,
			signature: "",
			secret:    secret,
			want:      false,
		},
		{
			name:      "empty secret",
			body:      body,
			signature: validSig,
			secret:    "",
			want:      false,
		},
		{
			name:      "wrong signature",
			body:      body,
			signature: "sha256=invalid",
			secret:    secret,
			want:      false,
		},
		{
			name:      "signature without prefix",
			body:      body,
			signature: "invalid",
			secret:    secret,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateWebhookSignature(tt.body, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("ValidateWebhookSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateWebhookSignatureValid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"test":"data"}`)

	// Generate a valid signature
	// sha256 HMAC of `{"test":"data"}` with secret "test-secret"
	// = 6b4e8c...  (this would be calculated)

	// For this test, we verify that the same body+secret produces consistent results
	sig1 := computeSignature(body, secret)
	sig2 := computeSignature(body, secret)

	if sig1 != sig2 {
		t.Error("Same input should produce same signature")
	}

	// And that the validation passes with that signature
	if !ValidateWebhookSignature(body, sig1, secret) {
		t.Error("Valid signature should pass validation")
	}

	// Different body should fail
	if ValidateWebhookSignature([]byte(`{"different":"data"}`), sig1, secret) {
		t.Error("Different body should fail validation")
	}
}

// computeSignature is a helper to generate signatures for testing
func computeSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestParseWebhookPayload(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
		check   func(*WebhookPayload) bool
	}{
		{
			name:    "valid issue created",
			body:    []byte(`{"webhookEvent":"jira:issue_created","issue":{"key":"TEST-123"}}`),
			wantErr: false,
			check: func(p *WebhookPayload) bool {
				return p.WebhookEvent == WebhookEventIssueCreated && p.Issue != nil && p.Issue.Key == "TEST-123"
			},
		},
		{
			name:    "valid issue updated with changelog",
			body:    []byte(`{"webhookEvent":"jira:issue_updated","changelog":{"id":"12345","items":[{"field":"status","fromString":"Open","toString":"Done"}]}}`),
			wantErr: false,
			check: func(p *WebhookPayload) bool {
				return p.WebhookEvent == WebhookEventIssueUpdated && p.Changelog != nil && len(p.Changelog.Items) == 1
			},
		},
		{
			name:    "valid comment created",
			body:    []byte(`{"webhookEvent":"comment_created","comment":{"id":"99999","body":"test"}}`),
			wantErr: false,
			check: func(p *WebhookPayload) bool {
				return p.WebhookEvent == WebhookEventCommentCreated && p.Comment != nil
			},
		},
		{
			name:    "invalid JSON",
			body:    []byte(`not json`),
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    []byte(``),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := ParseWebhookPayload(tt.body)
			if tt.wantErr {
				if err == nil {
					t.Error("ParseWebhookPayload() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseWebhookPayload() unexpected error: %v", err)
				return
			}
			if tt.check != nil && !tt.check(payload) {
				t.Errorf("ParseWebhookPayload() payload check failed")
			}
		})
	}
}

func TestChangelogHasFieldChange(t *testing.T) {
	changelog := &Changelog{
		ID: "123",
		Items: []ChangelogItem{
			{Field: "status", FromString: "Open", ToString: "In Progress"},
			{Field: "assignee", FromString: "", ToString: "jsmith"},
		},
	}

	if !changelog.HasFieldChange("status") {
		t.Error("HasFieldChange('status') should be true")
	}
	if !changelog.HasFieldChange("STATUS") { // case insensitive
		t.Error("HasFieldChange('STATUS') should be true (case insensitive)")
	}
	if !changelog.HasFieldChange("assignee") {
		t.Error("HasFieldChange('assignee') should be true")
	}
	if changelog.HasFieldChange("priority") {
		t.Error("HasFieldChange('priority') should be false")
	}

	// nil changelog
	var nilChangelog *Changelog
	if nilChangelog.HasFieldChange("status") {
		t.Error("nil changelog.HasFieldChange should be false")
	}
}

func TestChangelogGetFieldChange(t *testing.T) {
	changelog := &Changelog{
		ID: "123",
		Items: []ChangelogItem{
			{Field: "status", FromString: "Open", ToString: "Done"},
		},
	}

	item := changelog.GetFieldChange("status")
	if item == nil {
		t.Fatal("GetFieldChange('status') should not be nil")
	}
	if item.FromString != "Open" || item.ToString != "Done" {
		t.Errorf("GetFieldChange returned wrong values: %+v", item)
	}

	item = changelog.GetFieldChange("priority")
	if item != nil {
		t.Error("GetFieldChange('priority') should be nil")
	}

	var nilChangelog *Changelog
	if nilChangelog.GetFieldChange("status") != nil {
		t.Error("nil changelog.GetFieldChange should be nil")
	}
}

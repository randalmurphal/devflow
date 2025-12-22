// Package jira provides a client for the Jira REST API.
//
// This package supports both Jira Cloud (API v3) and Jira Server/Data Center
// (API v2). The client auto-detects the deployment type and uses the
// appropriate API version.
//
// # Authentication
//
// The client supports multiple authentication methods:
//   - API Token (Cloud): Email + API token
//   - Personal Access Token (Server/DC): PAT token
//   - Basic Auth (legacy): Username + password
//   - OAuth 2.0 (Cloud): Client credentials
//
// # Usage
//
//	cfg := &jira.Config{
//		URL:        "https://your-domain.atlassian.net",
//		Auth: jira.AuthConfig{
//			Type:  jira.AuthAPIToken,
//			Email: "you@example.com",
//			Token: "your-api-token",
//		},
//	}
//
//	client, err := jira.NewClient(cfg)
//	if err != nil {
//		return err
//	}
//
//	issue, err := client.GetIssue(ctx, "PROJ-123")
//
// # Rich Text
//
// Jira Cloud uses Atlassian Document Format (ADF) for rich text fields like
// description and comments. Jira Server uses Wiki Markup. This package
// provides converters between these formats and Markdown:
//
//	// Convert Markdown to ADF for Cloud
//	adf := jira.MarkdownToADF("**bold** text")
//
//	// Convert Wiki Markup to Markdown
//	md := jira.WikiToMarkdown("*bold* text")
//
// # Error Handling
//
// The package uses devflow/http error types for consistent error handling
// across integrations. Use errors.Is() to check for specific conditions:
//
//	if errors.Is(err, http.ErrNotFound) {
//		// Issue doesn't exist
//	}
//	if errors.Is(err, http.ErrRateLimited) {
//		// Rate limited, check Retry-After
//	}
package jira

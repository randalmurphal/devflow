package jira

import (
	"regexp"
	"time"
)

// DeploymentType represents the type of Jira deployment.
type DeploymentType string

// Deployment types for Jira instances.
const (
	DeploymentCloud      DeploymentType = "Cloud"
	DeploymentServer     DeploymentType = "Server"
	DeploymentDataCenter DeploymentType = "DataCenter"
)

// TimeFormat is the standard Jira timestamp format.
const TimeFormat = "2006-01-02T15:04:05.000-0700"

// APIVersion represents the Jira REST API version.
type APIVersion string

// API versions supported by the Jira REST API.
const (
	APIVersionAuto APIVersion = "auto"
	APIVersionV2   APIVersion = "v2"
	APIVersionV3   APIVersion = "v3"
)

// ServerInfo represents the response from /rest/api/X/serverInfo.
type ServerInfo struct {
	BaseURL        string `json:"baseUrl"`
	Version        string `json:"version"`
	VersionNumbers []int  `json:"versionNumbers"`
	DeploymentType string `json:"deploymentType"` // "Cloud", "Server", "DataCenter"
	BuildNumber    int    `json:"buildNumber"`
	BuildDate      string `json:"buildDate"`
	ServerTime     string `json:"serverTime"`
	ScmInfo        string `json:"scmInfo,omitempty"` // Cloud only
	ServerTitle    string `json:"serverTitle"`
}

// User represents a Jira user.
type User struct {
	AccountID    string            `json:"accountId,omitempty"`    // Cloud (GDPR-compliant)
	Name         string            `json:"name,omitempty"`         // Server (username)
	Key          string            `json:"key,omitempty"`          // Server (user key)
	EmailAddress string            `json:"emailAddress,omitempty"` // May require scope
	DisplayName  string            `json:"displayName"`
	Active       bool              `json:"active"`
	TimeZone     string            `json:"timeZone,omitempty"`
	AvatarURLs   map[string]string `json:"avatarUrls,omitempty"`
	Self         string            `json:"self,omitempty"`
}

// GetID returns the user identifier (accountId for Cloud, name for Server).
func (u *User) GetID() string {
	if u.AccountID != "" {
		return u.AccountID
	}
	return u.Name
}

// Project represents a Jira project.
type Project struct {
	ID          string            `json:"id"`
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Self        string            `json:"self"`
	AvatarURLs  map[string]string `json:"avatarUrls,omitempty"`
}

// IssueType represents an issue type in Jira.
type IssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Subtask     bool   `json:"subtask"`
	IconURL     string `json:"iconUrl,omitempty"`
	Self        string `json:"self,omitempty"`
}

// Priority represents an issue priority.
type Priority struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl,omitempty"`
	Self    string `json:"self,omitempty"`
}

// Status represents an issue status.
type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	IconURL        string         `json:"iconUrl,omitempty"`
	StatusCategory StatusCategory `json:"statusCategory"`
	Self           string         `json:"self,omitempty"`
}

// StatusCategory represents a status category.
type StatusCategory struct {
	ID        int    `json:"id"`
	Key       string `json:"key"` // "new", "indeterminate", "done"
	Name      string `json:"name"`
	ColorName string `json:"colorName,omitempty"`
	Self      string `json:"self,omitempty"`
}

// Resolution represents an issue resolution.
type Resolution struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Self        string `json:"self,omitempty"`
}

// Component represents a project component.
type Component struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Self        string `json:"self,omitempty"`
}

// Version represents a project version (fix version).
type Version struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Archived    bool   `json:"archived"`
	Released    bool   `json:"released"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	Self        string `json:"self,omitempty"`
}

// Issue represents a Jira issue.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains the fields of a Jira issue.
type IssueFields struct {
	Summary     string      `json:"summary"`
	Description any         `json:"description,omitempty"` // ADF (v3) or string (v2)
	Environment any         `json:"environment,omitempty"` // ADF (v3) or string (v2)
	Status      *Status     `json:"status,omitempty"`
	Priority    *Priority   `json:"priority,omitempty"`
	IssueType   *IssueType  `json:"issuetype,omitempty"`
	Project     *Project    `json:"project,omitempty"`
	Assignee    *User       `json:"assignee,omitempty"`
	Reporter    *User       `json:"reporter,omitempty"`
	Creator     *User       `json:"creator,omitempty"`
	Resolution  *Resolution `json:"resolution,omitempty"`
	Labels      []string    `json:"labels,omitempty"`
	Components  []Component `json:"components,omitempty"`
	FixVersions []Version   `json:"fixVersions,omitempty"`
	Created     string      `json:"created,omitempty"`
	Updated     string      `json:"updated,omitempty"`
	DueDate     string      `json:"duedate,omitempty"`

	// Custom fields are stored here with their field IDs as keys
	// e.g., "customfield_10001": 5.0 (story points)
	CustomFields map[string]any `json:"-"`

	// Parent for subtasks
	Parent *Issue `json:"parent,omitempty"`

	// Timetracking
	TimeTracking *TimeTracking `json:"timetracking,omitempty"`
}

// TimeTracking represents time tracking data.
type TimeTracking struct {
	OriginalEstimate         string `json:"originalEstimate,omitempty"`
	RemainingEstimate        string `json:"remainingEstimate,omitempty"`
	TimeSpent                string `json:"timeSpent,omitempty"`
	OriginalEstimateSeconds  int    `json:"originalEstimateSeconds,omitempty"`
	RemainingEstimateSeconds int    `json:"remainingEstimateSeconds,omitempty"`
	TimeSpentSeconds         int    `json:"timeSpentSeconds,omitempty"`
}

// CreatedTime parses and returns the Created timestamp.
func (f *IssueFields) CreatedTime() (time.Time, error) {
	return ParseTime(f.Created)
}

// UpdatedTime parses and returns the Updated timestamp.
func (f *IssueFields) UpdatedTime() (time.Time, error) {
	return ParseTime(f.Updated)
}

// Transition represents an available status transition.
type Transition struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	To            *Status `json:"to"`
	HasScreen     bool    `json:"hasScreen"`
	IsGlobal      bool    `json:"isGlobal"`
	IsInitial     bool    `json:"isInitial"`
	IsConditional bool    `json:"isConditional"`
}

// TransitionsResponse represents the response from the transitions endpoint.
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// Comment represents a Jira comment.
type Comment struct {
	ID           string             `json:"id"`
	Self         string             `json:"self,omitempty"`
	Author       *User              `json:"author,omitempty"`
	UpdateAuthor *User              `json:"updateAuthor,omitempty"`
	Body         any                `json:"body"` // ADF (v3) or string (v2)
	Created      string             `json:"created"`
	Updated      string             `json:"updated"`
	Visibility   *CommentVisibility `json:"visibility,omitempty"`
}

// CreatedTime parses and returns the Created timestamp.
func (c *Comment) CreatedTime() (time.Time, error) {
	return ParseTime(c.Created)
}

// UpdatedTime parses and returns the Updated timestamp.
func (c *Comment) UpdatedTime() (time.Time, error) {
	return ParseTime(c.Updated)
}

// CommentVisibility represents comment visibility restrictions.
type CommentVisibility struct {
	Type  string `json:"type"`  // "group" or "role"
	Value string `json:"value"` // group name or role name
}

// CommentsResponse represents the response from the comments endpoint.
type CommentsResponse struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// SearchResponse represents the response from the search endpoint.
type SearchResponse struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// RemoteLink represents a remote link on an issue.
type RemoteLink struct {
	ID           int              `json:"id,omitempty"`
	Self         string           `json:"self,omitempty"`
	GlobalID     string           `json:"globalId,omitempty"`
	Application  *RemoteLinkApp   `json:"application,omitempty"`
	Relationship string           `json:"relationship,omitempty"`
	Object       RemoteLinkObject `json:"object"`
}

// RemoteLinkApp represents the application information for a remote link.
type RemoteLinkApp struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// RemoteLinkObject represents the linked object details.
type RemoteLinkObject struct {
	URL     string            `json:"url"`
	Title   string            `json:"title"`
	Summary string            `json:"summary,omitempty"`
	Icon    *RemoteLinkIcon   `json:"icon,omitempty"`
	Status  *RemoteLinkStatus `json:"status,omitempty"`
}

// RemoteLinkIcon represents the icon for a remote link.
type RemoteLinkIcon struct {
	URL16x16 string `json:"url16x16,omitempty"`
	Title    string `json:"title,omitempty"`
}

// RemoteLinkStatus represents the status of a remote link.
type RemoteLinkStatus struct {
	Resolved bool            `json:"resolved"`
	Icon     *RemoteLinkIcon `json:"icon,omitempty"`
}

// CreateIssueRequest represents a request to create an issue.
type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

// CreateIssueFields represents the fields for creating an issue.
type CreateIssueFields struct {
	Project     ProjectRef     `json:"project"`
	IssueType   IssueTypeRef   `json:"issuetype"`
	Summary     string         `json:"summary"`
	Description any            `json:"description,omitempty"` // ADF or string
	Priority    *PriorityRef   `json:"priority,omitempty"`
	Assignee    *UserRef       `json:"assignee,omitempty"`
	Labels      []string       `json:"labels,omitempty"`
	Components  []ComponentRef `json:"components,omitempty"`
	FixVersions []VersionRef   `json:"fixVersions,omitempty"`
	DueDate     string         `json:"duedate,omitempty"`
	Parent      *IssueRef      `json:"parent,omitempty"`

	// Custom fields
	CustomFields map[string]any `json:"-"`
}

// ProjectRef references a project by key or ID.
type ProjectRef struct {
	Key string `json:"key,omitempty"`
	ID  string `json:"id,omitempty"`
}

// IssueTypeRef references an issue type by name or ID.
type IssueTypeRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// PriorityRef references a priority by name or ID.
type PriorityRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// UserRef references a user by accountId (Cloud) or name (Server).
type UserRef struct {
	AccountID string `json:"accountId,omitempty"` // Cloud
	Name      string `json:"name,omitempty"`      // Server
}

// ComponentRef references a component by name or ID.
type ComponentRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// VersionRef references a version by name or ID.
type VersionRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// IssueRef references an issue by key or ID.
type IssueRef struct {
	Key string `json:"key,omitempty"`
	ID  string `json:"id,omitempty"`
}

// CreateIssueResponse represents the response from creating an issue.
type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// TransitionRequest represents a request to transition an issue.
type TransitionRequest struct {
	Transition TransitionRef  `json:"transition"`
	Fields     map[string]any `json:"fields,omitempty"`
	Update     map[string]any `json:"update,omitempty"`
}

// TransitionRef references a transition by ID.
type TransitionRef struct {
	ID string `json:"id"`
}

// UpdateIssueRequest represents a request to update issue fields.
type UpdateIssueRequest struct {
	Fields map[string]any `json:"fields,omitempty"`
	Update map[string]any `json:"update,omitempty"`
}

// AddCommentRequest represents a request to add a comment.
type AddCommentRequest struct {
	Body       any                `json:"body"` // ADF or string
	Visibility *CommentVisibility `json:"visibility,omitempty"`
}

// issueKeyRegex validates Jira issue keys (e.g., PROJ-123).
var issueKeyRegex = regexp.MustCompile(`^[A-Z][A-Z0-9]*-\d+$`)

// ValidateIssueKey validates a Jira issue key format.
func ValidateIssueKey(key string) bool {
	return issueKeyRegex.MatchString(key)
}

// ParseTime parses a Jira timestamp string.
// Jira uses ISO 8601 format with timezone offset.
func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	// Jira format: "2025-01-15T10:30:00.000+0000"
	formats := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, &time.ParseError{Value: s}
}

// FormatTime formats a time.Time as a Jira timestamp string.
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

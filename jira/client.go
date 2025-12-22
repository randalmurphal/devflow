package jira

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	devhttp "github.com/randalmurphal/devflow/http"
)

// Client provides access to the Jira REST API.
type Client struct {
	cfg        *Config
	httpClient *http.Client
	baseURL    string
	apiVersion APIVersion

	// Rate limiting state
	mu        sync.RWMutex
	remaining int
	resetTime time.Time

	// Deployment info (cached)
	deploymentType DeploymentType
	serverInfo     *ServerInfo
}

// ClientOption configures the client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new Jira client.
func NewClient(cfg *Config, opts ...ClientOption) (*Client, error) {
	if validateErr := cfg.Validate(); validateErr != nil {
		return nil, validateErr
	}

	timeout := cfg.HTTP.Timeout
	if timeout == 0 {
		timeout = devhttp.DefaultTimeout
	}

	c := &Client{
		cfg:     cfg,
		baseURL: strings.TrimSuffix(cfg.URL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       cfg.HTTP.MaxIdleConns,
				IdleConnTimeout:    cfg.HTTP.IdleConnTimeout,
				DisableCompression: false,
				DisableKeepAlives:  false,
			},
		},
		remaining: -1, // Unknown
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Resolve API version
	c.apiVersion = cfg.GetAPIVersion()

	return c, nil
}

// DetectDeployment detects the Jira deployment type by calling serverInfo.
func (c *Client) DetectDeployment(ctx context.Context) (DeploymentType, error) {
	info, infoErr := c.GetServerInfo(ctx)
	if infoErr != nil {
		return "", infoErr
	}

	c.serverInfo = info
	c.deploymentType = DeploymentType(info.DeploymentType)

	// Update API version if auto-detected
	if c.cfg.APIVersion == APIVersionAuto {
		if c.deploymentType == DeploymentCloud {
			c.apiVersion = APIVersionV3
		} else {
			c.apiVersion = APIVersionV2
		}
	}

	return c.deploymentType, nil
}

// GetServerInfo fetches server information.
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	// Try v3 first (Cloud), then v2
	for _, version := range []string{"3", "2"} {
		info, err := c.tryGetServerInfo(ctx, version)
		if err != nil {
			continue // Try next version
		}
		return info, nil
	}

	return nil, fmt.Errorf("failed to get server info from %s", c.baseURL)
}

// tryGetServerInfo attempts to get server info with a specific API version.
func (c *Client) tryGetServerInfo(ctx context.Context, version string) (*ServerInfo, error) {
	path := fmt.Sprintf("/rest/api/%s/serverInfo", version)
	req, reqErr := c.newRequest(ctx, http.MethodGet, path, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	// serverInfo allows anonymous access, skip auth
	req.Header.Del("Authorization")

	resp, respErr := c.doRequest(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var info ServerInfo
	if decodeErr := json.NewDecoder(resp.Body).Decode(&info); decodeErr != nil {
		return nil, decodeErr
	}

	return &info, nil
}

// GetIssue retrieves an issue by key.
func (c *Client) GetIssue(ctx context.Context, key string) (*Issue, error) {
	if !ValidateIssueKey(key) {
		return nil, ErrIssueKeyInvalid
	}

	path := c.apiPath("/issue/" + key)
	req, reqErr := c.newRequest(ctx, http.MethodGet, path, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var issue Issue
	if decodeErr := json.NewDecoder(resp.Body).Decode(&issue); decodeErr != nil {
		return nil, fmt.Errorf("decode issue: %w", decodeErr)
	}

	return &issue, nil
}

// SearchOptions configures issue search.
type SearchOptions struct {
	StartAt    int      `json:"startAt,omitempty"`
	MaxResults int      `json:"maxResults,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Expand     []string `json:"expand,omitempty"`
}

// SearchIssues searches for issues using JQL.
func (c *Client) SearchIssues(ctx context.Context, jql string, opts *SearchOptions) (*SearchResponse, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}
	if opts.MaxResults == 0 {
		opts.MaxResults = 50
	}

	path := c.apiPath("/search")
	body := map[string]any{
		"jql":        jql,
		"startAt":    opts.StartAt,
		"maxResults": opts.MaxResults,
	}
	if len(opts.Fields) > 0 {
		body["fields"] = opts.Fields
	}
	if len(opts.Expand) > 0 {
		body["expand"] = opts.Expand
	}

	req, reqErr := c.newRequest(ctx, http.MethodPost, path, body)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var result SearchResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return nil, fmt.Errorf("decode search response: %w", decodeErr)
	}

	return &result, nil
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(ctx context.Context, createReq *CreateIssueRequest) (*CreateIssueResponse, error) {
	path := c.apiPath("/issue")
	httpReq, httpReqErr := c.newRequest(ctx, http.MethodPost, path, createReq)
	if httpReqErr != nil {
		return nil, httpReqErr
	}

	resp, respErr := c.doWithRetry(httpReq)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var result CreateIssueResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return nil, fmt.Errorf("decode create issue response: %w", decodeErr)
	}

	return &result, nil
}

// UpdateIssue updates an issue's fields.
func (c *Client) UpdateIssue(ctx context.Context, key string, fields map[string]any) error {
	if !ValidateIssueKey(key) {
		return ErrIssueKeyInvalid
	}

	path := c.apiPath("/issue/" + key)
	body := &UpdateIssueRequest{Fields: fields}

	req, reqErr := c.newRequest(ctx, http.MethodPut, path, body)
	if reqErr != nil {
		return reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return apiErr
	}

	return nil
}

// GetTransitions gets available transitions for an issue.
func (c *Client) GetTransitions(ctx context.Context, key string) ([]Transition, error) {
	if !ValidateIssueKey(key) {
		return nil, ErrIssueKeyInvalid
	}

	path := c.apiPath("/issue/" + key + "/transitions")
	req, reqErr := c.newRequest(ctx, http.MethodGet, path, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var result TransitionsResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return nil, fmt.Errorf("decode transitions: %w", decodeErr)
	}

	return result.Transitions, nil
}

// TransitionIssue transitions an issue to a new status.
func (c *Client) TransitionIssue(ctx context.Context, key, transitionID string) error {
	if !ValidateIssueKey(key) {
		return ErrIssueKeyInvalid
	}
	if transitionID == "" {
		return ErrTransitionIDRequired
	}

	path := c.apiPath("/issue/" + key + "/transitions")
	body := &TransitionRequest{
		Transition: TransitionRef{ID: transitionID},
	}

	req, reqErr := c.newRequest(ctx, http.MethodPost, path, body)
	if reqErr != nil {
		return reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return apiErr
	}

	return nil
}

// TransitionIssueByName finds and executes a transition by name.
func (c *Client) TransitionIssueByName(ctx context.Context, key, transitionName string) error {
	transitions, getErr := c.GetTransitions(ctx, key)
	if getErr != nil {
		return getErr
	}

	for _, t := range transitions {
		if strings.EqualFold(t.Name, transitionName) {
			return c.TransitionIssue(ctx, key, t.ID)
		}
	}

	return ErrTransitionNotFound
}

// GetComments retrieves comments for an issue.
func (c *Client) GetComments(ctx context.Context, key string) ([]Comment, error) {
	if !ValidateIssueKey(key) {
		return nil, ErrIssueKeyInvalid
	}

	path := c.apiPath("/issue/" + key + "/comment")
	req, reqErr := c.newRequest(ctx, http.MethodGet, path, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var result CommentsResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return nil, fmt.Errorf("decode comments: %w", decodeErr)
	}

	return result.Comments, nil
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(ctx context.Context, key string, body any) (*Comment, error) {
	if !ValidateIssueKey(key) {
		return nil, ErrIssueKeyInvalid
	}

	path := c.apiPath("/issue/" + key + "/comment")
	reqBody := &AddCommentRequest{Body: body}

	req, reqErr := c.newRequest(ctx, http.MethodPost, path, reqBody)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var comment Comment
	if decodeErr := json.NewDecoder(resp.Body).Decode(&comment); decodeErr != nil {
		return nil, fmt.Errorf("decode comment: %w", decodeErr)
	}

	return &comment, nil
}

// AddRemoteLink adds a remote link to an issue.
func (c *Client) AddRemoteLink(ctx context.Context, key string, link *RemoteLink) (*RemoteLink, error) {
	if !ValidateIssueKey(key) {
		return nil, ErrIssueKeyInvalid
	}

	path := c.apiPath("/issue/" + key + "/remotelink")
	req, reqErr := c.newRequest(ctx, http.MethodPost, path, link)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var result RemoteLink
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return nil, fmt.Errorf("decode remote link: %w", decodeErr)
	}

	return &result, nil
}

// GetRemoteLinks retrieves remote links for an issue.
func (c *Client) GetRemoteLinks(ctx context.Context, key string) ([]RemoteLink, error) {
	if !ValidateIssueKey(key) {
		return nil, ErrIssueKeyInvalid
	}

	path := c.apiPath("/issue/" + key + "/remotelink")
	req, reqErr := c.newRequest(ctx, http.MethodGet, path, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := c.doWithRetry(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrIssueNotFound
	}
	if apiErr := c.checkError(resp); apiErr != nil {
		return nil, apiErr
	}

	var links []RemoteLink
	if decodeErr := json.NewDecoder(resp.Body).Decode(&links); decodeErr != nil {
		return nil, fmt.Errorf("decode remote links: %w", decodeErr)
	}

	return links, nil
}

// apiPath returns the full API path for the given endpoint.
func (c *Client) apiPath(endpoint string) string {
	version := c.apiVersion
	if version == APIVersionAuto {
		version = APIVersionV3 // Default to v3
	}
	return fmt.Sprintf("/rest/api/%s%s", strings.TrimPrefix(string(version), "v"), endpoint)
}

// newRequest creates a new HTTP request with authentication.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	u, parseErr := url.Parse(c.baseURL + path)
	if parseErr != nil {
		return nil, fmt.Errorf("parse url: %w", parseErr)
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal body: %w", marshalErr)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, reqErr := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if reqErr != nil {
		return nil, fmt.Errorf("create request: %w", reqErr)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set authentication
	c.setAuth(req)

	return req, nil
}

// setAuth sets the authentication header based on config.
func (c *Client) setAuth(req *http.Request) {
	switch c.cfg.Auth.Type {
	case AuthAPIToken:
		// Cloud: email:api_token base64 encoded
		credentials := c.cfg.Auth.Email + ":" + c.cfg.Auth.Token
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		req.Header.Set("Authorization", "Basic "+encoded)

	case AuthBasic:
		// Server: username:password base64 encoded
		credentials := c.cfg.Auth.Username + ":" + c.cfg.Auth.Password
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		req.Header.Set("Authorization", "Basic "+encoded)

	case AuthPAT:
		// Data Center: Bearer token
		req.Header.Set("Authorization", "Bearer "+c.cfg.Auth.Token)

	case AuthOAuth2:
		// OAuth2: Bearer access token
		if c.cfg.Auth.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.Auth.AccessToken)
		} else if c.cfg.Auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.Auth.Token)
		}
	}
}

// doRequest executes an HTTP request.
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// doWithRetry executes a request with retry on rate limiting.
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	maxRetries := c.cfg.RateLimit.MaxRetries
	if maxRetries == 0 {
		maxRetries = devhttp.DefaultMaxRetries
	}

	delay := c.cfg.RateLimit.RetryWaitMin
	if delay == 0 {
		delay = devhttp.DefaultRetryWait
	}

	maxDelay := c.cfg.RateLimit.RetryWaitMax
	if maxDelay == 0 {
		maxDelay = 30 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone request for retry (body needs to be re-readable)
		clonedReq := req.Clone(req.Context())
		if req.Body != nil {
			// Read body and create new readers
			bodyBytes, readErr := io.ReadAll(req.Body)
			if readErr != nil {
				return nil, fmt.Errorf("read request body: %w", readErr)
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			clonedReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, doErr := c.doRequest(clonedReq)
		if doErr != nil {
			lastErr = doErr
			if devhttp.IsRetryable(doErr) && attempt < maxRetries {
				c.waitForRetry(req.Context(), delay)
				delay = min(delay*2, maxDelay)
				continue
			}
			return nil, doErr
		}

		// Update rate limit state from headers
		c.updateRateLimitState(resp)

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Rate limited - close body and retry
		_ = resp.Body.Close()
		lastErr = devhttp.ErrRateLimited

		// Get retry delay from header if available
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
				delay = time.Duration(seconds) * time.Second
			}
		}

		// Apply jitter if enabled
		if c.cfg.RateLimit.RetryJitter {
			jitter := 0.7 + cryptoRandFloat64()*0.6
			delay = time.Duration(float64(delay) * jitter)
		}

		if attempt < maxRetries {
			c.waitForRetry(req.Context(), delay)
			delay = min(delay*2, maxDelay)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
	}
	return nil, fmt.Errorf("max retries exceeded")
}

// waitForRetry waits for the specified duration or until context is canceled.
func (c *Client) waitForRetry(ctx context.Context, delay time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

// updateRateLimitState updates rate limit tracking from response headers.
func (c *Client) updateRateLimitState(resp *http.Response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, parseErr := strconv.Atoi(remaining); parseErr == nil {
			c.remaining = val
		}
	}

	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if t, parseErr := time.Parse(time.RFC3339, reset); parseErr == nil {
			c.resetTime = t
		}
	}
}

// checkError checks for API errors in the response.
func (c *Client) checkError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return parseAPIError(resp, resp.Request.URL.Path)
}

// IsCloud returns true if connected to Jira Cloud.
func (c *Client) IsCloud() bool {
	return c.deploymentType == DeploymentCloud
}

// APIVersionInUse returns the API version being used.
func (c *Client) APIVersionInUse() APIVersion {
	return c.apiVersion
}

// RateLimitRemaining returns the remaining rate limit capacity.
// Returns -1 if unknown.
func (c *Client) RateLimitRemaining() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.remaining
}

// DeploymentTypeDetected returns the detected deployment type.
func (c *Client) DeploymentTypeDetected() DeploymentType {
	return c.deploymentType
}

// ServerInfoCached returns the cached server info, if available.
func (c *Client) ServerInfoCached() *ServerInfo {
	return c.serverInfo
}

// Context key type for storing Jira client in context.
type jiraClientKey struct{}

// ClientFromContext extracts a Jira Client from a context.
// Returns nil if no Client is present.
func ClientFromContext(ctx context.Context) *Client {
	if c, ok := ctx.Value(jiraClientKey{}).(*Client); ok {
		return c
	}
	return nil
}

// ContextWithClient adds a Jira Client to a context.
func ContextWithClient(ctx context.Context, c *Client) context.Context {
	return context.WithValue(ctx, jiraClientKey{}, c)
}

// cryptoRandFloat64 returns a cryptographically secure random float64 in [0.0, 1.0).
func cryptoRandFloat64() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to 0.5 if crypto/rand fails (shouldn't happen)
		return 0.5
	}
	// Convert to uint64 and normalize to [0, 1)
	u := binary.LittleEndian.Uint64(b[:])
	return float64(u>>11) / (1 << 53) * math.Nextafter(1, 2)
}

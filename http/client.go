package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 30 * time.Second

// DefaultMaxRetries is the default number of retry attempts.
const DefaultMaxRetries = 3

// DefaultRetryWait is the default initial wait between retries.
const DefaultRetryWait = 1 * time.Second

// Client provides common HTTP functionality for integration clients.
type Client struct {
	client      *http.Client
	baseURL     string
	serviceName string
	maxRetries  int
	retryWait   time.Duration

	// beforeRequest is called before each request (for auth headers, etc.)
	beforeRequest func(req *http.Request)
}

// ClientConfig holds configuration for Client.
type ClientConfig struct {
	Client        *http.Client
	BaseURL       string
	ServiceName   string
	MaxRetries    int
	RetryWait     time.Duration
	BeforeRequest func(req *http.Request)
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfg ClientConfig) *Client {
	c := &Client{
		client:        cfg.Client,
		baseURL:       cfg.BaseURL,
		serviceName:   cfg.ServiceName,
		maxRetries:    cfg.MaxRetries,
		retryWait:     cfg.RetryWait,
		beforeRequest: cfg.BeforeRequest,
	}

	if c.client == nil {
		c.client = &http.Client{Timeout: DefaultTimeout}
	}
	if c.maxRetries <= 0 {
		c.maxRetries = DefaultMaxRetries
	}
	if c.retryWait <= 0 {
		c.retryWait = DefaultRetryWait
	}

	return c
}

// Request executes an HTTP request with retries for transient errors.
func (c *Client) Request(ctx context.Context, method, path string, body any) (*http.Response, error) {
	return c.RequestWithHeaders(ctx, method, path, body, nil)
}

// RequestWithHeaders executes an HTTP request with custom headers.
func (c *Client) RequestWithHeaders(
	ctx context.Context,
	method, path string,
	body any,
	headers map[string]string,
) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path

	var lastErr error
	for attempt := range c.maxRetries {
		// Reset body reader for retry
		if body != nil {
			data, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(data)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		// Set default headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		// Apply custom headers
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		// Apply auth headers via callback
		if c.beforeRequest != nil {
			c.beforeRequest(req)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if shouldRetry(err, nil) && attempt < c.maxRetries-1 {
				wait := c.retryWait * time.Duration(1<<attempt) // Exponential backoff
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(wait):
					continue
				}
			}
			return nil, fmt.Errorf("%s request failed: %w", c.serviceName, err)
		}

		// Check for retryable status codes
		if shouldRetry(nil, resp) && attempt < c.maxRetries-1 {
			wait := c.getRetryWait(resp, attempt)
			resp.Body.Close()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
				continue
			}
		}

		return resp, nil
	}

	return nil, lastErr
}

// Get performs a GET request and decodes the response into result.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	resp, err := c.Request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, path, result)
}

// Post performs a POST request and decodes the response into result.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	resp, err := c.Request(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, path, result)
}

// Put performs a PUT request and decodes the response into result.
func (c *Client) Put(ctx context.Context, path string, body, result any) error {
	resp, err := c.Request(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, path, result)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.Request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, path, nil)
}

// handleResponse checks status and decodes the response body.
func (c *Client) handleResponse(resp *http.Response, path string, result any) error {
	if resp.StatusCode >= 400 {
		return c.parseError(resp, path)
	}

	if result == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode %s response: %w", c.serviceName, err)
	}

	return nil
}

// parseError parses an error response into an APIError.
func (c *Client) parseError(resp *http.Response, path string) error {
	body, _ := io.ReadAll(resp.Body)

	apiErr := &APIError{
		Service:    c.serviceName,
		StatusCode: resp.StatusCode,
		Endpoint:   path,
		RequestID:  resp.Header.Get("X-Request-Id"),
	}

	// Try to parse error message from body
	var errResp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil {
		if errResp.Message != "" {
			apiErr.Message = errResp.Message
		} else if errResp.Error != "" {
			apiErr.Message = errResp.Error
		}
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}

	return apiErr
}

// getRetryWait calculates the wait time for a retry.
func (c *Client) getRetryWait(resp *http.Response, attempt int) time.Duration {
	// Check for Retry-After header
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}

	// Exponential backoff
	return c.retryWait * time.Duration(1<<attempt)
}

// shouldRetry determines if a request should be retried.
func shouldRetry(err error, resp *http.Response) bool {
	if err != nil {
		// Retry on network errors
		return true
	}

	if resp != nil {
		// Retry on rate limit or server errors
		return resp.StatusCode == 429 || resp.StatusCode >= 500
	}

	return false
}

// GetRaw performs a GET request and returns the raw response body.
func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.Request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp, path)
	}

	return io.ReadAll(resp.Body)
}

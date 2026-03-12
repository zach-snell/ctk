package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client is the Confluence Cloud API HTTP client.
// It supports both classic API tokens (Basic Auth, direct site URL)
// and scoped/fine-grained tokens (Bearer Auth, Atlassian gateway URL).
type Client struct {
	http      *http.Client
	baseURL   string // base URL for API calls (varies by token type)
	email     string
	token     string // API token
	tokenType TokenType

	rateLimiter *RateLimiter
}

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a rate limiter with the given max requests per interval.
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available or returns an error.
func (r *RateLimiter) Wait() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed / r.refillRate)
	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = now
	}

	if r.tokens <= 0 {
		return fmt.Errorf("rate limit exceeded: max %d requests per %v, please wait before retrying", r.maxTokens, r.refillRate*time.Duration(r.maxTokens))
	}

	r.tokens--
	return nil
}

// NewClient creates a Confluence API client with classic auth (Basic Auth, direct site URL).
// Use NewClientFromCredentials for automatic token type detection.
func NewClient(domain, email, token string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   fmt.Sprintf("https://%s.atlassian.net", domain),
		email:     email,
		token:     token,
		tokenType: TokenTypeClassic,
		// 20 requests per minute = 1 token every 3 seconds
		rateLimiter: NewRateLimiter(20, 3*time.Second),
	}
}

// NewScopedClient creates a Confluence API client for scoped/fine-grained tokens.
// Uses Bearer auth against the Atlassian gateway URL.
func NewScopedClient(cloudID, email, token string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     fmt.Sprintf("https://api.atlassian.com/ex/confluence/%s", cloudID),
		email:       email,
		token:       token,
		tokenType:   TokenTypeScoped,
		rateLimiter: NewRateLimiter(20, 3*time.Second),
	}
}

// NewClientFromCredentials creates a client from stored credentials.
// Automatically selects classic or scoped auth based on the stored token type.
// If the type is not set, it probes both auth methods to auto-detect.
func NewClientFromCredentials(creds *Credentials) (*Client, error) {
	tokenType := creds.Type

	// Auto-detect if type is not set (env vars without explicit type, old credentials)
	if tokenType == "" {
		detected, cloudID, err := ProbeTokenType(creds.Domain, creds.Email, creds.APIToken)
		if err != nil {
			return nil, fmt.Errorf("auto-detecting token type: %w", err)
		}
		tokenType = detected
		if cloudID != "" {
			creds.CloudID = cloudID
		}
	}

	if tokenType == TokenTypeScoped {
		cloudID := creds.CloudID
		if cloudID == "" {
			var err error
			cloudID, err = FetchCloudID(creds.Domain)
			if err != nil {
				return nil, fmt.Errorf("scoped token requires cloud ID: %w", err)
			}
		}
		return NewScopedClient(cloudID, creds.Email, creds.APIToken), nil
	}

	return NewClient(creds.Domain, creds.Email, creds.APIToken), nil
}

// BaseURL returns the base URL of the client (for testing/debugging).
func (c *Client) BaseURL() string {
	return c.baseURL
}

// TokenType returns the token type this client is using.
func (c *Client) TokenType() TokenType {
	return c.tokenType
}

// do executes an HTTP request with auth headers and rate limiting.
func (c *Client) do(method, path string, bodyData []byte, contentType string) (*http.Response, error) {
	if err := c.rateLimiter.Wait(); err != nil {
		return nil, err
	}

	u := c.baseURL + path

	var bodyReader io.Reader
	if bodyData != nil {
		bodyReader = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Apply auth based on token type
	switch c.tokenType {
	case TokenTypeScoped:
		req.Header.Set("Authorization", "Bearer "+c.token)
	default:
		// Classic token uses Basic Auth (email:api_token)
		req.SetBasicAuth(c.email, c.token)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

// Get performs a GET request and returns the response body.
func (c *Client) Get(path string) ([]byte, error) {
	resp, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, data)
	}

	return data, nil
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyData = b
	}

	resp, err := c.do(http.MethodPost, path, bodyData, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyData = b
	}

	resp, err := c.do(http.MethodPut, path, bodyData, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) error {
	resp, err := c.do(http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp.StatusCode, data)
	}

	return nil
}

// GetJSON performs a GET and unmarshals the JSON response.
func GetJSON[T any](c *Client, path string) (*T, error) {
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &result, nil
}

// GetPaged performs a GET and unmarshals the cursor-based paginated response.
func GetPaged[T any](c *Client, path string) (*PagedResult[T], error) {
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result PagedResult[T]
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling paginated response: %w", err)
	}

	return &result, nil
}

func parseAPIError(statusCode int, body []byte) error {
	if statusCode == http.StatusForbidden {
		return fmt.Errorf("403 Forbidden: Permission denied. Ensure your Confluence API token has the required permissions. Additional details: %s", string(body))
	}
	if statusCode == http.StatusNotFound {
		return fmt.Errorf("404 Not Found: The requested resource was not found. Details: %s", string(body))
	}
	return fmt.Errorf("API error %d: %s", statusCode, string(body))
}

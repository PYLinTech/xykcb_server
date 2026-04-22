package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	client     *http.Client
	baseURL    string
	maxRetries int
	retryDelay time.Duration
}

type ClientOption func(*Client)

func WithMaxRetries(max int) ClientOption {
	return func(c *Client) {
		c.maxRetries = max
	}
}

func WithRetryDelay(delay time.Duration) ClientOption {
	return func(c *Client) {
		c.retryDelay = delay
	}
}

func NewClient(baseURL string, timeout time.Duration, opts ...ClientOption) *Client {
	c := &Client{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL:    baseURL,
		maxRetries: 3,
		retryDelay: time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) Get(path string, queryParams map[string]string) (map[string]interface{}, error) {
	return c.doRequest(http.MethodGet, path, queryParams, nil)
}

func (c *Client) Post(path string, body string) (map[string]interface{}, error) {
	return c.doRequest(http.MethodPost, path, nil, strings.NewReader(body))
}

func (c *Client) doRequest(method, path string, queryParams map[string]string, body io.Reader) (map[string]interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		result, err := c.tryRequest(method, path, queryParams, body)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !isRetryable(err) {
			break
		}
	}

	return nil, lastErr
}

func (c *Client) tryRequest(method, path string, queryParams map[string]string, body io.Reader) (map[string]interface{}, error) {
	reqURL := c.baseURL + path
	if queryParams != nil {
		q := url.Values{}
		for k, v := range queryParams {
			q.Add(k, v)
		}
		reqURL += "?" + q.Encode()
	}

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return data, nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "server error: 5") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout")
}

func ExtractTokenFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	token := u.Query().Get("token")
	return token
}

func ReplaceTokenInPath(path, newToken string) string {
	q, err := url.ParseQuery(path)
	if err != nil {
		return path
	}
	q.Set("token", newToken)
	parts := strings.SplitN(path, "?", 2)
	if len(parts) == 2 {
		return parts[0] + "?" + q.Encode()
	}
	return path + "?token=" + newToken
}

func ReplaceTokenInURL(rawURL, newToken string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set("token", newToken)
	u.RawQuery = q.Encode()
	return u.String()
}

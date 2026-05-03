package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func NewClient(baseURL string, timeout time.Duration) *Client {
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
	return c
}

func (c *Client) Get(path string) (map[string]interface{}, error) {
	return c.doRequest(http.MethodGet, path, nil)
}

func (c *Client) Post(path string, body string) (map[string]interface{}, error) {
	return c.doRequest(http.MethodPost, path, []byte(body))
}

func (c *Client) doRequest(method, path string, body []byte) (map[string]interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		result, err := c.tryRequest(method, path, body)
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

func (c *Client) tryRequest(method, path string, body []byte) (map[string]interface{}, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequest(method, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
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

func ReplaceTokenInPath(path, newToken string) string {
	parts := strings.SplitN(path, "?", 2)
	if len(parts) != 2 {
		return path + "?token=" + url.QueryEscape(newToken)
	}

	q, err := url.ParseQuery(parts[1])
	if err != nil {
		return path
	}
	q.Set("token", newToken)
	return parts[0] + "?" + q.Encode()
}

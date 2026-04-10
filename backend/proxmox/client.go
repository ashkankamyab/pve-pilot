package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	tokenID    string
	tokenSecret string
	httpClient *http.Client
}

func NewClient(baseURL, tokenID, tokenSecret string, insecureTLS bool) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureTLS,
		},
	}

	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		tokenID:     tokenID,
		tokenSecret: tokenSecret,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

func (c *Client) get(path string, result interface{}) error {
	url := fmt.Sprintf("%s/api2/json/%s", c.baseURL, strings.TrimLeft(path, "/"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Proxmox wraps all responses in {"data": ...}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if result != nil {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("unmarshaling data: %w", err)
		}
	}

	return nil
}

func (c *Client) post(path string, body interface{}) (string, error) {
	url := fmt.Sprintf("%s/api2/json/%s", c.baseURL, strings.TrimLeft(path, "/"))

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("marshaling body: %w", err)
		}
		reqBody = strings.NewReader(string(data))
	}

	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	c.setAuth(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Try to extract UPID from response
	var taskResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &taskResp); err == nil && taskResp.Data != "" {
		return taskResp.Data, nil
	}

	return string(respBody), nil
}

func (c *Client) postForm(path string, params map[string]string) (string, error) {
	endpoint := fmt.Sprintf("%s/api2/json/%s", c.baseURL, strings.TrimLeft(path, "/"))

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	c.setAuth(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &taskResp); err == nil && taskResp.Data != "" {
		return taskResp.Data, nil
	}

	return string(respBody), nil
}

func (c *Client) putForm(path string, params map[string]string) (string, error) {
	endpoint := fmt.Sprintf("%s/api2/json/%s", c.baseURL, strings.TrimLeft(path, "/"))

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest("PUT", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	c.setAuth(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &taskResp); err == nil && taskResp.Data != "" {
		return taskResp.Data, nil
	}

	return string(respBody), nil
}

func (c *Client) delete(path string) (string, error) {
	url := fmt.Sprintf("%s/api2/json/%s", c.baseURL, strings.TrimLeft(path, "/"))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &taskResp); err == nil && taskResp.Data != "" {
		return taskResp.Data, nil
	}

	return string(respBody), nil
}

func (c *Client) setAuth(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.tokenSecret))
}

// Ping tests connectivity to the Proxmox API
func (c *Client) Ping() error {
	var version map[string]interface{}
	return c.get("version", &version)
}

// GetTaskStatus returns the status of an async Proxmox task.
func (c *Client) GetTaskStatus(node, upid string) (*TaskStatus, error) {
	var status TaskStatus
	err := c.get(fmt.Sprintf("nodes/%s/tasks/%s/status", node, url.PathEscape(upid)), &status)
	return &status, err
}

// TaskLogLine represents one line from a Proxmox task log.
type taskLogLine struct {
	N int    `json:"n"`
	T string `json:"t"`
}

// GetTaskLog returns the log output of a Proxmox task.
func (c *Client) GetTaskLog(node, upid string) ([]string, error) {
	var lines []taskLogLine
	err := c.get(fmt.Sprintf("nodes/%s/tasks/%s/log", node, url.PathEscape(upid)), &lines)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.T)
	}
	return out, nil
}

// summarizeTaskError inspects the task log to produce a human-friendly error message.
// Common Proxmox errors: out of space, storage full, already exists, etc.
func (c *Client) summarizeTaskError(node, upid, exitStatus string) string {
	logs, err := c.GetTaskLog(node, upid)
	if err != nil || len(logs) == 0 {
		return exitStatus
	}

	// Look for common error patterns in the tail of the log (most recent lines first).
	// Proxmox typically prints the actual error near the end before "TASK ERROR".
	for i := len(logs) - 1; i >= 0; i-- {
		line := strings.TrimSpace(logs[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "out of space") ||
			strings.Contains(lower, "no space left") ||
			strings.Contains(lower, "not enough space") ||
			strings.Contains(lower, "insufficient") ||
			strings.Contains(lower, "failed") ||
			strings.Contains(lower, "error") ||
			strings.Contains(lower, "unable to") {
			// Strip "TASK ERROR: " prefix if present
			line = strings.TrimPrefix(line, "TASK ERROR: ")
			return line
		}
	}

	// Fallback: return last non-empty line
	for i := len(logs) - 1; i >= 0; i-- {
		line := strings.TrimSpace(logs[i])
		if line != "" {
			return line
		}
	}
	return exitStatus
}

// WaitForTask polls until a Proxmox task completes, returns nil on success.
// On failure, it fetches the task log for a detailed error message.
func (c *Client) WaitForTask(node, upid string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := c.GetTaskStatus(node, upid)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if status.Status == "stopped" {
			if status.ExitStatus == "OK" || strings.HasPrefix(status.ExitStatus, "WARNINGS:") {
				return nil
			}
			detail := c.summarizeTaskError(node, upid, status.ExitStatus)
			return fmt.Errorf("%s", detail)
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for task %s", upid)
}

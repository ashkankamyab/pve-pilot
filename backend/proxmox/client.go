package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	url := fmt.Sprintf("%s/api2/json/%s", c.baseURL, strings.TrimLeft(path, "/"))

	form := make([]string, 0, len(params))
	for k, v := range params {
		form = append(form, fmt.Sprintf("%s=%s", k, v))
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(strings.Join(form, "&")))
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
	url := fmt.Sprintf("%s/api2/json/%s", c.baseURL, strings.TrimLeft(path, "/"))

	form := make([]string, 0, len(params))
	for k, v := range params {
		form = append(form, fmt.Sprintf("%s=%s", k, v))
	}

	req, err := http.NewRequest("PUT", url, strings.NewReader(strings.Join(form, "&")))
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

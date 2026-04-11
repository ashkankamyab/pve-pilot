package proxmox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestServer creates an httptest.Server that responds to Proxmox API requests.
// The handler map keys are paths (without /api2/json/ prefix), values are response bodies.
func newTestServer(t *testing.T, handlers map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip /api2/json/ prefix
		path := r.URL.Path
		if len(path) > 10 && path[:10] == "/api2/json" {
			path = path[11:]
		}

		for pattern, resp := range handlers {
			if path == pattern {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"data": resp})
				return
			}
		}
		http.NotFound(w, r)
	}))
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return NewClient(server.URL, "test@pve!token", "secret", true)
}

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.com:8006", "user@pve!token", "secret", true)
	if c.baseURL != "https://example.com:8006" {
		t.Errorf("unexpected baseURL: %s", c.baseURL)
	}
	if c.tokenID != "user@pve!token" {
		t.Errorf("unexpected tokenID: %s", c.tokenID)
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://example.com:8006/", "user@pve!token", "secret", false)
	if c.baseURL != "https://example.com:8006" {
		t.Errorf("expected trailing slash trimmed, got: %s", c.baseURL)
	}
}

func TestPing(t *testing.T) {
	server := newTestServer(t, map[string]interface{}{
		"version": map[string]string{"version": "8.0", "release": "8.0-1"},
	})
	defer server.Close()

	c := newTestClient(t, server)
	if err := c.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestPingFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":{"username":"invalid credentials"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	if err := c.Ping(); err == nil {
		t.Fatal("expected Ping to fail with 401")
	}
}

func TestSetAuth(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]string{}})
	}))
	defer server.Close()

	c := NewClient(server.URL, "user@pve!mytoken", "abc123", true)
	c.Ping()

	expected := "PVEAPIToken=user@pve!mytoken=abc123"
	if gotAuth != expected {
		t.Errorf("expected auth header %q, got %q", expected, gotAuth)
	}
}

func TestGetAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	var result map[string]interface{}
	err := c.get("some/path", &result)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetTaskStatus(t *testing.T) {
	server := newTestServer(t, map[string]interface{}{
		"nodes/pve/tasks/UPID:pve:001/status": TaskStatus{
			Status:     "stopped",
			ExitStatus: "OK",
			Type:       "qmclone",
			Node:       "pve",
		},
	})
	defer server.Close()

	c := newTestClient(t, server)
	status, err := c.GetTaskStatus("pve", "UPID:pve:001")
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	if status.Status != "stopped" {
		t.Errorf("expected stopped, got %s", status.Status)
	}
	if status.ExitStatus != "OK" {
		t.Errorf("expected OK, got %s", status.ExitStatus)
	}
}

func TestWaitForTaskSuccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		status := TaskStatus{Node: "pve", Type: "qmclone"}
		if callCount <= 2 {
			status.Status = "running"
		} else {
			status.Status = "stopped"
			status.ExitStatus = "OK"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"data": status})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.WaitForTask("pve", "UPID:pve:001", 30*time.Second)
	if err != nil {
		t.Fatalf("WaitForTask failed: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls (2 running + 1 stopped), got %d", callCount)
	}
}

func TestWaitForTaskWarnings(t *testing.T) {
	server := newTestServer(t, map[string]interface{}{
		"nodes/pve/tasks/UPID:pve:002/status": TaskStatus{
			Status:     "stopped",
			ExitStatus: "WARNINGS: 2",
		},
	})
	defer server.Close()

	c := newTestClient(t, server)
	err := c.WaitForTask("pve", "UPID:pve:002", 5*time.Second)
	if err != nil {
		t.Fatalf("WaitForTask should succeed on WARNINGS, got: %v", err)
	}
}

func TestWaitForTaskTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": TaskStatus{Status: "running"},
		})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.WaitForTask("pve", "UPID:pve:003", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if err.Error() != "timeout waiting for task UPID:pve:003" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSummarizeTaskError(t *testing.T) {
	tests := []struct {
		name     string
		logs     []string
		fallback string
		want     string
	}{
		{
			name:     "error keyword in log",
			logs:     []string{"starting backup", "ERROR: no space left on device", "TASK ERROR"},
			fallback: "unknown",
			want:     "TASK ERROR",
		},
		{
			name:     "failed keyword",
			logs:     []string{"trying to create disk", "failed to allocate disk space"},
			fallback: "unknown",
			want:     "failed to allocate disk space",
		},
		{
			name:     "last non-empty line fallback",
			logs:     []string{"some info", "another line", ""},
			fallback: "unknown",
			want:     "another line",
		},
		{
			name:     "empty logs",
			logs:     []string{},
			fallback: "unknown exit",
			want:     "unknown exit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build log response
			logLines := make([]map[string]interface{}, len(tt.logs))
			for i, line := range tt.logs {
				logLines[i] = map[string]interface{}{"n": i + 1, "t": line}
			}

			server := newTestServer(t, map[string]interface{}{
				"nodes/pve/tasks/UPID:test/log": logLines,
			})
			defer server.Close()

			c := newTestClient(t, server)
			got := c.summarizeTaskError("pve", "UPID:test", tt.fallback)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestPostReturnsUPID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:12345"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	upid, err := c.post("nodes/pve/qemu/100/status/start", nil)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if upid != "UPID:pve:12345" {
		t.Errorf("expected UPID:pve:12345, got %s", upid)
	}
}

func TestPostFormSendsFormEncoded(t *testing.T) {
	var gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:99"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.postForm("nodes/pve/qemu/100/config", map[string]string{"cores": "4"})
	if err != nil {
		t.Fatalf("postForm failed: %v", err)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("expected form content type, got %s", gotContentType)
	}
}

func TestDeleteReturnsUPID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:del001"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	upid, err := c.delete("nodes/pve/qemu/100")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if upid != "UPID:pve:del001" {
		t.Errorf("expected UPID:pve:del001, got %s", upid)
	}
}

func TestConnectionRefused(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "test@pve!token", "secret", true)
	err := c.Ping()
	if err == nil {
		t.Fatal("expected connection refused error")
	}
}

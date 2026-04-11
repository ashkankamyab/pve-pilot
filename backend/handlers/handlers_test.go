package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashkankamyab/pve-pilot/jobs"
	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupMockPVE creates a test Proxmox API server and configures the global PVE client.
// The handler map keys are URL paths (after /api2/json/), values are JSON-serializable responses.
// Returns the mock server (caller must defer server.Close()).
func setupMockPVE(t *testing.T, routes map[string]mockRoute) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 10 && path[:10] == "/api2/json" {
			path = path[11:]
		}

		for pattern, route := range routes {
			if path == pattern && (route.method == "" || route.method == r.Method) {
				if route.status != 0 {
					w.WriteHeader(route.status)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"data": route.response})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"errors": "not found"})
	}))

	PVE = proxmox.NewClient(server.URL, "test@pve!token", "secret", true)
	return server
}

type mockRoute struct {
	method   string // empty matches any method
	response interface{}
	status   int // 0 means 200
}

// setupJobStore initializes the global JobStore for tests.
func setupJobStore(t *testing.T) {
	t.Helper()
	JobStore = jobs.NewStore()
}

// performRequest sends an HTTP request to the Gin router and returns the response recorder.
func performRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// parseJSON is a helper to decode response body.
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response JSON: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

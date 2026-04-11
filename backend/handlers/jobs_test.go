package handlers

import (
	"net/http"
	"testing"

	"github.com/ashkankamyab/pve-pilot/jobs"
	"github.com/gin-gonic/gin"
)

func TestListJobs(t *testing.T) {
	setupJobStore(t)
	JobStore.Create(&jobs.Job{ID: "j1", Type: "vm", Name: "vm1", Status: jobs.StatusCompleted})
	JobStore.Create(&jobs.Job{ID: "j2", Type: "container", Name: "ct1", Status: jobs.StatusRunning})

	r := gin.New()
	r.GET("/api/jobs", ListJobs)

	w := performRequest(r, "GET", "/api/jobs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetJob(t *testing.T) {
	setupJobStore(t)
	JobStore.Create(&jobs.Job{ID: "j1", Type: "vm", Name: "test-vm", Status: jobs.StatusRunning})

	r := gin.New()
	r.GET("/api/jobs/:id", GetJob)

	w := performRequest(r, "GET", "/api/jobs/j1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := parseJSON(t, w)
	if body["name"] != "test-vm" {
		t.Errorf("expected name test-vm, got %v", body["name"])
	}
}

func TestGetJobNotFound(t *testing.T) {
	setupJobStore(t)

	r := gin.New()
	r.GET("/api/jobs/:id", GetJob)

	w := performRequest(r, "GET", "/api/jobs/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestStreamJobEventsNotFound(t *testing.T) {
	setupJobStore(t)

	r := gin.New()
	r.GET("/api/jobs/:id/events", StreamJobEvents)

	w := performRequest(r, "GET", "/api/jobs/nonexistent/events", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestStreamJobEventsTerminalState(t *testing.T) {
	setupJobStore(t)
	JobStore.Create(&jobs.Job{ID: "j1", Status: jobs.StatusCompleted, Step: jobs.StepReady, Progress: 100})

	r := gin.New()
	r.GET("/api/jobs/:id/events", StreamJobEvents)

	w := performRequest(r, "GET", "/api/jobs/j1/events", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Should have SSE content type
	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}

	// Body should contain at least one SSE data line
	if w.Body.Len() == 0 {
		t.Error("expected SSE data in response body")
	}
}

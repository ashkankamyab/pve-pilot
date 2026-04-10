package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/ashkankamyab/pve-pilot/jobs"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

var (
	JobStore *jobs.Store
	NatsConn *nats.Conn
)

// ListJobs returns all tracked jobs.
func ListJobs(c *gin.Context) {
	all := JobStore.List()
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	c.JSON(http.StatusOK, all)
}

// GetJob returns a single job by ID.
func GetJob(c *gin.Context) {
	id := c.Param("id")
	job := JobStore.Get(id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// StreamJobEvents sends SSE events for a job until it completes or client disconnects.
func StreamJobEvents(c *gin.Context) {
	id := c.Param("id")
	job := JobStore.Get(id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // disable nginx/proxy buffering
	c.Writer.Flush()

	// Send current state immediately
	initialEvent := jobs.JobEvent{
		JobID:    job.ID,
		Status:   job.Status,
		Step:     job.Step,
		Progress: job.Progress,
		Error:    job.Error,
		IP:       job.IPAddress,
	}
	writeSSE(c.Writer, initialEvent)
	c.Writer.Flush()

	// If already terminal, close
	if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusFailed {
		return
	}

	// Subscribe to updates
	ch := JobStore.Subscribe(id)
	defer JobStore.Unsubscribe(id, ch)

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(c.Writer, event)
			c.Writer.Flush()
			if event.Status == jobs.StatusCompleted || event.Status == jobs.StatusFailed {
				return
			}
		}
	}
}

func writeSSE(w io.Writer, event jobs.JobEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// submitProvisionJob creates a job, stores it, and publishes to NATS.
func submitProvisionJob(jobType string, sourceNode string, sourceVMID int, targetNode string, req *jobs.ProvisionPayload) (string, error) {
	id := uuid.New().String()

	job := &jobs.Job{
		ID:         id,
		Type:       jobType,
		Status:     jobs.StatusPending,
		Step:       StepNone,
		Progress:   0,
		SourceNode: sourceNode,
		SourceVMID: sourceVMID,
		TargetNode: targetNode,
		NewVMID:    req.NewVMID,
		Name:       req.Name,
		Storage:      req.Storage,
		CIUser:       req.CIUser,
		Cores:        req.Cores,
		Memory:       req.Memory,
		Password:     req.Password,
		SSHKeys:      req.SSHKeys,
		DiskSize:     req.DiskSize,
		ExtraVolumes: req.ExtraVolumes,
		UserData:     req.UserData,
		DNSDomain:    req.DNSDomain,
		FullClone:    req.FullClone,
	}
	JobStore.Create(job)

	req.JobID = id
	req.Type = jobType
	req.SourceNode = sourceNode
	req.SourceVMID = sourceVMID
	req.TargetNode = targetNode

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshaling payload: %w", err)
	}

	if err := NatsConn.Publish(jobs.SubjectProvision, data); err != nil {
		JobStore.SetError(id, fmt.Sprintf("failed to queue job: %v", err))
		return "", fmt.Errorf("publishing to NATS: %w", err)
	}

	return id, nil
}

// StepNone is used for initial pending state before any step runs.
const StepNone jobs.StepName = ""

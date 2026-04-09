package jobs

import (
	"time"

	"github.com/ashkankamyab/pve-pilot/proxmox"
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

type StepName string

const (
	StepCloning     StepName = "cloning"
	StepConfiguring StepName = "configuring"
	StepResizing    StepName = "resizing"
	StepAddingDisks StepName = "adding_disks"
	StepStarting    StepName = "starting"
	StepWaitingRun  StepName = "waiting_for_running"
	StepReady       StepName = "ready"
)

var StepProgress = map[StepName]int{
	StepCloning:     10,
	StepConfiguring: 30,
	StepResizing:    45,
	StepAddingDisks: 55,
	StepStarting:    70,
	StepWaitingRun:  85,
	StepReady:       100,
}

type Job struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    JobStatus `json:"status"`
	Step      StepName  `json:"step"`
	Progress  int       `json:"progress"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	SourceNode   string                `json:"source_node"`
	SourceVMID   int                   `json:"source_vmid"`
	TargetNode   string                `json:"target_node"`
	NewVMID      int                   `json:"new_vmid"`
	Name         string                `json:"name"`
	Storage      string                `json:"storage,omitempty"`
	CIUser       string                `json:"ciuser,omitempty"`
	Password     string                `json:"-"`
	SSHKeys      string                `json:"-"`
	DiskSize     int                   `json:"disk_size,omitempty"`
	ExtraVolumes []proxmox.ExtraVolume `json:"extra_volumes,omitempty"`
	UserData     string                `json:"-"`
	DNSDomain    string                `json:"dns_domain,omitempty"`
	FullClone    bool                  `json:"full_clone"`

	IPAddress string `json:"ip_address,omitempty"`
}

type ProvisionPayload struct {
	JobID        string                `json:"job_id"`
	Type         string                `json:"type"`
	SourceNode   string                `json:"source_node"`
	SourceVMID   int                   `json:"source_vmid"`
	TargetNode   string                `json:"target_node"`
	NewVMID      int                   `json:"new_vmid"`
	Name         string                `json:"name"`
	Storage      string                `json:"storage,omitempty"`
	CIUser       string                `json:"ciuser,omitempty"`
	Password     string                `json:"password,omitempty"`
	SSHKeys      string                `json:"sshkeys,omitempty"`
	DiskSize     int                   `json:"disk_size,omitempty"`
	ExtraVolumes []proxmox.ExtraVolume `json:"extra_volumes,omitempty"`
	UserData     string                `json:"user_data,omitempty"`
	DNSDomain    string                `json:"dns_domain,omitempty"`
	FullClone    bool                  `json:"full_clone"`
}

type JobEvent struct {
	JobID    string    `json:"job_id"`
	Status   JobStatus `json:"status"`
	Step     StepName  `json:"step"`
	Progress int       `json:"progress"`
	Error    string    `json:"error,omitempty"`
	IP       string    `json:"ip_address,omitempty"`
}

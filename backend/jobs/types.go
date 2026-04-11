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
	// Provision steps
	StepCloning     StepName = "cloning"
	StepConfiguring StepName = "configuring"
	StepResizing    StepName = "resizing"
	StepAddingDisks StepName = "adding_disks"
	StepStarting    StepName = "starting"
	StepWaitingRun  StepName = "waiting_for_running"
	StepReady       StepName = "ready"

	// Backup steps
	StepBackingUp StepName = "backing_up"

	// Restore steps
	StepStopping  StepName = "stopping"
	StepDeleting  StepName = "deleting"
	StepRestoring StepName = "restoring"
)

var StepProgress = map[StepName]int{
	// Provision
	StepCloning:     10,
	StepConfiguring: 30,
	StepResizing:    45,
	StepAddingDisks: 55,
	StepStarting:    70,
	StepWaitingRun:  85,
	StepReady:       100,

	// Backup
	StepBackingUp: 50,

	// Restore
	StepStopping:  15,
	StepDeleting:  30,
	StepRestoring: 60,
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
	Cores        int                   `json:"cores,omitempty"`
	Memory       int                   `json:"memory,omitempty"`
	Password     string                `json:"-"`
	SSHKeys      string                `json:"-"`
	DiskSize     int                   `json:"disk_size,omitempty"`
	ExtraVolumes []proxmox.ExtraVolume `json:"extra_volumes,omitempty"`
	UserData     string                `json:"-"`
	DNSDomain    string                `json:"dns_domain,omitempty"`
	IPMode       string                `json:"ip_mode,omitempty"`
	StaticIP     string                `json:"static_ip,omitempty"`
	StaticGW     string                `json:"static_gw,omitempty"`
	StaticSubnet int                   `json:"static_subnet,omitempty"`
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
	Cores        int                   `json:"cores,omitempty"`
	Memory       int                   `json:"memory,omitempty"`
	Password     string                `json:"password,omitempty"`
	SSHKeys      string                `json:"sshkeys,omitempty"`
	DiskSize     int                   `json:"disk_size,omitempty"`
	ExtraVolumes []proxmox.ExtraVolume `json:"extra_volumes,omitempty"`
	UserData     string                `json:"user_data,omitempty"`
	DNSDomain    string                `json:"dns_domain,omitempty"`
	IPMode       string                `json:"ip_mode,omitempty"`
	StaticIP     string                `json:"static_ip,omitempty"`
	StaticGW     string                `json:"static_gw,omitempty"`
	StaticSubnet int                   `json:"static_subnet,omitempty"`
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

// BackupPayload is the NATS message for backup jobs.
type BackupPayload struct {
	JobID   string `json:"job_id"`
	Type    string `json:"type"` // "vm" or "container"
	Node    string `json:"node"`
	VMID    int    `json:"vmid"`
	Name    string `json:"name"`
	Storage string `json:"storage"`
	Notes   string `json:"notes,omitempty"`
}

// RestorePayload is the NATS message for restore jobs.
type RestorePayload struct {
	JobID   string `json:"job_id"`
	Type    string `json:"type"` // "vm" or "container"
	Node    string `json:"node"`
	VMID    int    `json:"vmid"`
	Name    string `json:"name"`
	Archive string `json:"archive"`
	Storage string `json:"storage"`
	InPlace bool   `json:"in_place"`
}

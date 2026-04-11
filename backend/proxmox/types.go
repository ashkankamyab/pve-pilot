package proxmox

// APIResponse wraps all Proxmox API responses
type APIResponse struct {
	Data interface{} `json:"data"`
}

// ClusterResource represents a resource from GET /cluster/resources
type ClusterResource struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Node       string  `json:"node"`
	VMID       int     `json:"vmid,omitempty"`
	Name       string  `json:"name,omitempty"`
	Status     string  `json:"status"`
	CPU        float64 `json:"cpu,omitempty"`
	MaxCPU     int     `json:"maxcpu,omitempty"`
	Mem        int64   `json:"mem,omitempty"`
	MaxMem     int64   `json:"maxmem,omitempty"`
	Disk       int64   `json:"disk,omitempty"`
	MaxDisk    int64   `json:"maxdisk,omitempty"`
	Uptime     int64   `json:"uptime,omitempty"`
	Template   int     `json:"template,omitempty"`
	Pool       string  `json:"pool,omitempty"`
	Storage    string  `json:"storage,omitempty"`
	Content    string  `json:"content,omitempty"`
	PluginType string  `json:"plugintype,omitempty"`
}

// NodeStatus represents detailed node information
type NodeStatus struct {
	CPU        float64    `json:"cpu"`
	Memory     MemInfo    `json:"memory"`
	RootFS     DiskInfo   `json:"rootfs"`
	Swap       MemInfo    `json:"swap"`
	Uptime     int64      `json:"uptime"`
	KVersion   string     `json:"kversion"`
	PVEVersion string     `json:"pveversion"`
	CPUInfo    CPUInfo    `json:"cpuinfo"`
	LoadAvg    []string   `json:"loadavg"`
}

type MemInfo struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
	Free  int64 `json:"free"`
}

type DiskInfo struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
	Free  int64 `json:"free"`
	Avail int64 `json:"avail"`
}

type CPUInfo struct {
	Cores   int    `json:"cores"`
	CPUs    int    `json:"cpus"`
	MHz     string `json:"mhz"`
	Model   string `json:"model"`
	Sockets int    `json:"sockets"`
	Threads int    `json:"threads"`
}

// VMStatus represents QEMU VM status
type VMStatus struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	CPU       float64 `json:"cpu"`
	CPUs      int     `json:"cpus"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	DiskRead  int64   `json:"diskread"`
	DiskWrite int64   `json:"diskwrite"`
	Uptime    int64   `json:"uptime"`
	PID       int     `json:"pid,omitempty"`
	Template  int     `json:"template,omitempty"`
}

// ContainerStatus represents LXC container status
type ContainerStatus struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Type      string  `json:"type"`
	CPU       float64 `json:"cpu"`
	CPUs      int     `json:"cpus"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	Uptime    int64   `json:"uptime"`
	Template  int     `json:"template,omitempty"`
}

// StorageInfo represents storage pool information
type StorageInfo struct {
	Storage string `json:"storage"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Total   int64  `json:"total"`
	Used    int64  `json:"used"`
	Avail   int64  `json:"avail"`
	Active  int    `json:"active"`
	Enabled int    `json:"enabled"`
	Shared  int    `json:"shared"`
}

// CloneRequest represents a clone operation
type CloneRequest struct {
	NewID  int    `json:"newid" binding:"required"`
	Name   string `json:"name"`
	Target string `json:"target,omitempty"`
	Full   *bool  `json:"full,omitempty"`
}

// TaskResponse represents an async task
type TaskResponse struct {
	UPID string `json:"data"`
}

// TaskStatus represents the status of a Proxmox async task
type TaskStatus struct {
	Status     string `json:"status"` // "running", "stopped"
	ExitStatus string `json:"exitstatus,omitempty"` // "OK" on success, error message on failure
	Type       string `json:"type"`
	ID         string `json:"id"`
	Node       string `json:"node"`
	PID        int    `json:"pid"`
}

// ClusterSummary is an aggregated overview
type ClusterSummary struct {
	Nodes              int     `json:"nodes"`
	NodesOnline        int     `json:"nodes_online"`
	VMsRunning         int     `json:"vms_running"`
	VMsTotal           int     `json:"vms_total"`
	ContainersRunning  int     `json:"containers_running"`
	ContainersTotal    int     `json:"containers_total"`
	CPUUsage           float64 `json:"cpu_usage"`
	CPUTotal           int     `json:"cpu_total"`
	MemUsed            int64   `json:"mem_used"`
	MemTotal           int64   `json:"mem_total"`
	DiskUsed           int64   `json:"disk_used"`
	DiskTotal          int64   `json:"disk_total"`
}

// TemplateInfo extends ClusterResource with type info
type TemplateInfo struct {
	ClusterResource
	VMType string `json:"vmtype"` // "qemu" or "lxc"
}

// ExtraVolume represents an additional disk to attach
type ExtraVolume struct {
	Storage string `json:"storage"`
	SizeGB  int    `json:"size_gb"`
}

// ProvisionRequest extends CloneRequest with cloud-init and disk options
type ProvisionRequest struct {
	NewID        int           `json:"newid" binding:"required"`
	Name         string        `json:"name" binding:"required"`
	Target       string        `json:"target,omitempty"`
	Storage      string        `json:"storage,omitempty"`
	Full         *bool         `json:"full,omitempty"`
	CIUser       string        `json:"ciuser,omitempty"`
	Password     string        `json:"password,omitempty"`
	SSHKeys      string        `json:"sshkeys,omitempty"`
	Cores        int           `json:"cores,omitempty"`
	Memory       int           `json:"memory,omitempty"` // in MB
	DiskSize     int           `json:"disk_size,omitempty"` // in GB
	ExtraVolumes []ExtraVolume `json:"extra_volumes,omitempty"`
	UserData     string        `json:"user_data,omitempty"`
	IPMode       string        `json:"ip_mode,omitempty"`  // "dhcp" or "static"
	IP           string        `json:"ip,omitempty"`       // e.g. "192.168.2.100"
	Gateway      string        `json:"gateway,omitempty"`  // e.g. "192.168.2.1"
	Subnet       int           `json:"subnet,omitempty"`   // CIDR e.g. 24
}

// ScaleRequest sets cores and/or memory on a VM or container
type ScaleRequest struct {
	Cores    int `json:"cores"`
	MemoryMB int `json:"memory"`
}

// ResizeDiskRequest grows a disk (grow only, never shrink)
type ResizeDiskRequest struct {
	Disk string `json:"disk"` // e.g. "scsi0", "rootfs", "mp0"
	Size string `json:"size"` // e.g. "50G"
}

// AddDiskRequest attaches a new disk to a QEMU VM
type AddDiskRequest struct {
	Storage string `json:"storage"`
	SizeGB  int    `json:"size_gb"`
}

// AddVolumeRequest attaches a new mountpoint to an LXC container
type AddVolumeRequest struct {
	Storage   string `json:"storage"`
	SizeGB    int    `json:"size_gb"`
	MountPath string `json:"mount_path"` // e.g. "/mnt/data"
}

// NetworkInterface represents a network interface from qemu-guest-agent
type NetworkInterface struct {
	Name        string      `json:"name"`
	HWAddr      string      `json:"hardware-address"`
	IPAddresses []IPAddress `json:"ip-addresses"`
}

// IPAddress represents an IP address on an interface
type IPAddress struct {
	Type    string `json:"ip-address-type"`
	Address string `json:"ip-address"`
	Prefix  int    `json:"prefix"`
}

// FilesystemInfo represents filesystem usage from the guest agent
type FilesystemInfo struct {
	Name       string              `json:"name"`
	MountPoint string              `json:"mountpoint"`
	Type       string              `json:"type"`
	TotalBytes int64               `json:"total-bytes,omitempty"`
	UsedBytes  int64               `json:"used-bytes,omitempty"`
	Disk       []FilesystemDiskRef `json:"disk,omitempty"`
}

// FilesystemDiskRef references the backing device
type FilesystemDiskRef struct {
	Dev string `json:"dev,omitempty"`
}

// BackupInfo represents a backup volume on a Proxmox storage
type BackupInfo struct {
	VolID   string `json:"volid"`
	Size    int64  `json:"size"`
	CTime   int64  `json:"ctime"`
	Notes   string `json:"notes,omitempty"`
	VMID    int    `json:"vmid,omitempty"`
	Format  string `json:"format"`
	Content string `json:"content"`
}

// BackupSchedule represents a Proxmox cluster backup schedule
type BackupSchedule struct {
	ID        string `json:"id"`
	Type      string `json:"type,omitempty"`
	VMID      string `json:"vmid,omitempty"`
	Storage   string `json:"storage"`
	Schedule  string `json:"schedule"`
	Enabled   int    `json:"enabled"`
	Comment   string `json:"comment,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Compress  string `json:"compress,omitempty"`
	Node      string `json:"node,omitempty"`
	MailTo    string `json:"mailto,omitempty"`
}

// BackupScheduleRequest is used to create a new backup schedule
type BackupScheduleRequest struct {
	VMID     string `json:"vmid"`
	Storage  string `json:"storage"`
	Schedule string `json:"schedule"`
	Mode     string `json:"mode"`
	Compress string `json:"compress"`
	Comment  string `json:"comment"`
	Enabled  bool   `json:"enabled"`
	Node     string `json:"node,omitempty"`
}

// RestoreRequest is used to restore a VM or container from a backup
type RestoreRequest struct {
	Archive string `json:"archive" binding:"required"`
	VMID    int    `json:"vmid"`
	Storage string `json:"storage"`
	InPlace bool   `json:"in_place"`
}

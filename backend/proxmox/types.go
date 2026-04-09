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
	DiskSize     int           `json:"disk_size,omitempty"` // in GB
	ExtraVolumes []ExtraVolume `json:"extra_volumes,omitempty"`
	UserData     string        `json:"user_data,omitempty"` // cloud-init user-data script
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

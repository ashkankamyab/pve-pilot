package proxmox

import "time"

// API defines the interface for all Proxmox VE operations.
// *Client satisfies this interface. Use it in handlers and workers
// so they can be tested with mock implementations.
type API interface {
	// Connection
	Ping() error

	// Tasks
	GetTaskStatus(node, upid string) (*TaskStatus, error)
	GetTaskLog(node, upid string) ([]string, error)
	WaitForTask(node, upid string, timeout time.Duration) error

	// Cluster
	GetClusterResources() ([]ClusterResource, error)
	GetClusterSummary() (*ClusterSummary, error)
	ListNodes() ([]ClusterResource, error)
	GetNodeStatus(node string) (map[string]interface{}, error)

	// Storage
	ListStorage(node string) ([]StorageInfo, error)

	// QEMU VMs
	ListVMs(node string) ([]VMStatus, error)
	GetVMStatus(node string, vmid int) (*VMStatus, error)
	GetVMConfig(node string, vmid int) (map[string]interface{}, error)
	StartVM(node string, vmid int) (string, error)
	StopVM(node string, vmid int) (string, error)
	RebootVM(node string, vmid int) (string, error)
	DeleteVM(node string, vmid int) (string, error)
	CloneVM(node string, vmid int, newID int, name, target, storage string, full bool) (string, error)
	ConfigureCloudInit(node string, vmid int, ciuser, password, sshkeys, searchdomain, ipMode, ip, gateway string, subnet int) error
	SetVMResources(node string, vmid int, cores, memoryMB int) error
	ResizeDisk(node string, vmid int, disk string, size string) error
	AddDisk(node string, vmid int, bus string, storage string, sizeGB int) error
	GetVMInterfaces(node string, vmid int) ([]NetworkInterface, error)
	GetVMFilesystems(node string, vmid int) ([]FilesystemInfo, error)
	GuestExec(node string, vmid int, script string) error
	GuestExecWithRetry(node string, vmid int, script string, timeout time.Duration) error

	// LXC Containers
	ListContainers(node string) ([]ContainerStatus, error)
	GetContainerStatus(node string, vmid int) (*ContainerStatus, error)
	GetContainerConfig(node string, vmid int) (map[string]interface{}, error)
	GetContainerInterfaces(node string, vmid int) ([]map[string]interface{}, error)
	StartContainer(node string, vmid int) (string, error)
	StopContainer(node string, vmid int) (string, error)
	RebootContainer(node string, vmid int) (string, error)
	DeleteContainer(node string, vmid int) (string, error)
	CloneContainer(node string, vmid int, newID int, hostname, target, storage string, full bool) (string, error)
	SetContainerResources(node string, vmid int, cores, memoryMB int) error
	ConfigureContainerHostname(node string, vmid int, hostname string) error
	ConfigureContainerNetwork(node string, vmid int, ip, gateway string, subnet int) error
	ResizeContainerDisk(node string, vmid int, disk string, size string) (string, error)
	AddContainerMountPoint(node string, vmid int, mpKey, storage string, sizeGB int, mountPath string) error

	// Backups
	Backup(node string, vmid int, storage, mode, compress, notes string) (string, error)
	ListBackups(node, storage string) ([]BackupInfo, error)
	DeleteBackup(node, volid string) (string, error)
	RestoreVM(node, archive string, vmid int, storage string) (string, error)
	RestoreContainer(node, archive string, vmid int, storage string) (string, error)
	ListBackupSchedules() ([]BackupSchedule, error)
	CreateBackupSchedule(req BackupScheduleRequest) (string, error)
	DeleteBackupSchedule(id string) error
}

// HostExecutor abstracts SSH-based host commands (pct exec for LXC).
// *HostSSH satisfies this interface.
type HostExecutor interface {
	IsEnabled() bool
	PctExec(vmid int, script string) error
}

// Compile-time checks that concrete types satisfy interfaces.
var _ API = (*Client)(nil)
var _ HostExecutor = (*HostSSH)(nil)

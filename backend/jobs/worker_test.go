package jobs

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ashkankamyab/pve-pilot/proxmox"
)

// mockPVE is a test double for proxmox.API.
// Only methods used by the worker are implemented; others panic if called.
type mockPVE struct {
	mu sync.Mutex

	// Track calls for assertions
	calls []string

	// Configurable return values
	cloneVMErr           error
	cloneContainerErr    error
	waitForTaskErr       error
	startVMErr           error
	startContainerErr    error
	vmStatuses           map[int]*proxmox.VMStatus // vmid -> status (changes over time)
	containerStatuses    map[int]*proxmox.ContainerStatus
	vmInterfaces         []proxmox.NetworkInterface
	backupErr            error
	restoreVMErr         error
	restoreContainerErr  error
	deleteVMErr          error
	deleteContainerErr   error
	stopVMErr            error
	stopContainerErr     error
	guestExecErr         error
}

func newMockPVE() *mockPVE {
	return &mockPVE{
		vmStatuses:        make(map[int]*proxmox.VMStatus),
		containerStatuses: make(map[int]*proxmox.ContainerStatus),
	}
}

func (m *mockPVE) record(call string) {
	m.mu.Lock()
	m.calls = append(m.calls, call)
	m.mu.Unlock()
}

func (m *mockPVE) getCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]string, len(m.calls))
	copy(cp, m.calls)
	return cp
}

func (m *mockPVE) hasCalled(name string) bool {
	for _, c := range m.getCalls() {
		if c == name {
			return true
		}
	}
	return false
}

// VM operations
func (m *mockPVE) CloneVM(node string, vmid int, newID int, name, target, storage string, full bool) (string, error) {
	m.record("CloneVM")
	return "UPID:pve:clone", m.cloneVMErr
}

func (m *mockPVE) WaitForTask(node, upid string, timeout time.Duration) error {
	m.record("WaitForTask")
	return m.waitForTaskErr
}

func (m *mockPVE) ConfigureCloudInit(node string, vmid int, ciuser, password, sshkeys, searchdomain, ipMode, ip, gateway string, subnet int) error {
	m.record("ConfigureCloudInit")
	return nil
}

func (m *mockPVE) SetVMResources(node string, vmid int, cores, memoryMB int) error {
	m.record("SetVMResources")
	return nil
}

func (m *mockPVE) ResizeDisk(node string, vmid int, disk string, size string) error {
	m.record("ResizeDisk")
	return nil
}

func (m *mockPVE) AddDisk(node string, vmid int, bus string, storage string, sizeGB int) error {
	m.record("AddDisk")
	return nil
}

func (m *mockPVE) StartVM(node string, vmid int) (string, error) {
	m.record("StartVM")
	if m.startVMErr != nil {
		return "", m.startVMErr
	}
	// Simulate VM becoming running after start
	m.mu.Lock()
	if s, ok := m.vmStatuses[vmid]; ok {
		s.Status = "running"
	}
	m.mu.Unlock()
	return "UPID:pve:start", nil
}

func (m *mockPVE) GetVMStatus(node string, vmid int) (*proxmox.VMStatus, error) {
	m.record("GetVMStatus")
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.vmStatuses[vmid]; ok {
		cp := *s
		return &cp, nil
	}
	return &proxmox.VMStatus{VMID: vmid, Status: "stopped"}, nil
}

func (m *mockPVE) GetVMInterfaces(node string, vmid int) ([]proxmox.NetworkInterface, error) {
	m.record("GetVMInterfaces")
	if m.vmInterfaces != nil {
		return m.vmInterfaces, nil
	}
	return nil, fmt.Errorf("QEMU guest agent not running")
}

func (m *mockPVE) GuestExec(node string, vmid int, script string) error {
	m.record("GuestExec")
	return m.guestExecErr
}

func (m *mockPVE) GuestExecWithRetry(node string, vmid int, script string, timeout time.Duration) error {
	m.record("GuestExecWithRetry")
	return m.guestExecErr
}

// Container operations
func (m *mockPVE) CloneContainer(node string, vmid int, newID int, hostname, target, storage string, full bool) (string, error) {
	m.record("CloneContainer")
	return "UPID:pve:clone-ct", m.cloneContainerErr
}

func (m *mockPVE) ConfigureContainerHostname(node string, vmid int, hostname string) error {
	m.record("ConfigureContainerHostname")
	return nil
}

func (m *mockPVE) ConfigureContainerNetwork(node string, vmid int, ip, gateway string, subnet int) error {
	m.record("ConfigureContainerNetwork")
	return nil
}

func (m *mockPVE) SetContainerResources(node string, vmid int, cores, memoryMB int) error {
	m.record("SetContainerResources")
	return nil
}

func (m *mockPVE) ResizeContainerDisk(node string, vmid int, disk string, size string) (string, error) {
	m.record("ResizeContainerDisk")
	return "UPID:pve:resize-ct", nil
}

func (m *mockPVE) StartContainer(node string, vmid int) (string, error) {
	m.record("StartContainer")
	if m.startContainerErr != nil {
		return "", m.startContainerErr
	}
	m.mu.Lock()
	if s, ok := m.containerStatuses[vmid]; ok {
		s.Status = "running"
	}
	m.mu.Unlock()
	return "UPID:pve:start-ct", nil
}

func (m *mockPVE) GetContainerStatus(node string, vmid int) (*proxmox.ContainerStatus, error) {
	m.record("GetContainerStatus")
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.containerStatuses[vmid]; ok {
		cp := *s
		return &cp, nil
	}
	return &proxmox.ContainerStatus{VMID: vmid, Status: "stopped"}, nil
}

func (m *mockPVE) StopVM(node string, vmid int) (string, error) {
	m.record("StopVM")
	if m.stopVMErr != nil {
		return "", m.stopVMErr
	}
	m.mu.Lock()
	if s, ok := m.vmStatuses[vmid]; ok {
		s.Status = "stopped"
	}
	m.mu.Unlock()
	return "UPID:pve:stop", nil
}

func (m *mockPVE) StopContainer(node string, vmid int) (string, error) {
	m.record("StopContainer")
	if m.stopContainerErr != nil {
		return "", m.stopContainerErr
	}
	m.mu.Lock()
	if s, ok := m.containerStatuses[vmid]; ok {
		s.Status = "stopped"
	}
	m.mu.Unlock()
	return "UPID:pve:stop-ct", nil
}

func (m *mockPVE) DeleteVM(node string, vmid int) (string, error) {
	m.record("DeleteVM")
	return "UPID:pve:del", m.deleteVMErr
}

func (m *mockPVE) DeleteContainer(node string, vmid int) (string, error) {
	m.record("DeleteContainer")
	return "UPID:pve:del-ct", m.deleteContainerErr
}

// Backup/Restore
func (m *mockPVE) Backup(node string, vmid int, storage, mode, compress, notes string) (string, error) {
	m.record("Backup")
	return "UPID:pve:backup", m.backupErr
}

func (m *mockPVE) RestoreVM(node, archive string, vmid int, storage string) (string, error) {
	m.record("RestoreVM")
	return "UPID:pve:restore", m.restoreVMErr
}

func (m *mockPVE) RestoreContainer(node, archive string, vmid int, storage string) (string, error) {
	m.record("RestoreContainer")
	return "UPID:pve:restore-ct", m.restoreContainerErr
}

// Unused by worker — stub implementations
func (m *mockPVE) Ping() error                                           { return nil }
func (m *mockPVE) GetTaskStatus(node, upid string) (*proxmox.TaskStatus, error) {
	return &proxmox.TaskStatus{Status: "stopped", ExitStatus: "OK"}, nil
}
func (m *mockPVE) GetTaskLog(node, upid string) ([]string, error)        { return nil, nil }
func (m *mockPVE) GetClusterResources() ([]proxmox.ClusterResource, error) { return nil, nil }
func (m *mockPVE) GetClusterSummary() (*proxmox.ClusterSummary, error)   { return nil, nil }
func (m *mockPVE) ListNodes() ([]proxmox.ClusterResource, error)         { return nil, nil }
func (m *mockPVE) GetNodeStatus(node string) (map[string]interface{}, error) { return nil, nil }
func (m *mockPVE) ListStorage(node string) ([]proxmox.StorageInfo, error) { return nil, nil }
func (m *mockPVE) ListVMs(node string) ([]proxmox.VMStatus, error)       { return nil, nil }
func (m *mockPVE) GetVMConfig(node string, vmid int) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockPVE) RebootVM(node string, vmid int) (string, error)        { return "", nil }
func (m *mockPVE) GetVMFilesystems(node string, vmid int) ([]proxmox.FilesystemInfo, error) {
	return nil, nil
}
func (m *mockPVE) ListContainers(node string) ([]proxmox.ContainerStatus, error) { return nil, nil }
func (m *mockPVE) GetContainerConfig(node string, vmid int) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockPVE) GetContainerInterfaces(node string, vmid int) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockPVE) RebootContainer(node string, vmid int) (string, error) { return "", nil }
func (m *mockPVE) AddContainerMountPoint(node string, vmid int, mpKey, storage string, sizeGB int, mountPath string) error {
	return nil
}
func (m *mockPVE) ListBackups(node, storage string) ([]proxmox.BackupInfo, error) { return nil, nil }
func (m *mockPVE) DeleteBackup(node, volid string) (string, error)       { return "", nil }
func (m *mockPVE) ListBackupSchedules() ([]proxmox.BackupSchedule, error) { return nil, nil }
func (m *mockPVE) CreateBackupSchedule(req proxmox.BackupScheduleRequest) (string, error) {
	return "", nil
}
func (m *mockPVE) DeleteBackupSchedule(id string) error { return nil }

// mockHostSSH is a test double for proxmox.HostExecutor.
type mockHostSSH struct {
	enabled    bool
	pctExecErr error
	calls      []string
	mu         sync.Mutex
}

func (m *mockHostSSH) IsEnabled() bool { return m.enabled }

func (m *mockHostSSH) PctExec(vmid int, script string) error {
	m.mu.Lock()
	m.calls = append(m.calls, fmt.Sprintf("PctExec(%d)", vmid))
	m.mu.Unlock()
	return m.pctExecErr
}

// --- Tests ---

func TestProvisionVMHappyPath(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.vmStatuses[10001] = &proxmox.VMStatus{VMID: 10001, Status: "stopped"}
	pve.vmInterfaces = []proxmox.NetworkInterface{
		{Name: "eth0", IPAddresses: []proxmox.IPAddress{{Type: "ipv4", Address: "192.168.2.50"}}},
	}

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-provision-vm"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	payload := ProvisionPayload{
		JobID:      jobID,
		Type:       "vm",
		SourceNode: "pve",
		SourceVMID: 9000,
		TargetNode: "pve",
		NewVMID:    10001,
		Name:       "test-vm",
		Storage:    "local-lvm",
		CIUser:     "ubuntu",
		Password:   "secret",
		SSHKeys:    "ssh-ed25519 AAAA",
		Cores:      4,
		Memory:     2048,
		DiskSize:   30,
		ExtraVolumes: []proxmox.ExtraVolume{
			{Storage: "local-lvm", SizeGB: 50},
		},
		DNSDomain: "ashkmb.com",
		FullClone: true,
	}

	w.provisionVM(payload)

	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}
	if job.Step != StepReady {
		t.Errorf("expected step ready, got %s", job.Step)
	}
	if job.Progress != 100 {
		t.Errorf("expected progress 100, got %d", job.Progress)
	}
	if job.IPAddress != "192.168.2.50" {
		t.Errorf("expected IP 192.168.2.50, got %s", job.IPAddress)
	}

	// Verify call order
	calls := pve.getCalls()
	expectedCalls := []string{"CloneVM", "WaitForTask", "ConfigureCloudInit", "SetVMResources", "ResizeDisk", "AddDisk", "StartVM", "WaitForTask"}
	for _, expected := range expectedCalls {
		if !pve.hasCalled(expected) {
			t.Errorf("expected call to %s, calls were: %v", expected, calls)
		}
	}
}

func TestProvisionVMCloneFails(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.cloneVMErr = fmt.Errorf("storage full")

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-clone-fail"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.provisionVM(ProvisionPayload{
		JobID:      jobID,
		SourceNode: "pve",
		SourceVMID: 9000,
		NewVMID:    10001,
		Name:       "fail-vm",
	})

	job := store.Get(jobID)
	if job.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", job.Status)
	}
	if job.Error == "" {
		t.Error("expected error message")
	}
}

func TestProvisionVMStartFails(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.startVMErr = fmt.Errorf("not enough resources")

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-start-fail"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.provisionVM(ProvisionPayload{
		JobID:      jobID,
		SourceNode: "pve",
		SourceVMID: 9000,
		NewVMID:    10001,
		Name:       "fail-vm",
	})

	job := store.Get(jobID)
	if job.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", job.Status)
	}
}

func TestProvisionVMWithUserData(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.vmStatuses[10001] = &proxmox.VMStatus{VMID: 10001, Status: "stopped"}
	pve.vmInterfaces = []proxmox.NetworkInterface{
		{Name: "eth0", IPAddresses: []proxmox.IPAddress{{Type: "ipv4", Address: "10.0.0.1"}}},
	}

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-userdata"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.provisionVM(ProvisionPayload{
		JobID:      jobID,
		SourceNode: "pve",
		SourceVMID: 9000,
		NewVMID:    10001,
		Name:       "ud-vm",
		UserData:   "#!/bin/bash\napt update",
	})

	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}
	if !pve.hasCalled("GuestExecWithRetry") {
		t.Error("expected GuestExecWithRetry for user-data")
	}
}

func TestWaitForRunningAndIPNoAgent(t *testing.T) {
	pve := newMockPVE()
	pve.vmStatuses[10001] = &proxmox.VMStatus{VMID: 10001, Status: "running"}
	// No interfaces → guest agent not available

	w := &Worker{pve: pve}

	// Use a short timeout so the test doesn't block
	running, ip := w.waitForRunningAndIP("pve", 10001, 500*time.Millisecond)
	if !running {
		t.Error("expected running=true")
	}
	if ip != "" {
		t.Errorf("expected empty IP, got %s", ip)
	}
}

func TestWaitForRunningAndIPVMNeverStarts(t *testing.T) {
	pve := newMockPVE()
	pve.vmStatuses[10001] = &proxmox.VMStatus{VMID: 10001, Status: "stopped"}

	w := &Worker{pve: pve}

	running, ip := w.waitForRunningAndIP("pve", 10001, 500*time.Millisecond)
	if running {
		t.Error("expected running=false for stopped VM")
	}
	if ip != "" {
		t.Errorf("expected empty IP, got %s", ip)
	}
}

func TestProvisionContainerHappyPath(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.containerStatuses[10002] = &proxmox.ContainerStatus{VMID: 10002, Status: "stopped"}

	hostSSH := &mockHostSSH{enabled: true}

	w := &Worker{store: store, pve: pve, hostSSH: hostSSH}

	jobID := "test-provision-ct"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.provisionContainer(ProvisionPayload{
		JobID:      jobID,
		Type:       "container",
		SourceNode: "pve",
		SourceVMID: 8000,
		NewVMID:    10002,
		Name:       "test-ct",
		Cores:      2,
		Memory:     512,
		DiskSize:   5,
		Password:   "rootpass",
		SSHKeys:    "ssh-ed25519 AAAA",
	})

	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}

	if !pve.hasCalled("CloneContainer") {
		t.Error("expected CloneContainer call")
	}
	if !pve.hasCalled("ConfigureContainerHostname") {
		t.Error("expected ConfigureContainerHostname call")
	}
	if !pve.hasCalled("SetContainerResources") {
		t.Error("expected SetContainerResources call")
	}
	// Should inject credentials via SSH
	if len(hostSSH.calls) == 0 {
		t.Error("expected PctExec call for credential injection")
	}
}

func TestProvisionContainerCloneFails(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.cloneContainerErr = fmt.Errorf("template locked")

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-ct-clone-fail"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.provisionContainer(ProvisionPayload{
		JobID:      jobID,
		SourceNode: "pve",
		SourceVMID: 8000,
		NewVMID:    10002,
		Name:       "fail-ct",
	})

	job := store.Get(jobID)
	if job.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", job.Status)
	}
}

func TestProvisionContainerNoSSH(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.containerStatuses[10002] = &proxmox.ContainerStatus{VMID: 10002, Status: "stopped"}

	hostSSH := &mockHostSSH{enabled: false} // SSH disabled

	w := &Worker{store: store, pve: pve, hostSSH: hostSSH}

	jobID := "test-ct-no-ssh"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.provisionContainer(ProvisionPayload{
		JobID:      jobID,
		SourceNode: "pve",
		SourceVMID: 8000,
		NewVMID:    10002,
		Name:       "nossh-ct",
		Password:   "secret",
	})

	// Should still complete — SSH is optional
	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}
	// PctExec should NOT be called
	if len(hostSSH.calls) != 0 {
		t.Errorf("expected no PctExec calls when SSH disabled, got %v", hostSSH.calls)
	}
}

func TestBackupJobHappyPath(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-backup"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.backupJob(BackupPayload{
		JobID:   jobID,
		Type:    "vm",
		Node:    "pve",
		VMID:    100,
		Name:    "web-server",
		Storage: "nfs-drive",
		Notes:   "pre-upgrade backup",
	})

	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}
	if !pve.hasCalled("Backup") {
		t.Error("expected Backup call")
	}
}

func TestBackupJobFails(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.backupErr = fmt.Errorf("storage offline")

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-backup-fail"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.backupJob(BackupPayload{
		JobID:   jobID,
		Type:    "vm",
		Node:    "pve",
		VMID:    100,
		Storage: "nfs-drive",
	})

	job := store.Get(jobID)
	if job.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", job.Status)
	}
}

func TestRestoreVMHappyPath(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-restore-vm"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.restoreJob(RestorePayload{
		JobID:   jobID,
		Type:    "vm",
		Node:    "pve",
		VMID:    10005,
		Archive: "nfs-drive:dump/vzdump-qemu-100.vma.zst",
		Storage: "local-lvm",
		InPlace: false,
	})

	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}
	if !pve.hasCalled("RestoreVM") {
		t.Error("expected RestoreVM call")
	}
	// Should NOT stop/delete since InPlace=false
	if pve.hasCalled("StopVM") {
		t.Error("should not stop VM for non-in-place restore")
	}
}

func TestRestoreVMInPlace(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.vmStatuses[100] = &proxmox.VMStatus{VMID: 100, Status: "running"}

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-restore-inplace"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.restoreJob(RestorePayload{
		JobID:   jobID,
		Type:    "vm",
		Node:    "pve",
		VMID:    100,
		Archive: "nfs-drive:dump/vzdump-qemu-100.vma.zst",
		Storage: "local-lvm",
		InPlace: true,
	})

	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}
	if !pve.hasCalled("StopVM") {
		t.Error("expected StopVM for in-place restore")
	}
	if !pve.hasCalled("DeleteVM") {
		t.Error("expected DeleteVM for in-place restore")
	}
	if !pve.hasCalled("RestoreVM") {
		t.Error("expected RestoreVM")
	}
}

func TestRestoreContainerInPlace(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.containerStatuses[200] = &proxmox.ContainerStatus{VMID: 200, Status: "running"}

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-restore-ct-inplace"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.restoreJob(RestorePayload{
		JobID:   jobID,
		Type:    "container",
		Node:    "pve",
		VMID:    200,
		Archive: "nfs-drive:dump/vzdump-lxc-200.tar.zst",
		Storage: "local-lvm",
		InPlace: true,
	})

	job := store.Get(jobID)
	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", job.Status, job.Error)
	}
	if !pve.hasCalled("StopContainer") {
		t.Error("expected StopContainer for in-place restore")
	}
	if !pve.hasCalled("DeleteContainer") {
		t.Error("expected DeleteContainer for in-place restore")
	}
	if !pve.hasCalled("RestoreContainer") {
		t.Error("expected RestoreContainer")
	}
}

func TestRestoreDeleteFails(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.vmStatuses[100] = &proxmox.VMStatus{VMID: 100, Status: "stopped"}
	pve.deleteVMErr = fmt.Errorf("VM locked")

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	jobID := "test-restore-del-fail"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	w.restoreJob(RestorePayload{
		JobID:   jobID,
		Type:    "vm",
		Node:    "pve",
		VMID:    100,
		Archive: "nfs-drive:dump/vzdump-qemu-100.vma.zst",
		InPlace: true,
	})

	job := store.Get(jobID)
	if job.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", job.Status)
	}
}

func TestProvisionVMStepProgression(t *testing.T) {
	store := NewStore()
	pve := newMockPVE()
	pve.vmStatuses[10001] = &proxmox.VMStatus{VMID: 10001, Status: "stopped"}
	pve.vmInterfaces = []proxmox.NetworkInterface{
		{Name: "eth0", IPAddresses: []proxmox.IPAddress{{Type: "ipv4", Address: "10.0.0.1"}}},
	}

	jobID := "test-steps"
	store.Create(&Job{ID: jobID, Status: StatusPending})

	// Subscribe to track step progression
	ch := store.Subscribe(jobID)
	defer store.Unsubscribe(jobID, ch)

	w := &Worker{store: store, pve: pve, hostSSH: &mockHostSSH{}}

	// Run in goroutine since we're collecting events
	done := make(chan struct{})
	go func() {
		w.provisionVM(ProvisionPayload{
			JobID:      jobID,
			SourceNode: "pve",
			SourceVMID: 9000,
			NewVMID:    10001,
			Name:       "step-vm",
			DiskSize:   30,
		})
		close(done)
	}()

	var steps []StepName
	for {
		select {
		case event := <-ch:
			steps = append(steps, event.Step)
			if event.Status == StatusCompleted || event.Status == StatusFailed {
				goto verify
			}
		case <-done:
			// Drain remaining events
			for len(ch) > 0 {
				event := <-ch
				steps = append(steps, event.Step)
			}
			goto verify
		}
	}

verify:
	// Should see: cloning, configuring, resizing, starting, waiting_for_running, ready
	if len(steps) < 4 {
		t.Errorf("expected at least 4 step transitions, got %d: %v", len(steps), steps)
	}
	// First step should be cloning
	if steps[0] != StepCloning {
		t.Errorf("expected first step to be cloning, got %s", steps[0])
	}
	// Last step should be ready
	if steps[len(steps)-1] != StepReady {
		t.Errorf("expected last step to be ready, got %s", steps[len(steps)-1])
	}
}

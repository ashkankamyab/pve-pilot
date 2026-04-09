package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/nats-io/nats.go"
)

const SubjectProvision = "jobs.provision"

// Worker subscribes to NATS and executes provision jobs.
type Worker struct {
	nc    *nats.Conn
	store *Store
	pve   *proxmox.Client
}

func NewWorker(nc *nats.Conn, store *Store, pve *proxmox.Client) *Worker {
	return &Worker{nc: nc, store: store, pve: pve}
}

// Start subscribes to the provision subject and processes jobs.
func (w *Worker) Start() error {
	_, err := w.nc.Subscribe(SubjectProvision, func(msg *nats.Msg) {
		var payload ProvisionPayload
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("worker: invalid payload: %v", err)
			return
		}
		log.Printf("worker: picked up job %s (type=%s, vmid=%d)", payload.JobID, payload.Type, payload.NewVMID)

		if payload.Type == "container" {
			w.provisionContainer(payload)
		} else {
			w.provisionVM(payload)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribing to %s: %w", SubjectProvision, err)
	}
	log.Printf("worker: listening on %s", SubjectProvision)
	return nil
}

func (w *Worker) provisionVM(p ProvisionPayload) {
	id := p.JobID

	targetNode := p.SourceNode
	if p.TargetNode != "" {
		targetNode = p.TargetNode
	}

	// Step 1: Clone — returns UPID, the actual clone runs async in Proxmox
	w.store.UpdateStep(id, StepCloning, StatusRunning)
	log.Printf("worker [%s]: cloning VM %d -> %d on %s", id, p.SourceVMID, p.NewVMID, targetNode)
	upid, err := w.pve.CloneVM(p.SourceNode, p.SourceVMID, p.NewVMID, p.Name, p.TargetNode, p.Storage, p.FullClone)
	if err != nil {
		w.store.SetError(id, fmt.Sprintf("clone failed: %v", err))
		return
	}

	// Wait for the actual Proxmox clone TASK to complete (not just VM existence)
	log.Printf("worker [%s]: waiting for clone task %s", id, upid)
	if err := w.pve.WaitForTask(p.SourceNode, upid, 300*time.Second); err != nil {
		w.store.SetError(id, fmt.Sprintf("clone task failed: %v", err))
		return
	}
	log.Printf("worker [%s]: clone task complete for VM %d", id, p.NewVMID)

	// Step 2: Configure cloud-init
	w.store.UpdateStep(id, StepConfiguring, StatusRunning)
	if p.CIUser != "" || p.Password != "" || p.SSHKeys != "" || p.DNSDomain != "" {
		log.Printf("worker [%s]: configuring cloud-init for VM %d (user=%s, domain=%s)", id, p.NewVMID, p.CIUser, p.DNSDomain)
		if err := w.pve.ConfigureCloudInit(targetNode, p.NewVMID, p.CIUser, p.Password, p.SSHKeys, p.DNSDomain); err != nil {
			log.Printf("worker [%s]: cloud-init config failed for VM %d: %v", id, p.NewVMID, err)
		}
	}

	// User-data will be executed via guest agent after VM boots (see below)

	// Step 3: Resize disk
	w.store.UpdateStep(id, StepResizing, StatusRunning)
	if p.DiskSize > 0 {
		sizeStr := fmt.Sprintf("%dG", p.DiskSize)
		log.Printf("worker [%s]: resizing disk to %s for VM %d", id, sizeStr, p.NewVMID)
		if err := w.pve.ResizeDisk(targetNode, p.NewVMID, "scsi0", sizeStr); err != nil {
			if err2 := w.pve.ResizeDisk(targetNode, p.NewVMID, "virtio0", sizeStr); err2 != nil {
				log.Printf("worker [%s]: disk resize failed for VM %d: scsi0: %v, virtio0: %v", id, p.NewVMID, err, err2)
			}
		}
	}

	// Step 4: Add extra volumes
	if len(p.ExtraVolumes) > 0 {
		w.store.UpdateStep(id, StepAddingDisks, StatusRunning)
		for i, vol := range p.ExtraVolumes {
			bus := fmt.Sprintf("scsi%d", i+1) // scsi1, scsi2, etc.
			log.Printf("worker [%s]: adding disk %s (%s, %dGB) to VM %d", id, bus, vol.Storage, vol.SizeGB, p.NewVMID)
			if err := w.pve.AddDisk(targetNode, p.NewVMID, bus, vol.Storage, vol.SizeGB); err != nil {
				log.Printf("worker [%s]: failed to add disk %s: %v", id, bus, err)
				// non-fatal, continue
			}
		}
	}

	// Step 5: Start VM — also wait for start task to complete
	w.store.UpdateStep(id, StepStarting, StatusRunning)
	log.Printf("worker [%s]: starting VM %d", id, p.NewVMID)
	startUpid, err := w.pve.StartVM(targetNode, p.NewVMID)
	if err != nil {
		w.store.SetError(id, fmt.Sprintf("start failed: %v", err))
		return
	}

	// Wait for the start task itself to finish on Proxmox
	log.Printf("worker [%s]: waiting for start task %s", id, startUpid)
	if err := w.pve.WaitForTask(targetNode, startUpid, 60*time.Second); err != nil {
		w.store.SetError(id, fmt.Sprintf("start task failed: %v", err))
		return
	}
	log.Printf("worker [%s]: start task complete for VM %d", id, p.NewVMID)

	// Step 5: Wait for VM to be running and get IP
	w.store.UpdateStep(id, StepWaitingRun, StatusRunning)
	running, ip := w.waitForRunningAndIP(targetNode, p.NewVMID, 180*time.Second)

	if !running {
		w.store.SetError(id, "VM failed to reach running state within timeout")
		log.Printf("worker [%s]: VM %d never reached running state", id, p.NewVMID)
		return
	}

	if ip != "" {
		w.store.SetIP(id, ip)
	}

	// Execute user-data script via guest agent if provided
	if p.UserData != "" && running {
		log.Printf("worker [%s]: executing user-data script on VM %d via guest agent", id, p.NewVMID)
		if err := w.pve.GuestExecWithRetry(targetNode, p.NewVMID, p.UserData, 60*time.Second); err != nil {
			log.Printf("worker [%s]: failed to execute user-data script: %v", id, err)
		} else {
			log.Printf("worker [%s]: user-data script executed on VM %d", id, p.NewVMID)
		}
	}

	// Done
	w.store.UpdateStep(id, StepReady, StatusCompleted)
	log.Printf("worker [%s]: job completed (VM %d on %s, ip=%s)", id, p.NewVMID, targetNode, ip)
}

func (w *Worker) provisionContainer(p ProvisionPayload) {
	id := p.JobID

	targetNode := p.SourceNode
	if p.TargetNode != "" {
		targetNode = p.TargetNode
	}

	// Step 1: Clone
	w.store.UpdateStep(id, StepCloning, StatusRunning)
	log.Printf("worker [%s]: cloning container %d -> %d", id, p.SourceVMID, p.NewVMID)
	upid, err := w.pve.CloneContainer(p.SourceNode, p.SourceVMID, p.NewVMID, p.Name, p.TargetNode, p.Storage, p.FullClone)
	if err != nil {
		w.store.SetError(id, fmt.Sprintf("clone failed: %v", err))
		return
	}

	// Wait for clone task to complete
	if err := w.pve.WaitForTask(p.SourceNode, upid, 300*time.Second); err != nil {
		w.store.SetError(id, fmt.Sprintf("clone task failed: %v", err))
		return
	}
	log.Printf("worker [%s]: clone task complete for container %d", id, p.NewVMID)

	// Step 2: Configure
	w.store.UpdateStep(id, StepConfiguring, StatusRunning)
	if p.Password != "" || p.SSHKeys != "" {
		if err := w.pve.ConfigureContainerCloudInit(targetNode, p.NewVMID, p.Password, p.SSHKeys); err != nil {
			log.Printf("worker [%s]: container config failed for %d: %v", id, p.NewVMID, err)
		}
	}

	// Step 3: Resize disk
	w.store.UpdateStep(id, StepResizing, StatusRunning)
	if p.DiskSize > 0 {
		sizeStr := fmt.Sprintf("%dG", p.DiskSize)
		if err := w.pve.ResizeContainerDisk(targetNode, p.NewVMID, "rootfs", sizeStr); err != nil {
			log.Printf("worker [%s]: disk resize failed for container %d: %v", id, p.NewVMID, err)
		}
	}

	// Step 4: Start container
	w.store.UpdateStep(id, StepStarting, StatusRunning)
	log.Printf("worker [%s]: starting container %d", id, p.NewVMID)
	startUpid, err := w.pve.StartContainer(targetNode, p.NewVMID)
	if err != nil {
		w.store.SetError(id, fmt.Sprintf("start failed: %v", err))
		return
	}

	if err := w.pve.WaitForTask(targetNode, startUpid, 60*time.Second); err != nil {
		w.store.SetError(id, fmt.Sprintf("start task failed: %v", err))
		return
	}

	// Step 5: Wait for running
	w.store.UpdateStep(id, StepWaitingRun, StatusRunning)
	if !w.waitForRunningContainer(targetNode, p.NewVMID, 120*time.Second) {
		w.store.SetError(id, "container failed to reach running state within timeout")
		return
	}

	// Done
	w.store.UpdateStep(id, StepReady, StatusCompleted)
	log.Printf("worker [%s]: job completed (container %d on %s)", id, p.NewVMID, targetNode)
}

// waitForRunningAndIP polls until the VM is running, then tries to get IP.
func (w *Worker) waitForRunningAndIP(node string, vmid int, timeout time.Duration) (bool, string) {
	deadline := time.Now().Add(timeout)
	vmRunning := false

	for time.Now().Before(deadline) {
		status, err := w.pve.GetVMStatus(node, vmid)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if status.Status == "running" {
			if !vmRunning {
				vmRunning = true
				log.Printf("worker: VM %d is now running, waiting for IP...", vmid)
			}

			// Try to get IP from guest agent
			interfaces, err := w.pve.GetVMInterfaces(node, vmid)
			if err == nil {
				for _, iface := range interfaces {
					if iface.Name == "lo" {
						continue
					}
					for _, addr := range iface.IPAddresses {
						if addr.Type == "ipv4" && addr.Address != "127.0.0.1" {
							return true, addr.Address
						}
					}
				}
			}
			// Running but no IP yet — guest agent may not be ready
			time.Sleep(3 * time.Second)
			continue
		}

		time.Sleep(2 * time.Second)
	}

	if vmRunning {
		return true, ""
	}
	return false, ""
}

// waitForRunningContainer polls until the container is running.
func (w *Worker) waitForRunningContainer(node string, vmid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := w.pve.GetContainerStatus(node, vmid)
		if err == nil && status != nil && status.Status == "running" {
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

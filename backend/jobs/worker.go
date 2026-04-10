package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/nats-io/nats.go"
)

const SubjectProvision = "jobs.provision"

// Worker subscribes to NATS and executes provision jobs.
type Worker struct {
	nc      *nats.Conn
	store   *Store
	pve     *proxmox.Client
	hostSSH *proxmox.HostSSH
}

func NewWorker(nc *nats.Conn, store *Store, pve *proxmox.Client, hostSSH *proxmox.HostSSH) *Worker {
	return &Worker{nc: nc, store: store, pve: pve, hostSSH: hostSSH}
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

	// Apply cores/memory overrides from the form
	if p.Cores > 0 || p.Memory > 0 {
		log.Printf("worker [%s]: setting VM %d resources cores=%d memory=%dMB", id, p.NewVMID, p.Cores, p.Memory)
		if err := w.pve.SetVMResources(targetNode, p.NewVMID, p.Cores, p.Memory); err != nil {
			log.Printf("worker [%s]: failed to set VM resources: %v", id, err)
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

	// Step 2: Configure hostname via API
	w.store.UpdateStep(id, StepConfiguring, StatusRunning)
	log.Printf("worker [%s]: configuring container %d hostname=%s", id, p.NewVMID, p.Name)
	if err := w.pve.ConfigureContainerHostname(targetNode, p.NewVMID, p.Name); err != nil {
		log.Printf("worker [%s]: container hostname config failed: %v", id, err)
	}
	if p.Cores > 0 || p.Memory > 0 {
		log.Printf("worker [%s]: setting container %d resources cores=%d memory=%dMB", id, p.NewVMID, p.Cores, p.Memory)
		if err := w.pve.SetContainerResources(targetNode, p.NewVMID, p.Cores, p.Memory); err != nil {
			log.Printf("worker [%s]: failed to set container resources: %v", id, err)
		}
	}
	time.Sleep(2 * time.Second)

	// Step 3: Resize disk
	w.store.UpdateStep(id, StepResizing, StatusRunning)
	if p.DiskSize > 0 {
		sizeStr := fmt.Sprintf("%dG", p.DiskSize)
		log.Printf("worker [%s]: resizing container %d disk to %s", id, p.NewVMID, sizeStr)
		resizeUpid, err := w.pve.ResizeContainerDisk(targetNode, p.NewVMID, "rootfs", sizeStr)
		if err != nil {
			log.Printf("worker [%s]: disk resize failed for container %d: %v", id, p.NewVMID, err)
		} else if resizeUpid != "" {
			if err := w.pve.WaitForTask(targetNode, resizeUpid, 120*time.Second); err != nil {
				log.Printf("worker [%s]: resize task failed for container %d: %v", id, p.NewVMID, err)
			}
		}
		time.Sleep(2 * time.Second)
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

	// Step 6: Inject credentials + user-data via pct exec (requires host SSH)
	if w.hostSSH.IsEnabled() && (p.Password != "" || p.SSHKeys != "" || p.UserData != "") {
		log.Printf("worker [%s]: injecting credentials into container %d via pct exec", id, p.NewVMID)
		// Give container a moment to fully start services
		time.Sleep(5 * time.Second)
		if err := w.injectLXCCredentials(id, p); err != nil {
			log.Printf("worker [%s]: failed to inject credentials: %v", id, err)
		}
	} else if !w.hostSSH.IsEnabled() && (p.Password != "" || p.SSHKeys != "") {
		log.Printf("worker [%s]: host SSH not configured — credentials will not be injected. Set PROXMOX_SSH_HOST to enable.", id)
	}

	// Done
	w.store.UpdateStep(id, StepReady, StatusCompleted)
	log.Printf("worker [%s]: job completed (container %d on %s)", id, p.NewVMID, targetNode)
}

// injectLXCCredentials runs pct exec commands on the Proxmox host to set
// root password, add SSH keys, enable password/root SSH, and run user-data.
func (w *Worker) injectLXCCredentials(id string, p ProvisionPayload) error {
	vmid := p.NewVMID

	// Build a best-effort script — each command has its own error handling.
	// No `set -e` because we want to continue on failures.
	var script strings.Builder

	script.WriteString(`#!/bin/bash
# PVE Pilot LXC credential injection

# Ensure openssh-server is installed (best effort across distros)
if ! command -v sshd >/dev/null 2>&1 && ! [ -x /usr/sbin/sshd ]; then
  (apt-get update -qq && apt-get install -y -qq openssh-server) >/dev/null 2>&1 \
    || dnf install -y openssh-server >/dev/null 2>&1 \
    || apk add --no-cache openssh >/dev/null 2>&1 \
    || true
fi
`)

	// Set root password (write to stdin of chpasswd via heredoc)
	if p.Password != "" {
		fmt.Fprintf(&script, `
# Set root password
cat <<'PVEPILOT_PW_EOF' | chpasswd
root:%s
PVEPILOT_PW_EOF
`, p.Password)
	}

	// Add SSH public key to root's authorized_keys
	if p.SSHKeys != "" {
		fmt.Fprintf(&script, `
# Add SSH public key
mkdir -p /root/.ssh
chmod 700 /root/.ssh
cat <<'PVEPILOT_KEY_EOF' >> /root/.ssh/authorized_keys
%s
PVEPILOT_KEY_EOF
chmod 600 /root/.ssh/authorized_keys
# Deduplicate in case the same key was added before
sort -u /root/.ssh/authorized_keys -o /root/.ssh/authorized_keys 2>/dev/null || true
`, p.SSHKeys)
	}

	// Enable password auth and root login in sshd_config
	if p.Password != "" || p.SSHKeys != "" {
		rootLoginMode := "prohibit-password"
		if p.Password != "" {
			rootLoginMode = "yes"
		}
		fmt.Fprintf(&script, `
# Configure sshd for the requested auth
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^#*PermitRootLogin.*/PermitRootLogin %s/' /etc/ssh/sshd_config 2>/dev/null || true
fi
`, rootLoginMode)

		if p.Password != "" {
			script.WriteString(`
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config 2>/dev/null || true
fi
# Also patch drop-in configs if any exist
for f in /etc/ssh/sshd_config.d/*.conf; do
  [ -f "$f" ] || continue
  sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication yes/' "$f" 2>/dev/null || true
  sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' "$f" 2>/dev/null || true
done
`)
		}

		script.WriteString(`
# Restart sshd (try multiple service names)
systemctl restart sshd 2>/dev/null \
  || systemctl restart ssh 2>/dev/null \
  || service ssh restart 2>/dev/null \
  || service sshd restart 2>/dev/null \
  || true
`)
	}

	script.WriteString("\necho 'PVE Pilot credential injection complete'\nexit 0\n")

	if err := w.hostSSH.PctExec(vmid, script.String()); err != nil {
		return fmt.Errorf("credential setup: %w", err)
	}
	log.Printf("worker [%s]: credentials injected into container %d", id, vmid)

	// Run user-data script separately
	if p.UserData != "" {
		log.Printf("worker [%s]: running user-data script in container %d", id, vmid)
		if err := w.hostSSH.PctExec(vmid, p.UserData); err != nil {
			return fmt.Errorf("user-data: %w", err)
		}
		log.Printf("worker [%s]: user-data executed in container %d", id, vmid)
	}

	return nil
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

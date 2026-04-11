package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
)

// vmNameForBackup fetches the VM/container name for job display.
func vmNameForBackup(node string, vmid int, vmType string) string {
	if vmType == "container" {
		st, err := PVE.GetContainerStatus(node, vmid)
		if err == nil && st != nil {
			return st.Name
		}
	} else {
		st, err := PVE.GetVMStatus(node, vmid)
		if err == nil && st != nil {
			return st.Name
		}
	}
	return fmt.Sprintf("%d", vmid)
}

// BackupStorage is the target storage for all backups (e.g. "nfs-drive").
var BackupStorage string

// BackupVM triggers an async backup of a QEMU VM via NATS job.
func BackupVM(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&body)

	name := vmNameForBackup(node, vmid, "vm")
	jobID, err := submitBackupJob("vm", node, vmid, name, BackupStorage, body.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
}

// BackupContainer triggers an async backup of an LXC container via NATS job.
func BackupContainer(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&body)

	name := vmNameForBackup(node, vmid, "container")
	jobID, err := submitBackupJob("container", node, vmid, name, BackupStorage, body.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
}

// ListVMBackups lists backups for a specific VM.
func ListVMBackups(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	listBackupsForVMID(c, node, vmid)
}

// ListContainerBackups lists backups for a specific container.
func ListContainerBackups(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	listBackupsForVMID(c, node, vmid)
}

func listBackupsForVMID(c *gin.Context, node string, vmid int) {
	all, err := PVE.ListBackups(node, BackupStorage)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Filter by VMID
	var filtered []proxmox.BackupInfo
	for _, b := range all {
		if b.VMID == vmid {
			filtered = append(filtered, b)
		}
	}

	// Sort newest first
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CTime > filtered[j].CTime
	})

	c.JSON(http.StatusOK, filtered)
}

// DeleteBackupHandler removes a backup volume.
func DeleteBackupHandler(c *gin.Context) {
	node := c.Query("node")
	volid := c.Query("volid")
	if node == "" || volid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node and volid query params required"})
		return
	}

	upid, err := PVE.DeleteBackup(node, volid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	if upid != "" {
		if err := PVE.WaitForTask(node, upid, 60*time.Second); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("delete failed: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// nextAvailableVMID finds the next VMID >= 10000 by scanning cluster resources.
func nextAvailableVMID() (int, error) {
	resources, err := PVE.GetClusterResources()
	if err != nil {
		return 0, err
	}
	maxID := 9999
	for _, r := range resources {
		if (r.Type == "qemu" || r.Type == "lxc") && r.VMID > maxID {
			maxID = r.VMID
		}
	}
	return maxID + 1, nil
}

// RestoreVMHandler restores a QEMU VM from a backup via async NATS job.
func RestoreVMHandler(c *gin.Context) {
	node := c.Param("node")

	var req proxmox.RestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vmid := req.VMID

	// Auto-assign VMID for "restore as new"
	if vmid <= 0 {
		next, err := nextAvailableVMID()
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to find next VMID: %v", err)})
			return
		}
		vmid = next
	}

	storage := req.Storage
	if storage == "" {
		storage = BackupStorage
	}

	name := vmNameForBackup(node, vmid, "vm")
	jobID, err := submitRestoreJob("vm", node, vmid, name, req.Archive, storage, req.InPlace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "vmid": vmid})
}

// RestoreContainerHandler restores an LXC container from a backup via async NATS job.
func RestoreContainerHandler(c *gin.Context) {
	node := c.Param("node")

	var req proxmox.RestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vmid := req.VMID

	// Auto-assign VMID for "restore as new"
	if vmid <= 0 {
		next, err := nextAvailableVMID()
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to find next VMID: %v", err)})
			return
		}
		vmid = next
	}

	storage := req.Storage
	if storage == "" {
		storage = BackupStorage
	}

	name := vmNameForBackup(node, vmid, "container")
	jobID, err := submitRestoreJob("container", node, vmid, name, req.Archive, storage, req.InPlace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "vmid": vmid})
}

// ListBackupSchedulesHandler returns all cluster backup schedules.
func ListBackupSchedulesHandler(c *gin.Context) {
	schedules, err := PVE.ListBackupSchedules()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedules)
}

// CreateBackupScheduleHandler creates a new backup schedule.
func CreateBackupScheduleHandler(c *gin.Context) {
	var req proxmox.BackupScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to backup storage
	if req.Storage == "" {
		req.Storage = BackupStorage
	}

	id, err := PVE.CreateBackupSchedule(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// DeleteBackupScheduleHandler removes a backup schedule.
func DeleteBackupScheduleHandler(c *gin.Context) {
	id := c.Param("id")
	if err := PVE.DeleteBackupSchedule(id); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

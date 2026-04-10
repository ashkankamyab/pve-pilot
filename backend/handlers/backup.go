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

// BackupStorage is the target storage for all backups (e.g. "nfs-drive").
var BackupStorage string

// BackupVM triggers an instant backup of a QEMU VM.
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

	upid, err := PVE.Backup(node, vmid, BackupStorage, "snapshot", "zstd", body.Notes)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Wait for backup to complete (can take minutes)
	if err := PVE.WaitForTask(node, upid, 10*time.Minute); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("backup failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "upid": upid})
}

// BackupContainer triggers an instant backup of an LXC container.
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

	upid, err := PVE.Backup(node, vmid, BackupStorage, "snapshot", "zstd", body.Notes)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	if err := PVE.WaitForTask(node, upid, 10*time.Minute); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("backup failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "upid": upid})
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

// RestoreVMHandler restores a QEMU VM from a backup.
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

	// In-place: stop + delete existing, then restore to same VMID
	if req.InPlace && req.VMID > 0 {
		// Stop if running
		status, _ := PVE.GetVMStatus(node, vmid)
		if status != nil && status.Status == "running" {
			stopUpid, err := PVE.StopVM(node, vmid)
			if err == nil {
				_ = PVE.WaitForTask(node, stopUpid, 60*time.Second)
			}
			// Wait for stopped
			for i := 0; i < 15; i++ {
				s, _ := PVE.GetVMStatus(node, vmid)
				if s != nil && s.Status == "stopped" {
					break
				}
				time.Sleep(2 * time.Second)
			}
		}
		// Delete
		delUpid, err := PVE.DeleteVM(node, vmid)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("delete old VM failed: %v", err)})
			return
		}
		if err := PVE.WaitForTask(node, delUpid, 60*time.Second); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("delete task failed: %v", err)})
			return
		}
		time.Sleep(3 * time.Second) // let Proxmox release resources
	}

	// Restore
	storage := req.Storage
	if storage == "" {
		storage = BackupStorage
	}

	upid, err := PVE.RestoreVM(node, req.Archive, vmid, storage)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("restore failed: %v", err)})
		return
	}

	if err := PVE.WaitForTask(node, upid, 10*time.Minute); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("restore task failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "restored", "vmid": vmid})
}

// RestoreContainerHandler restores an LXC container from a backup.
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

	// In-place: stop + delete existing
	if req.InPlace && req.VMID > 0 {
		status, _ := PVE.GetContainerStatus(node, vmid)
		if status != nil && status.Status == "running" {
			stopUpid, err := PVE.StopContainer(node, vmid)
			if err == nil {
				_ = PVE.WaitForTask(node, stopUpid, 60*time.Second)
			}
			for i := 0; i < 15; i++ {
				s, _ := PVE.GetContainerStatus(node, vmid)
				if s != nil && s.Status == "stopped" {
					break
				}
				time.Sleep(2 * time.Second)
			}
		}
		delUpid, err := PVE.DeleteContainer(node, vmid)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("delete old container failed: %v", err)})
			return
		}
		if err := PVE.WaitForTask(node, delUpid, 60*time.Second); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("delete task failed: %v", err)})
			return
		}
		time.Sleep(3 * time.Second)
	}

	storage := req.Storage
	if storage == "" {
		storage = BackupStorage
	}

	upid, err := PVE.RestoreContainer(node, req.Archive, vmid, storage)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("restore failed: %v", err)})
		return
	}

	if err := PVE.WaitForTask(node, upid, 10*time.Minute); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("restore task failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "restored", "vmid": vmid})
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

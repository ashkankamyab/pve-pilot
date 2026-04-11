package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ashkankamyab/pve-pilot/jobs"
	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
)

func ListVMs(c *gin.Context) {
	node := c.Param("node")
	vms, err := PVE.ListVMs(node)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vms)
}

func GetVMStatus(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	status, err := PVE.GetVMStatus(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func StartVM(c *gin.Context) {
	node := c.Param("node")
	vmid, _ := strconv.Atoi(c.Param("vmid"))
	upid, err := PVE.StartVM(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func StopVM(c *gin.Context) {
	node := c.Param("node")
	vmid, _ := strconv.Atoi(c.Param("vmid"))
	upid, err := PVE.StopVM(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func RebootVM(c *gin.Context) {
	node := c.Param("node")
	vmid, _ := strconv.Atoi(c.Param("vmid"))
	upid, err := PVE.RebootVM(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func DeleteVM(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	upid, err := PVE.DeleteVM(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func CloneVM(c *gin.Context) {
	node := c.Param("node")
	vmid, _ := strconv.Atoi(c.Param("vmid"))

	var req proxmox.CloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	full := false
	if req.Full != nil {
		full = *req.Full
	}

	upid, err := PVE.CloneVM(node, vmid, req.NewID, req.Name, req.Target, "", full)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

// ProvisionVM queues a provision job via NATS and returns immediately.
func ProvisionVM(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	var req proxmox.ProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	full := true
	if req.Full != nil {
		full = *req.Full
	}

	targetNode := node
	if req.Target != "" {
		targetNode = req.Target
	}

	payload := &jobs.ProvisionPayload{
		NewVMID:      req.NewID,
		Name:         req.Name,
		Storage:      req.Storage,
		CIUser:       req.CIUser,
		Password:     req.Password,
		SSHKeys:      req.SSHKeys,
		Cores:        req.Cores,
		Memory:       req.Memory,
		DiskSize:     req.DiskSize,
		ExtraVolumes: req.ExtraVolumes,
		UserData:     req.UserData,
		DNSDomain:    DNSDomain,
		IPMode:       req.IPMode,
		StaticIP:     req.IP,
		StaticGW:     req.Gateway,
		StaticSubnet: req.Subnet,
		FullClone:    full,
	}

	jobID, err := submitProvisionJob("vm", node, vmid, targetNode, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": jobID,
		"vmid":   req.NewID,
		"node":   targetNode,
	})
}

// GetVMInterfaces returns network interfaces from the QEMU guest agent
func GetVMFilesystems(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	filesystems, err := PVE.GetVMFilesystems(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, filesystems)
}

func GetVMInterfaces(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	interfaces, err := PVE.GetVMInterfaces(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, interfaces)
}

// GetVMConfig returns the full QEMU VM configuration.
func GetVMConfig(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	cfg, err := PVE.GetVMConfig(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// ScaleVM sets cores/memory on a VM. If running, orchestrates stop → apply → start.
func ScaleVM(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	var req proxmox.ScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := PVE.GetVMStatus(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	restarted := false

	if status.Status == "running" {
		// Stop
		stopUpid, err := PVE.StopVM(node, vmid)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("stop failed: %v", err)})
			return
		}
		if err := PVE.WaitForTask(node, stopUpid, 60*time.Second); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("stop task failed: %v", err)})
			return
		}
		// Wait for stopped state
		for i := 0; i < 15; i++ {
			s, _ := PVE.GetVMStatus(node, vmid)
			if s != nil && s.Status == "stopped" {
				break
			}
			time.Sleep(2 * time.Second)
		}
		restarted = true
	}

	// Apply
	if err := PVE.SetVMResources(node, vmid, req.Cores, req.MemoryMB); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("apply failed: %v", err)})
		return
	}

	if restarted {
		// Start
		startUpid, err := PVE.StartVM(node, vmid)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("start failed: %v", err)})
			return
		}
		if err := PVE.WaitForTask(node, startUpid, 60*time.Second); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("start task failed: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"restarted": restarted, "cores": req.Cores, "memory": req.MemoryMB})
}

// ResizeVMDisk grows a disk on a QEMU VM (hot resize supported).
func ResizeVMDisk(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	var req proxmox.ResizeDiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := PVE.ResizeDisk(node, vmid, req.Disk, req.Size); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"disk": req.Disk, "size": req.Size})
}

// AddVMDisk attaches a new disk to a QEMU VM, auto-selecting the next free scsiN slot.
func AddVMDisk(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	var req proxmox.AddDiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find next free scsi slot
	cfg, err := PVE.GetVMConfig(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	bus := ""
	for i := 1; i <= 13; i++ {
		key := fmt.Sprintf("scsi%d", i)
		if _, exists := cfg[key]; !exists {
			bus = key
			break
		}
	}
	if bus == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "no free SCSI slots (scsi1-scsi13 all in use)"})
		return
	}

	if err := PVE.AddDisk(node, vmid, bus, req.Storage, req.SizeGB); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bus": bus, "storage": req.Storage, "size_gb": req.SizeGB})
}

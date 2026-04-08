package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

	upid, err := PVE.CloneVM(node, vmid, req.NewID, req.Name, req.Target, full)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

// ProvisionVM orchestrates: clone -> configure cloud-init -> resize disk -> start
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

	// Step 1: Clone
	_, err = PVE.CloneVM(node, vmid, req.NewID, req.Name, req.Target, full)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("clone failed: %v", err)})
		return
	}

	// Wait for clone to complete by polling VM status
	targetNode := node
	if req.Target != "" {
		targetNode = req.Target
	}
	if err := waitForVM(targetNode, req.NewID, 120*time.Second); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("waiting for clone: %v", err)})
		return
	}

	// Step 2: Configure cloud-init
	if req.Password != "" || req.SSHKeys != "" {
		if err := PVE.ConfigureCloudInit(targetNode, req.NewID, req.Password, req.SSHKeys); err != nil {
			log.Printf("WARNING: cloud-init config failed for VM %d: %v", req.NewID, err)
			// Don't fail the whole operation
		}
	}

	// Step 3: Resize disk
	if req.DiskSize > 0 {
		sizeStr := fmt.Sprintf("%dG", req.DiskSize)
		if err := PVE.ResizeDisk(targetNode, req.NewID, "scsi0", sizeStr); err != nil {
			// Try virtio0 as fallback
			if err2 := PVE.ResizeDisk(targetNode, req.NewID, "virtio0", sizeStr); err2 != nil {
				log.Printf("WARNING: disk resize failed for VM %d: scsi0: %v, virtio0: %v", req.NewID, err, err2)
			}
		}
	}

	// Step 4: Start VM
	_, err = PVE.StartVM(targetNode, req.NewID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("start failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vmid": req.NewID,
		"node": targetNode,
	})
}

// GetVMInterfaces returns network interfaces from the QEMU guest agent
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

// waitForVM polls until the VM exists and is stopped (clone complete)
func waitForVM(node string, vmid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := PVE.GetVMStatus(node, vmid)
		if err == nil && status != nil {
			// Clone is done when the VM exists (status will be "stopped")
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for VM %d to be ready", vmid)
}

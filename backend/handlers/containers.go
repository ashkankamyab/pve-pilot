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

func ListContainers(c *gin.Context) {
	node := c.Param("node")
	containers, err := PVE.ListContainers(node)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, containers)
}

func GetContainerStatus(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	status, err := PVE.GetContainerStatus(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func StartContainer(c *gin.Context) {
	node := c.Param("node")
	vmid, _ := strconv.Atoi(c.Param("vmid"))
	upid, err := PVE.StartContainer(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func StopContainer(c *gin.Context) {
	node := c.Param("node")
	vmid, _ := strconv.Atoi(c.Param("vmid"))
	upid, err := PVE.StopContainer(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func RebootContainer(c *gin.Context) {
	node := c.Param("node")
	vmid, _ := strconv.Atoi(c.Param("vmid"))
	upid, err := PVE.RebootContainer(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func DeleteContainer(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	upid, err := PVE.DeleteContainer(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

func CloneContainer(c *gin.Context) {
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

	upid, err := PVE.CloneContainer(node, vmid, req.NewID, req.Name, req.Target, full)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

// ProvisionContainer orchestrates: clone -> configure -> resize disk -> start
func ProvisionContainer(c *gin.Context) {
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
	_, err = PVE.CloneContainer(node, vmid, req.NewID, req.Name, req.Target, full)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("clone failed: %v", err)})
		return
	}

	// Wait for clone to complete
	targetNode := node
	if req.Target != "" {
		targetNode = req.Target
	}
	if err := waitForContainer(targetNode, req.NewID, 120*time.Second); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("waiting for clone: %v", err)})
		return
	}

	// Step 2: Configure cloud-init / password
	if req.Password != "" || req.SSHKeys != "" {
		if err := PVE.ConfigureContainerCloudInit(targetNode, req.NewID, req.Password, req.SSHKeys); err != nil {
			log.Printf("WARNING: cloud-init config failed for container %d: %v", req.NewID, err)
		}
	}

	// Step 3: Resize disk
	if req.DiskSize > 0 {
		sizeStr := fmt.Sprintf("%dG", req.DiskSize)
		if err := PVE.ResizeContainerDisk(targetNode, req.NewID, "rootfs", sizeStr); err != nil {
			log.Printf("WARNING: disk resize failed for container %d: %v", req.NewID, err)
		}
	}

	// Step 4: Start container
	_, err = PVE.StartContainer(targetNode, req.NewID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("start failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vmid": req.NewID,
		"node": targetNode,
	})
}

// waitForContainer polls until the container exists (clone complete)
func waitForContainer(node string, vmid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := PVE.GetContainerStatus(node, vmid)
		if err == nil && status != nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for container %d to be ready", vmid)
}

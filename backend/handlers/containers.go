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

func ListContainers(c *gin.Context) {
	node := c.Param("node")
	containers, err := PVE.ListContainers(node)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, containers)
}

func GetContainerConfig(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	cfg, err := PVE.GetContainerConfig(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func GetContainerInterfaces(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}
	ifaces, err := PVE.GetContainerInterfaces(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ifaces)
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

	upid, err := PVE.CloneContainer(node, vmid, req.NewID, req.Name, req.Target, "", full)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upid": upid})
}

// ProvisionContainer queues a provision job via NATS and returns immediately.
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

	jobID, err := submitProvisionJob("container", node, vmid, targetNode, payload)
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

// ScaleContainer sets cores/memory on a container (hot update, no restart needed).
func ScaleContainer(c *gin.Context) {
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

	if err := PVE.SetContainerResources(node, vmid, req.Cores, req.MemoryMB); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cores": req.Cores, "memory": req.MemoryMB})
}

// ResizeContainerDiskHandler grows a disk on an LXC container (hot resize, rootfs auto-expands).
func ResizeContainerDiskHandler(c *gin.Context) {
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

	upid, err := PVE.ResizeContainerDisk(node, vmid, req.Disk, req.Size)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	if upid != "" {
		if err := PVE.WaitForTask(node, upid, 120*time.Second); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resize task failed: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"disk": req.Disk, "size": req.Size})
}

// AddContainerVolume attaches a new mountpoint to an LXC container, auto-selecting the next free mpN.
func AddContainerVolume(c *gin.Context) {
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
		return
	}

	var req proxmox.AddVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find next free mp slot
	cfg, err := PVE.GetContainerConfig(node, vmid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	mpKey := ""
	for i := 0; i <= 255; i++ {
		key := fmt.Sprintf("mp%d", i)
		if _, exists := cfg[key]; !exists {
			mpKey = key
			break
		}
	}
	if mpKey == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "no free mount point slots"})
		return
	}

	if err := PVE.AddContainerMountPoint(node, vmid, mpKey, req.Storage, req.SizeGB, req.MountPath); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mount_point": mpKey, "storage": req.Storage, "size_gb": req.SizeGB, "path": req.MountPath})
}

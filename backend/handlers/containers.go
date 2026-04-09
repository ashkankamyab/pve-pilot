package handlers

import (
	"net/http"
	"strconv"

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
		DiskSize:     req.DiskSize,
		ExtraVolumes: req.ExtraVolumes,
		UserData:     req.UserData,
		DNSDomain:    DNSDomain,
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

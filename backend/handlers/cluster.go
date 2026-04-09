package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Settings returns public configuration values for the frontend.
func Settings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"dns_domain": DNSDomain,
	})
}

func ClusterResources(c *gin.Context) {
	resources, err := PVE.GetClusterResources()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resources)
}

// NextVMID returns the next available VMID (starting from 10000).
func NextVMID(c *gin.Context) {
	resources, err := PVE.GetClusterResources()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	maxID := 9999
	for _, r := range resources {
		if (r.Type == "qemu" || r.Type == "lxc") && r.VMID > maxID {
			maxID = r.VMID
		}
	}

	c.JSON(http.StatusOK, gin.H{"vmid": maxID + 1})
}

func ClusterSummary(c *gin.Context) {
	summary, err := PVE.GetClusterSummary()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ClusterResources(c *gin.Context) {
	resources, err := PVE.GetClusterResources()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resources)
}

func ClusterSummary(c *gin.Context) {
	summary, err := PVE.GetClusterSummary()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

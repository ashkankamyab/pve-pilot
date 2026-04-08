package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListNodes(c *gin.Context) {
	nodes, err := PVE.ListNodes()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

func GetNodeStatus(c *gin.Context) {
	node := c.Param("node")
	status, err := PVE.GetNodeStatus(node)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

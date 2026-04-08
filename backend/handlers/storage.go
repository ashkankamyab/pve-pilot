package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListNodeStorage(c *gin.Context) {
	node := c.Param("node")
	storage, err := PVE.ListStorage(node)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, storage)
}

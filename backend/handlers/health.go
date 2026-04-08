package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	pveStatus := "connected"
	if err := PVE.Ping(); err != nil {
		pveStatus = "unreachable"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"proxmox": pveStatus,
	})
}

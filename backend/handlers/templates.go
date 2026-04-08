package handlers

import (
	"net/http"

	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
)

func ListTemplates(c *gin.Context) {
	resources, err := PVE.GetClusterResources()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var templates []proxmox.TemplateInfo
	for _, r := range resources {
		if r.Template == 1 {
			vmType := ""
			switch r.Type {
			case "qemu":
				vmType = "qemu"
			case "lxc":
				vmType = "lxc"
			default:
				continue
			}
			templates = append(templates, proxmox.TemplateInfo{
				ClusterResource: r,
				VMType:          vmType,
			})
		}
	}

	c.JSON(http.StatusOK, templates)
}

package main

import (
	"log"

	"github.com/ashkankamyab/pve-pilot/config"
	"github.com/ashkankamyab/pve-pilot/handlers"
	"github.com/ashkankamyab/pve-pilot/middleware"
	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	if cfg.ProxmoxTokenID == "" || cfg.ProxmoxTokenSecret == "" {
		log.Fatal("PROXMOX_TOKEN_ID and PROXMOX_TOKEN_SECRET are required")
	}

	handlers.PVE = proxmox.NewClient(
		cfg.ProxmoxURL,
		cfg.ProxmoxTokenID,
		cfg.ProxmoxTokenSecret,
		cfg.InsecureTLS,
	)

	if err := handlers.PVE.Ping(); err != nil {
		log.Printf("WARNING: Cannot reach Proxmox at %s: %v", cfg.ProxmoxURL, err)
	} else {
		log.Printf("Connected to Proxmox at %s", cfg.ProxmoxURL)
	}

	r := gin.Default()
	r.Use(middleware.CORS(cfg.FrontendURL))

	api := r.Group("/api")
	{
		api.GET("/health", handlers.Health)

		api.GET("/cluster/resources", handlers.ClusterResources)
		api.GET("/cluster/summary", handlers.ClusterSummary)

		api.GET("/nodes", handlers.ListNodes)
		api.GET("/nodes/:node/status", handlers.GetNodeStatus)
		api.GET("/nodes/:node/storage", handlers.ListNodeStorage)

		api.GET("/nodes/:node/vms", handlers.ListVMs)
		api.GET("/nodes/:node/vms/:vmid/status", handlers.GetVMStatus)
		api.POST("/nodes/:node/vms/:vmid/start", handlers.StartVM)
		api.POST("/nodes/:node/vms/:vmid/stop", handlers.StopVM)
		api.POST("/nodes/:node/vms/:vmid/reboot", handlers.RebootVM)
		api.POST("/nodes/:node/vms/:vmid/clone", handlers.CloneVM)
		api.DELETE("/nodes/:node/vms/:vmid", handlers.DeleteVM)

		api.GET("/nodes/:node/containers", handlers.ListContainers)
		api.GET("/nodes/:node/containers/:vmid/status", handlers.GetContainerStatus)
		api.POST("/nodes/:node/containers/:vmid/start", handlers.StartContainer)
		api.POST("/nodes/:node/containers/:vmid/stop", handlers.StopContainer)
		api.POST("/nodes/:node/containers/:vmid/reboot", handlers.RebootContainer)
		api.POST("/nodes/:node/containers/:vmid/clone", handlers.CloneContainer)
		api.DELETE("/nodes/:node/containers/:vmid", handlers.DeleteContainer)

		api.GET("/templates", handlers.ListTemplates)
	}

	log.Printf("PVE Pilot backend starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

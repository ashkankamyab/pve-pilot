package main

import (
	"log"

	"github.com/ashkankamyab/pve-pilot/config"
	"github.com/ashkankamyab/pve-pilot/handlers"
	"github.com/ashkankamyab/pve-pilot/jobs"
	"github.com/ashkankamyab/pve-pilot/middleware"
	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
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

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("Cannot connect to NATS at %s: %v", cfg.NatsURL, err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS at %s", cfg.NatsURL)

	// Initialize job store and wire into handlers
	jobStore := jobs.NewStore()
	handlers.JobStore = jobStore
	handlers.NatsConn = nc
	handlers.DNSDomain = cfg.DNSDomain

	// Start worker
	worker := jobs.NewWorker(nc, jobStore, handlers.PVE)
	if err := worker.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	r := gin.Default()
	r.Use(middleware.CORS(cfg.FrontendURL))

	api := r.Group("/api")
	{
		api.GET("/health", handlers.Health)
		api.GET("/settings", handlers.Settings)

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
		api.POST("/nodes/:node/vms/:vmid/provision", handlers.ProvisionVM)
		api.GET("/nodes/:node/vms/:vmid/interfaces", handlers.GetVMInterfaces)
		api.GET("/nodes/:node/vms/:vmid/filesystems", handlers.GetVMFilesystems)
		api.DELETE("/nodes/:node/vms/:vmid", handlers.DeleteVM)

		api.GET("/nodes/:node/containers", handlers.ListContainers)
		api.GET("/nodes/:node/containers/:vmid/status", handlers.GetContainerStatus)
		api.POST("/nodes/:node/containers/:vmid/start", handlers.StartContainer)
		api.POST("/nodes/:node/containers/:vmid/stop", handlers.StopContainer)
		api.POST("/nodes/:node/containers/:vmid/reboot", handlers.RebootContainer)
		api.POST("/nodes/:node/containers/:vmid/clone", handlers.CloneContainer)
		api.POST("/nodes/:node/containers/:vmid/provision", handlers.ProvisionContainer)
		api.DELETE("/nodes/:node/containers/:vmid", handlers.DeleteContainer)

		api.GET("/templates", handlers.ListTemplates)
		api.GET("/next-vmid", handlers.NextVMID)

		// Job endpoints
		api.GET("/jobs", handlers.ListJobs)
		api.GET("/jobs/:id", handlers.GetJob)
		api.GET("/jobs/:id/events", handlers.StreamJobEvents)
	}

	log.Printf("PVE Pilot backend starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

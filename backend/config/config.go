package config

import (
	"os"
	"strconv"
)

type Config struct {
	ProxmoxURL         string
	ProxmoxTokenID     string
	ProxmoxTokenSecret string
	FrontendURL        string
	Port               string
	InsecureTLS        bool
}

func Load() Config {
	insecure, _ := strconv.ParseBool(getEnv("INSECURE_TLS", "true"))

	return Config{
		ProxmoxURL:         getEnv("PROXMOX_URL", "https://localhost:8006"),
		ProxmoxTokenID:     getEnv("PROXMOX_TOKEN_ID", ""),
		ProxmoxTokenSecret: getEnv("PROXMOX_TOKEN_SECRET", ""),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		Port:               getEnv("PORT", "8080"),
		InsecureTLS:        insecure,
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

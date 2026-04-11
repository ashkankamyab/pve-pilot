package handlers

import (
	"net/http"
	"testing"

	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
)

func TestHealth(t *testing.T) {
	server := setupMockPVE(t, map[string]mockRoute{
		"version": {response: map[string]string{"version": "8.0"}},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/health", Health)

	w := performRequest(r, "GET", "/api/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := parseJSON(t, w)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
	if body["proxmox"] != "connected" {
		t.Errorf("expected proxmox connected, got %v", body["proxmox"])
	}
}

func TestHealthUnreachable(t *testing.T) {
	// Point PVE at a dead server
	PVE = proxmox.NewClient("http://127.0.0.1:1", "test@pve!token", "secret", true)

	r := gin.New()
	r.GET("/api/health", Health)

	w := performRequest(r, "GET", "/api/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := parseJSON(t, w)
	if body["proxmox"] != "unreachable" {
		t.Errorf("expected proxmox unreachable, got %v", body["proxmox"])
	}
}

func TestSettings(t *testing.T) {
	DNSDomain = "ashkmb.com"
	DefaultGateway = "192.168.2.1"

	r := gin.New()
	r.GET("/api/settings", Settings)

	w := performRequest(r, "GET", "/api/settings", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := parseJSON(t, w)
	if body["dns_domain"] != "ashkmb.com" {
		t.Errorf("expected ashkmb.com, got %v", body["dns_domain"])
	}
	if body["default_gateway"] != "192.168.2.1" {
		t.Errorf("expected 192.168.2.1, got %v", body["default_gateway"])
	}
}

func TestClusterResources(t *testing.T) {
	resources := []proxmox.ClusterResource{
		{ID: "node/pve", Type: "node", Node: "pve", Status: "online"},
		{ID: "qemu/100", Type: "qemu", VMID: 100, Status: "running"},
	}

	server := setupMockPVE(t, map[string]mockRoute{
		"cluster/resources": {response: resources},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/cluster/resources", ClusterResources)

	w := performRequest(r, "GET", "/api/cluster/resources", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClusterResourcesAPIError(t *testing.T) {
	server := setupMockPVE(t, map[string]mockRoute{
		"cluster/resources": {status: http.StatusInternalServerError, response: "error"},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/cluster/resources", ClusterResources)

	w := performRequest(r, "GET", "/api/cluster/resources", nil)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestNextVMID(t *testing.T) {
	resources := []proxmox.ClusterResource{
		{Type: "qemu", VMID: 10001},
		{Type: "lxc", VMID: 10003},
		{Type: "qemu", VMID: 9000, Template: 1}, // template
		{Type: "storage", Storage: "local"},       // not qemu/lxc
	}

	server := setupMockPVE(t, map[string]mockRoute{
		"cluster/resources": {response: resources},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/next-vmid", NextVMID)

	w := performRequest(r, "GET", "/api/next-vmid", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := parseJSON(t, w)
	vmid := body["vmid"].(float64)
	if vmid != 10004 {
		t.Errorf("expected next VMID 10004, got %v", vmid)
	}
}

func TestNextVMIDNoExisting(t *testing.T) {
	resources := []proxmox.ClusterResource{
		{Type: "node", Node: "pve"},
		{Type: "storage", Storage: "local"},
	}

	server := setupMockPVE(t, map[string]mockRoute{
		"cluster/resources": {response: resources},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/next-vmid", NextVMID)

	w := performRequest(r, "GET", "/api/next-vmid", nil)
	body := parseJSON(t, w)
	vmid := body["vmid"].(float64)
	if vmid != 10000 {
		t.Errorf("expected first VMID 10000, got %v", vmid)
	}
}

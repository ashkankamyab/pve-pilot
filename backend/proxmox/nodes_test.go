package proxmox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClusterSummary(t *testing.T) {
	resources := []ClusterResource{
		{Type: "node", Status: "online", MaxCPU: 8, CPU: 0.25, Mem: 4 * 1024 * 1024 * 1024, MaxMem: 16 * 1024 * 1024 * 1024},
		{Type: "node", Status: "online", MaxCPU: 8, CPU: 0.50, Mem: 8 * 1024 * 1024 * 1024, MaxMem: 16 * 1024 * 1024 * 1024},
		{Type: "qemu", Status: "running", VMID: 100, Template: 0},
		{Type: "qemu", Status: "stopped", VMID: 101, Template: 0},
		{Type: "qemu", Status: "running", VMID: 9000, Template: 1}, // template, should not count
		{Type: "lxc", Status: "running", VMID: 200, Template: 0},
		{Type: "lxc", Status: "stopped", VMID: 201, Template: 0},
		{Type: "storage", Disk: 100 * 1024 * 1024 * 1024, MaxDisk: 500 * 1024 * 1024 * 1024},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": resources})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	summary, err := c.GetClusterSummary()
	if err != nil {
		t.Fatalf("GetClusterSummary failed: %v", err)
	}

	if summary.Nodes != 2 {
		t.Errorf("expected 2 nodes, got %d", summary.Nodes)
	}
	if summary.NodesOnline != 2 {
		t.Errorf("expected 2 online nodes, got %d", summary.NodesOnline)
	}
	if summary.VMsTotal != 2 { // templates excluded
		t.Errorf("expected 2 VMs total, got %d", summary.VMsTotal)
	}
	if summary.VMsRunning != 1 {
		t.Errorf("expected 1 running VM, got %d", summary.VMsRunning)
	}
	if summary.ContainersTotal != 2 {
		t.Errorf("expected 2 containers total, got %d", summary.ContainersTotal)
	}
	if summary.ContainersRunning != 1 {
		t.Errorf("expected 1 running container, got %d", summary.ContainersRunning)
	}
	if summary.CPUTotal != 16 {
		t.Errorf("expected 16 total CPU, got %d", summary.CPUTotal)
	}
	// CPU usage should be averaged: (0.25+0.50)/2 = 0.375
	if summary.CPUUsage < 0.37 || summary.CPUUsage > 0.38 {
		t.Errorf("expected CPU usage ~0.375, got %f", summary.CPUUsage)
	}
}

func TestGetClusterSummaryEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []ClusterResource{}})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	summary, err := c.GetClusterSummary()
	if err != nil {
		t.Fatalf("GetClusterSummary failed: %v", err)
	}
	if summary.Nodes != 0 {
		t.Errorf("expected 0 nodes, got %d", summary.Nodes)
	}
}

func TestListNodes(t *testing.T) {
	resources := []ClusterResource{
		{Type: "node", Node: "pve1", Status: "online"},
		{Type: "qemu", VMID: 100, Status: "running"},
		{Type: "node", Node: "pve2", Status: "offline"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": resources})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	nodes, err := c.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Node != "pve1" || nodes[1].Node != "pve2" {
		t.Errorf("unexpected nodes: %v", nodes)
	}
}

package handlers

import (
	"net/http"
	"testing"

	"github.com/ashkankamyab/pve-pilot/proxmox"
	"github.com/gin-gonic/gin"
)

func TestListVMs(t *testing.T) {
	vms := []proxmox.VMStatus{
		{VMID: 100, Name: "web", Status: "running"},
		{VMID: 101, Name: "db", Status: "stopped"},
	}

	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu": {response: vms},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/nodes/:node/vms", ListVMs)

	w := performRequest(r, "GET", "/api/nodes/pve/vms", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetVMStatus(t *testing.T) {
	vm := proxmox.VMStatus{VMID: 100, Name: "web", Status: "running", CPUs: 4}

	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100/status/current": {response: vm},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/nodes/:node/vms/:vmid/status", GetVMStatus)

	w := performRequest(r, "GET", "/api/nodes/pve/vms/100/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := parseJSON(t, w)
	if body["name"] != "web" {
		t.Errorf("expected name web, got %v", body["name"])
	}
}

func TestGetVMStatusInvalidVMID(t *testing.T) {
	r := gin.New()
	r.GET("/api/nodes/:node/vms/:vmid/status", GetVMStatus)

	w := performRequest(r, "GET", "/api/nodes/pve/vms/notanumber/status", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := parseJSON(t, w)
	if body["error"] != "invalid vmid" {
		t.Errorf("expected 'invalid vmid' error, got %v", body["error"])
	}
}

func TestStartVM(t *testing.T) {
	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100/status/start": {method: "POST", response: "UPID:pve:start100"},
	})
	defer server.Close()

	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/start", StartVM)

	w := performRequest(r, "POST", "/api/nodes/pve/vms/100/start", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := parseJSON(t, w)
	if body["upid"] != "UPID:pve:start100" {
		t.Errorf("expected UPID, got %v", body["upid"])
	}
}

func TestStopVM(t *testing.T) {
	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100/status/stop": {method: "POST", response: "UPID:pve:stop100"},
	})
	defer server.Close()

	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/stop", StopVM)

	w := performRequest(r, "POST", "/api/nodes/pve/vms/100/stop", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDeleteVM(t *testing.T) {
	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100": {method: "DELETE", response: "UPID:pve:del100"},
	})
	defer server.Close()

	r := gin.New()
	r.DELETE("/api/nodes/:node/vms/:vmid", DeleteVM)

	w := performRequest(r, "DELETE", "/api/nodes/pve/vms/100", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteVMInvalidVMID(t *testing.T) {
	r := gin.New()
	r.DELETE("/api/nodes/:node/vms/:vmid", DeleteVM)

	w := performRequest(r, "DELETE", "/api/nodes/pve/vms/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCloneVM(t *testing.T) {
	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/9000/clone": {method: "POST", response: "UPID:pve:clone"},
	})
	defer server.Close()

	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/clone", CloneVM)

	body := map[string]interface{}{"newid": 10001, "name": "my-vm"}
	w := performRequest(r, "POST", "/api/nodes/pve/vms/9000/clone", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCloneVMMissingNewID(t *testing.T) {
	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/clone", CloneVM)

	body := map[string]interface{}{"name": "no-id"}
	w := performRequest(r, "POST", "/api/nodes/pve/vms/9000/clone", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetVMConfig(t *testing.T) {
	config := map[string]interface{}{
		"cores":  4,
		"memory": 2048,
		"scsi0":  "local-lvm:vm-100-disk-0,size=30G",
	}

	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100/config": {response: config},
	})
	defer server.Close()

	r := gin.New()
	r.GET("/api/nodes/:node/vms/:vmid/config", GetVMConfig)

	w := performRequest(r, "GET", "/api/nodes/pve/vms/100/config", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestResizeVMDisk(t *testing.T) {
	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100/resize": {method: "PUT", response: ""},
	})
	defer server.Close()

	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/resize-disk", ResizeVMDisk)

	body := map[string]interface{}{"disk": "scsi0", "size": "50G"}
	w := performRequest(r, "POST", "/api/nodes/pve/vms/100/resize-disk", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResizeVMDiskInvalidVMID(t *testing.T) {
	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/resize-disk", ResizeVMDisk)

	body := map[string]interface{}{"disk": "scsi0", "size": "50G"}
	w := performRequest(r, "POST", "/api/nodes/pve/vms/abc/resize-disk", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddVMDisk(t *testing.T) {
	config := map[string]interface{}{
		"scsi0": "local-lvm:vm-100-disk-0,size=30G",
		// scsi1 is free
	}

	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100/config": {response: config},
	})
	defer server.Close()

	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/add-disk", AddVMDisk)

	body := map[string]interface{}{"storage": "local-lvm", "size_gb": 50}
	w := performRequest(r, "POST", "/api/nodes/pve/vms/100/add-disk", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["bus"] != "scsi1" {
		t.Errorf("expected scsi1, got %v", result["bus"])
	}
}

func TestAddVMDiskAllSlotsFull(t *testing.T) {
	config := map[string]interface{}{
		"scsi0":  "disk0",
		"scsi1":  "disk1",
		"scsi2":  "disk2",
		"scsi3":  "disk3",
		"scsi4":  "disk4",
		"scsi5":  "disk5",
		"scsi6":  "disk6",
		"scsi7":  "disk7",
		"scsi8":  "disk8",
		"scsi9":  "disk9",
		"scsi10": "disk10",
		"scsi11": "disk11",
		"scsi12": "disk12",
		"scsi13": "disk13",
	}

	server := setupMockPVE(t, map[string]mockRoute{
		"nodes/pve/qemu/100/config": {response: config},
	})
	defer server.Close()

	r := gin.New()
	r.POST("/api/nodes/:node/vms/:vmid/add-disk", AddVMDisk)

	body := map[string]interface{}{"storage": "local-lvm", "size_gb": 10}
	w := performRequest(r, "POST", "/api/nodes/pve/vms/100/add-disk", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d: %s", w.Code, w.Body.String())
	}
}

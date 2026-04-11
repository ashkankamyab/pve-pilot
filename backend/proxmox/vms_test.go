package proxmox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListVMs(t *testing.T) {
	vms := []VMStatus{
		{VMID: 100, Name: "web-server", Status: "running", CPUs: 4, MaxMem: 8 * 1024 * 1024 * 1024},
		{VMID: 101, Name: "db-server", Status: "stopped", CPUs: 2, MaxMem: 4 * 1024 * 1024 * 1024},
	}

	server := newTestServer(t, map[string]interface{}{
		"nodes/pve/qemu": vms,
	})
	defer server.Close()

	c := newTestClient(t, server)
	result, err := c.ListVMs("pve")
	if err != nil {
		t.Fatalf("ListVMs failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 VMs, got %d", len(result))
	}
	if result[0].Name != "web-server" {
		t.Errorf("expected web-server, got %s", result[0].Name)
	}
}

func TestGetVMStatus(t *testing.T) {
	vm := VMStatus{VMID: 100, Name: "test-vm", Status: "running", CPUs: 4}

	server := newTestServer(t, map[string]interface{}{
		"nodes/pve/qemu/100/status/current": vm,
	})
	defer server.Close()

	c := newTestClient(t, server)
	status, err := c.GetVMStatus("pve", 100)
	if err != nil {
		t.Fatalf("GetVMStatus failed: %v", err)
	}
	if status.Name != "test-vm" {
		t.Errorf("expected test-vm, got %s", status.Name)
	}
	if status.Status != "running" {
		t.Errorf("expected running, got %s", status.Status)
	}
}

func TestStartVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api2/json/nodes/pve/qemu/100/status/start" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:start100"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	upid, err := c.StartVM("pve", 100)
	if err != nil {
		t.Fatalf("StartVM failed: %v", err)
	}
	if upid != "UPID:pve:start100" {
		t.Errorf("expected UPID:pve:start100, got %s", upid)
	}
}

func TestStopVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve/qemu/100/status/stop" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:stop100"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	upid, err := c.StopVM("pve", 100)
	if err != nil {
		t.Fatalf("StopVM failed: %v", err)
	}
	if upid != "UPID:pve:stop100" {
		t.Errorf("expected UPID:pve:stop100, got %s", upid)
	}
}

func TestDeleteVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api2/json/nodes/pve/qemu/100" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:del100"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	upid, err := c.DeleteVM("pve", 100)
	if err != nil {
		t.Fatalf("DeleteVM failed: %v", err)
	}
	if upid != "UPID:pve:del100" {
		t.Errorf("expected UPID:pve:del100, got %s", upid)
	}
}

func TestCloneVM(t *testing.T) {
	var gotParams map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve/qemu/9000/clone" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		r.ParseForm()
		gotParams = map[string]string{}
		for k, v := range r.PostForm {
			gotParams[k] = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:clone9000"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	upid, err := c.CloneVM("pve", 9000, 10001, "my-clone", "", "local-lvm", true)
	if err != nil {
		t.Fatalf("CloneVM failed: %v", err)
	}
	if upid != "UPID:pve:clone9000" {
		t.Errorf("unexpected UPID: %s", upid)
	}
	if gotParams["newid"] != "10001" {
		t.Errorf("expected newid=10001, got %s", gotParams["newid"])
	}
	if gotParams["name"] != "my-clone" {
		t.Errorf("expected name=my-clone, got %s", gotParams["name"])
	}
	if gotParams["full"] != "1" {
		t.Errorf("expected full=1, got %s", gotParams["full"])
	}
	if gotParams["storage"] != "local-lvm" {
		t.Errorf("expected storage=local-lvm, got %s", gotParams["storage"])
	}
}

func TestCloneVMLinked(t *testing.T) {
	var gotParams map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotParams = map[string]string{}
		for k, v := range r.PostForm {
			gotParams[k] = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve:linked"})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.CloneVM("pve", 9000, 10002, "linked-clone", "", "", false)
	if err != nil {
		t.Fatalf("CloneVM failed: %v", err)
	}
	// full should not be set for linked clone
	if _, exists := gotParams["full"]; exists {
		t.Error("full param should not be set for linked clone")
	}
}

func TestConfigureCloudInit(t *testing.T) {
	var gotParams map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotParams = map[string]string{}
		for k, v := range r.PostForm {
			gotParams[k] = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": ""})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.ConfigureCloudInit("pve", 100, "ubuntu", "pass123", "ssh-ed25519 AAAA user@host", "ashkmb.com", "static", "192.168.2.50", "192.168.2.1", 24)
	if err != nil {
		t.Fatalf("ConfigureCloudInit failed: %v", err)
	}

	if gotParams["ciuser"] != "ubuntu" {
		t.Errorf("expected ciuser=ubuntu, got %s", gotParams["ciuser"])
	}
	if gotParams["cipassword"] != "pass123" {
		t.Errorf("expected cipassword=pass123, got %s", gotParams["cipassword"])
	}
	if gotParams["searchdomain"] != "ashkmb.com" {
		t.Errorf("expected searchdomain=ashkmb.com, got %s", gotParams["searchdomain"])
	}
	if gotParams["ipconfig0"] != "ip=192.168.2.50/24,gw=192.168.2.1" {
		t.Errorf("unexpected ipconfig0: %s", gotParams["ipconfig0"])
	}
	// SSH keys should be URL-encoded with %20 for spaces
	if gotParams["sshkeys"] == "" {
		t.Error("sshkeys should be set")
	}
}

func TestConfigureCloudInitNoParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make HTTP request when no params")
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.ConfigureCloudInit("pve", 100, "", "", "", "", "dhcp", "", "", 0)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestSetVMResources(t *testing.T) {
	var gotParams map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotParams = map[string]string{}
		for k, v := range r.PostForm {
			gotParams[k] = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": ""})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.SetVMResources("pve", 100, 4, 2048)
	if err != nil {
		t.Fatalf("SetVMResources failed: %v", err)
	}
	if gotParams["cores"] != "4" {
		t.Errorf("expected cores=4, got %s", gotParams["cores"])
	}
	if gotParams["memory"] != "2048" {
		t.Errorf("expected memory=2048, got %s", gotParams["memory"])
	}
}

func TestSetVMResourcesNoParams(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.SetVMResources("pve", 100, 0, 0)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if called {
		t.Error("should not make HTTP request when no params")
	}
}

func TestResizeDisk(t *testing.T) {
	var gotParams map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		r.ParseForm()
		gotParams = map[string]string{}
		for k, v := range r.PostForm {
			gotParams[k] = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": ""})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.ResizeDisk("pve", 100, "scsi0", "50G")
	if err != nil {
		t.Fatalf("ResizeDisk failed: %v", err)
	}
	if gotParams["disk"] != "scsi0" {
		t.Errorf("expected disk=scsi0, got %s", gotParams["disk"])
	}
	if gotParams["size"] != "50G" {
		t.Errorf("expected size=50G, got %s", gotParams["size"])
	}
}

func TestAddDisk(t *testing.T) {
	var gotParams map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotParams = map[string]string{}
		for k, v := range r.PostForm {
			gotParams[k] = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": ""})
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.AddDisk("pve", 100, "scsi1", "local-lvm", 50)
	if err != nil {
		t.Fatalf("AddDisk failed: %v", err)
	}
	if gotParams["scsi1"] != "local-lvm:50" {
		t.Errorf("expected scsi1=local-lvm:50, got %s", gotParams["scsi1"])
	}
}

func TestGetVMInterfaces(t *testing.T) {
	interfaces := struct {
		Result []NetworkInterface `json:"result"`
	}{
		Result: []NetworkInterface{
			{Name: "lo", IPAddresses: []IPAddress{{Type: "ipv4", Address: "127.0.0.1"}}},
			{Name: "eth0", IPAddresses: []IPAddress{{Type: "ipv4", Address: "192.168.2.100", Prefix: 24}}},
		},
	}

	server := newTestServer(t, map[string]interface{}{
		"nodes/pve/qemu/100/agent/network-get-interfaces": interfaces,
	})
	defer server.Close()

	c := newTestClient(t, server)
	ifaces, err := c.GetVMInterfaces("pve", 100)
	if err != nil {
		t.Fatalf("GetVMInterfaces failed: %v", err)
	}
	if len(ifaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(ifaces))
	}
	if ifaces[1].Name != "eth0" {
		t.Errorf("expected eth0, got %s", ifaces[1].Name)
	}
	if ifaces[1].IPAddresses[0].Address != "192.168.2.100" {
		t.Errorf("expected 192.168.2.100, got %s", ifaces[1].IPAddresses[0].Address)
	}
}

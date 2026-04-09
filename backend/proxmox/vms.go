package proxmox

import (
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) ListVMs(node string) ([]VMStatus, error) {
	var vms []VMStatus
	err := c.get(fmt.Sprintf("nodes/%s/qemu", node), &vms)
	return vms, err
}

func (c *Client) GetVMStatus(node string, vmid int) (*VMStatus, error) {
	var status VMStatus
	err := c.get(fmt.Sprintf("nodes/%s/qemu/%d/status/current", node, vmid), &status)
	return &status, err
}

func (c *Client) StartVM(node string, vmid int) (string, error) {
	return c.post(fmt.Sprintf("nodes/%s/qemu/%d/status/start", node, vmid), nil)
}

func (c *Client) StopVM(node string, vmid int) (string, error) {
	return c.post(fmt.Sprintf("nodes/%s/qemu/%d/status/stop", node, vmid), nil)
}

func (c *Client) RebootVM(node string, vmid int) (string, error) {
	return c.post(fmt.Sprintf("nodes/%s/qemu/%d/status/reboot", node, vmid), nil)
}

func (c *Client) DeleteVM(node string, vmid int) (string, error) {
	return c.delete(fmt.Sprintf("nodes/%s/qemu/%d", node, vmid))
}

func (c *Client) CloneVM(node string, vmid int, newID int, name, target, storage string, full bool) (string, error) {
	params := map[string]string{
		"newid": fmt.Sprintf("%d", newID),
	}
	if name != "" {
		params["name"] = name
	}
	if target != "" {
		params["target"] = target
	}
	if storage != "" {
		params["storage"] = storage
	}
	if full {
		params["full"] = "1"
	}
	return c.postForm(fmt.Sprintf("nodes/%s/qemu/%d/clone", node, vmid), params)
}

// ConfigureCloudInit sets cloud-init user, password, SSH keys, and DNS domain on a VM
func (c *Client) ConfigureCloudInit(node string, vmid int, ciuser, password, sshkeys, searchdomain string) error {
	params := map[string]string{}
	if ciuser != "" {
		params["ciuser"] = ciuser
	}
	if password != "" {
		params["cipassword"] = password
	}
	if sshkeys != "" {
		// Proxmox requires sshkeys to be fully percent-encoded.
		// url.QueryEscape uses + for spaces; replace with %20 for Proxmox compat.
		encoded := url.QueryEscape(sshkeys)
		encoded = strings.ReplaceAll(encoded, "+", "%20")
		params["sshkeys"] = encoded
	}
	if searchdomain != "" {
		params["searchdomain"] = searchdomain
	}
	if len(params) == 0 {
		return nil
	}
	_, err := c.postForm(fmt.Sprintf("nodes/%s/qemu/%d/config", node, vmid), params)
	return err
}

// AddDisk adds a new disk to a VM (e.g. scsi1, virtio1, etc.)
// size is in GB, e.g. "50" creates a 50GB disk
func (c *Client) AddDisk(node string, vmid int, bus string, storage string, sizeGB int) error {
	// Proxmox format: scsi1=storage:sizeInGB
	params := map[string]string{
		bus: fmt.Sprintf("%s:%d", storage, sizeGB),
	}
	_, err := c.postForm(fmt.Sprintf("nodes/%s/qemu/%d/config", node, vmid), params)
	return err
}

// ResizeDisk resizes a disk on a VM
func (c *Client) ResizeDisk(node string, vmid int, disk string, size string) error {
	params := map[string]string{
		"disk": disk,
		"size": size,
	}
	_, err := c.putForm(fmt.Sprintf("nodes/%s/qemu/%d/resize", node, vmid), params)
	return err
}


// GetVMFilesystems retrieves filesystem usage from the QEMU guest agent
func (c *Client) GetVMFilesystems(node string, vmid int) ([]FilesystemInfo, error) {
	var result struct {
		Result []FilesystemInfo `json:"result"`
	}
	err := c.get(fmt.Sprintf("nodes/%s/qemu/%d/agent/get-fsinfo", node, vmid), &result)
	if err != nil {
		return nil, err
	}
	return result.Result, nil
}

// GetVMInterfaces retrieves network interfaces from the QEMU guest agent
func (c *Client) GetVMInterfaces(node string, vmid int) ([]NetworkInterface, error) {
	var result struct {
		Result []NetworkInterface `json:"result"`
	}
	err := c.get(fmt.Sprintf("nodes/%s/qemu/%d/agent/network-get-interfaces", node, vmid), &result)
	if err != nil {
		return nil, err
	}
	return result.Result, nil
}

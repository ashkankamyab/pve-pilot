package proxmox

import (
	"fmt"
	"net/url"
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

func (c *Client) CloneVM(node string, vmid int, newID int, name, target string, full bool) (string, error) {
	params := map[string]string{
		"newid": fmt.Sprintf("%d", newID),
	}
	if name != "" {
		params["name"] = name
	}
	if target != "" {
		params["target"] = target
	}
	if full {
		params["full"] = "1"
	}
	return c.postForm(fmt.Sprintf("nodes/%s/qemu/%d/clone", node, vmid), params)
}

// ConfigureCloudInit sets cloud-init password and SSH keys on a VM
func (c *Client) ConfigureCloudInit(node string, vmid int, password, sshkeys string) error {
	params := map[string]string{}
	if password != "" {
		params["cipassword"] = password
	}
	if sshkeys != "" {
		params["sshkeys"] = url.QueryEscape(sshkeys)
	}
	if len(params) == 0 {
		return nil
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

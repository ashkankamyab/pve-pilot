package proxmox

import "fmt"

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

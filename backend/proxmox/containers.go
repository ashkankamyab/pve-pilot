package proxmox

import "fmt"

func (c *Client) ListContainers(node string) ([]ContainerStatus, error) {
	var containers []ContainerStatus
	err := c.get(fmt.Sprintf("nodes/%s/lxc", node), &containers)
	return containers, err
}

func (c *Client) GetContainerStatus(node string, vmid int) (*ContainerStatus, error) {
	var status ContainerStatus
	err := c.get(fmt.Sprintf("nodes/%s/lxc/%d/status/current", node, vmid), &status)
	return &status, err
}

func (c *Client) StartContainer(node string, vmid int) (string, error) {
	return c.post(fmt.Sprintf("nodes/%s/lxc/%d/status/start", node, vmid), nil)
}

func (c *Client) StopContainer(node string, vmid int) (string, error) {
	return c.post(fmt.Sprintf("nodes/%s/lxc/%d/status/stop", node, vmid), nil)
}

func (c *Client) RebootContainer(node string, vmid int) (string, error) {
	return c.post(fmt.Sprintf("nodes/%s/lxc/%d/status/reboot", node, vmid), nil)
}

func (c *Client) CloneContainer(node string, vmid int, newID int, hostname, target string, full bool) (string, error) {
	params := map[string]string{
		"newid": fmt.Sprintf("%d", newID),
	}
	if hostname != "" {
		params["hostname"] = hostname
	}
	if target != "" {
		params["target"] = target
	}
	if full {
		params["full"] = "1"
	}
	return c.postForm(fmt.Sprintf("nodes/%s/lxc/%d/clone", node, vmid), params)
}

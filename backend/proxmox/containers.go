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

func (c *Client) DeleteContainer(node string, vmid int) (string, error) {
	return c.delete(fmt.Sprintf("nodes/%s/lxc/%d", node, vmid))
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

// ConfigureContainerCloudInit sets password and SSH keys on an LXC container
func (c *Client) ConfigureContainerCloudInit(node string, vmid int, password, sshkeys string) error {
	params := map[string]string{}
	if password != "" {
		params["password"] = password
	}
	if sshkeys != "" {
		params["ssh-public-keys"] = sshkeys
	}
	if len(params) == 0 {
		return nil
	}
	_, err := c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/config", node, vmid), params)
	return err
}

// ResizeContainerDisk resizes a disk on an LXC container
func (c *Client) ResizeContainerDisk(node string, vmid int, disk string, size string) error {
	params := map[string]string{
		"disk": disk,
		"size": size,
	}
	_, err := c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/resize", node, vmid), params)
	return err
}

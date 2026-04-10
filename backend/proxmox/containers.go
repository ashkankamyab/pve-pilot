package proxmox

import (
	"fmt"
)

// GetContainerConfig returns the full LXC container configuration.
func (c *Client) GetContainerConfig(node string, vmid int) (map[string]interface{}, error) {
	var cfg map[string]interface{}
	err := c.get(fmt.Sprintf("nodes/%s/lxc/%d/config", node, vmid), &cfg)
	return cfg, err
}

// GetContainerInterfaces returns network interfaces as reported by the running container.
func (c *Client) GetContainerInterfaces(node string, vmid int) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	err := c.get(fmt.Sprintf("nodes/%s/lxc/%d/interfaces", node, vmid), &result)
	return result, err
}

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

func (c *Client) CloneContainer(node string, vmid int, newID int, hostname, target, storage string, full bool) (string, error) {
	params := map[string]string{
		"newid": fmt.Sprintf("%d", newID),
	}
	if hostname != "" {
		params["hostname"] = hostname
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
	return c.postForm(fmt.Sprintf("nodes/%s/lxc/%d/clone", node, vmid), params)
}

// SetContainerResources updates cores/memory on an LXC container.
func (c *Client) SetContainerResources(node string, vmid int, cores, memoryMB int) error {
	params := map[string]string{}
	if cores > 0 {
		params["cores"] = fmt.Sprintf("%d", cores)
	}
	if memoryMB > 0 {
		params["memory"] = fmt.Sprintf("%d", memoryMB)
	}
	if len(params) == 0 {
		return nil
	}
	_, err := c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/config", node, vmid), params)
	return err
}

// ConfigureContainerHostname sets the hostname and other basic config on an LXC container.
// Note: Proxmox LXC /config endpoint does NOT accept password/ssh-public-keys — those are only
// valid during container creation (POST /lxc), not clone/update. For credentials on LXC,
// bake them into the template or use SSH to the Proxmox host to run pct exec.
func (c *Client) ConfigureContainerHostname(node string, vmid int, hostname string) error {
	if hostname == "" {
		return nil
	}
	params := map[string]string{
		"hostname": hostname,
	}
	_, err := c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/config", node, vmid), params)
	return err
}

// ResizeContainerDisk resizes a disk on an LXC container. Returns UPID.
func (c *Client) ResizeContainerDisk(node string, vmid int, disk string, size string) (string, error) {
	params := map[string]string{
		"disk": disk,
		"size": size,
	}
	return c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/resize", node, vmid), params)
}

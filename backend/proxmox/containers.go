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

// ConfigureContainerNetwork sets a static IP on an LXC container's net0.
// Reads the current net0 value, replaces ip= and gw= parts, writes it back.
func (c *Client) ConfigureContainerNetwork(node string, vmid int, ip, gateway string, subnet int) error {
	// Read current config to get net0
	cfg, err := c.GetContainerConfig(node, vmid)
	if err != nil {
		return fmt.Errorf("reading container config: %w", err)
	}

	net0, ok := cfg["net0"]
	if !ok {
		return fmt.Errorf("container has no net0 interface")
	}

	net0Str := fmt.Sprintf("%v", net0)

	// Parse net0 into key=value parts
	cidr := 24
	if subnet > 0 {
		cidr = subnet
	}

	// Remove existing ip= and gw= parts
	var parts []string
	for _, part := range splitNet0(net0Str) {
		key := part
		if idx := indexOf(part, '='); idx >= 0 {
			key = part[:idx]
		}
		if key != "ip" && key != "gw" {
			parts = append(parts, part)
		}
	}

	// Add new ip and gw
	parts = append(parts, fmt.Sprintf("ip=%s/%d", ip, cidr))
	if gateway != "" {
		parts = append(parts, fmt.Sprintf("gw=%s", gateway))
	}

	newNet0 := joinParts(parts)
	params := map[string]string{
		"net0": newNet0,
	}
	_, err = c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/config", node, vmid), params)
	return err
}

func splitNet0(s string) []string {
	var parts []string
	for _, p := range splitComma(s) {
		p = trimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitComma(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func indexOf(s string, ch byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}

// ResizeContainerDisk resizes a disk on an LXC container. Returns UPID.
func (c *Client) ResizeContainerDisk(node string, vmid int, disk string, size string) (string, error) {
	params := map[string]string{
		"disk": disk,
		"size": size,
	}
	return c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/resize", node, vmid), params)
}

// AddContainerMountPoint adds a new mountpoint to an LXC container.
// mpKey is e.g. "mp0", "mp1". Format: storage:sizeGB,mp=/mount/path
func (c *Client) AddContainerMountPoint(node string, vmid int, mpKey, storage string, sizeGB int, mountPath string) error {
	params := map[string]string{
		mpKey: fmt.Sprintf("%s:%d,mp=%s", storage, sizeGB, mountPath),
	}
	_, err := c.putForm(fmt.Sprintf("nodes/%s/lxc/%d/config", node, vmid), params)
	return err
}

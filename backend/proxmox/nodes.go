package proxmox

import "fmt"

func (c *Client) GetClusterResources() ([]ClusterResource, error) {
	var resources []ClusterResource
	err := c.get("cluster/resources", &resources)
	return resources, err
}

func (c *Client) GetClusterSummary() (*ClusterSummary, error) {
	resources, err := c.GetClusterResources()
	if err != nil {
		return nil, err
	}

	summary := &ClusterSummary{}
	for _, r := range resources {
		switch r.Type {
		case "node":
			summary.Nodes++
			if r.Status == "online" {
				summary.NodesOnline++
			}
			summary.CPUTotal += r.MaxCPU
			summary.CPUUsage += r.CPU
			summary.MemUsed += r.Mem
			summary.MemTotal += r.MaxMem
		case "qemu":
			if r.Template == 0 {
				summary.VMsTotal++
				if r.Status == "running" {
					summary.VMsRunning++
				}
			}
		case "lxc":
			if r.Template == 0 {
				summary.ContainersTotal++
				if r.Status == "running" {
					summary.ContainersRunning++
				}
			}
		case "storage":
			summary.DiskUsed += r.Disk
			summary.DiskTotal += r.MaxDisk
		}
	}

	if summary.Nodes > 0 {
		summary.CPUUsage = summary.CPUUsage / float64(summary.Nodes)
	}

	return summary, nil
}

func (c *Client) ListNodes() ([]ClusterResource, error) {
	resources, err := c.GetClusterResources()
	if err != nil {
		return nil, err
	}

	var nodes []ClusterResource
	for _, r := range resources {
		if r.Type == "node" {
			nodes = append(nodes, r)
		}
	}
	return nodes, nil
}

func (c *Client) GetNodeStatus(node string) (map[string]interface{}, error) {
	var status map[string]interface{}
	err := c.get(fmt.Sprintf("nodes/%s/status", node), &status)
	return status, err
}

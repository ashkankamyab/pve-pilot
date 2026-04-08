package proxmox

import "fmt"

func (c *Client) ListStorage(node string) ([]StorageInfo, error) {
	var storage []StorageInfo
	err := c.get(fmt.Sprintf("nodes/%s/storage", node), &storage)
	return storage, err
}

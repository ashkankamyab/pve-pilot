package proxmox

import (
	"fmt"
	"time"
)

func (c *Client) ListStorage(node string) ([]StorageInfo, error) {
	var storage []StorageInfo
	err := c.get(fmt.Sprintf("nodes/%s/storage", node), &storage)
	return storage, err
}

// GuestExec runs a command inside the VM via the QEMU guest agent.
// Proxmox agent/exec requires JSON body with "command" (the binary) and "input-data" (stdin).
func (c *Client) GuestExec(node string, vmid int, script string) error {
	body := map[string]interface{}{
		"command":    "bash",
		"input-data": script + "\n",
	}
	_, err := c.post(fmt.Sprintf("nodes/%s/qemu/%d/agent/exec", node, vmid), body)
	return err
}

// GuestExecWithRetry retries guest agent exec until success or timeout.
func (c *Client) GuestExecWithRetry(node string, vmid int, script string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = c.GuestExec(node, vmid, script)
		if lastErr == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("guest agent exec failed after %s: %v", timeout, lastErr)
}

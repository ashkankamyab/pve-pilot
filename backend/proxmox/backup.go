package proxmox

import "fmt"

// Backup creates a vzdump backup of a VM or container. Returns UPID.
func (c *Client) Backup(node string, vmid int, storage, mode, compress, notes string) (string, error) {
	params := map[string]string{
		"vmid":    fmt.Sprintf("%d", vmid),
		"storage": storage,
	}
	if mode != "" {
		params["mode"] = mode
	}
	if compress != "" {
		params["compress"] = compress
	}
	if notes != "" {
		params["notes-template"] = notes
	}
	return c.postForm(fmt.Sprintf("nodes/%s/vzdump", node), params)
}

// ListBackups lists all backup volumes on a storage.
func (c *Client) ListBackups(node, storage string) ([]BackupInfo, error) {
	var items []BackupInfo
	err := c.get(fmt.Sprintf("nodes/%s/storage/%s/content?content=backup", node, storage), &items)
	return items, err
}

// DeleteBackup removes a backup volume from storage.
func (c *Client) DeleteBackup(node, volid string) (string, error) {
	// volid contains the storage name, e.g. "nfs-drive:dump/vzdump-qemu-100-..."
	// Proxmox endpoint: DELETE /nodes/{node}/storage/{storage}/content/{volid}
	// But the simpler form works: DELETE /nodes/{node}/storage/{storage}/content/{volume}
	return c.delete(fmt.Sprintf("nodes/%s/storage/%s/content/%s", node, storageFromVolID(volid), volid))
}

// RestoreVM restores a QEMU VM from a backup archive. Returns UPID.
func (c *Client) RestoreVM(node, archive string, vmid int, storage string) (string, error) {
	params := map[string]string{
		"archive": archive,
	}
	if vmid > 0 {
		params["vmid"] = fmt.Sprintf("%d", vmid)
	}
	if storage != "" {
		params["storage"] = storage
	}
	return c.postForm(fmt.Sprintf("nodes/%s/qemu", node), params)
}

// RestoreContainer restores an LXC container from a backup archive. Returns UPID.
func (c *Client) RestoreContainer(node, archive string, vmid int, storage string) (string, error) {
	params := map[string]string{
		"ostemplate": archive,
		"restore":    "1",
	}
	if vmid > 0 {
		params["vmid"] = fmt.Sprintf("%d", vmid)
	}
	if storage != "" {
		params["storage"] = storage
	}
	return c.postForm(fmt.Sprintf("nodes/%s/lxc", node), params)
}

// ListBackupSchedules returns all cluster-level backup schedules.
func (c *Client) ListBackupSchedules() ([]BackupSchedule, error) {
	var schedules []BackupSchedule
	err := c.get("cluster/backup", &schedules)
	return schedules, err
}

// CreateBackupSchedule creates a new cluster backup schedule. Returns the schedule ID.
func (c *Client) CreateBackupSchedule(req BackupScheduleRequest) (string, error) {
	params := map[string]string{
		"storage":  req.Storage,
		"schedule": req.Schedule,
	}
	if req.VMID != "" {
		params["vmid"] = req.VMID
	}
	if req.Mode != "" {
		params["mode"] = req.Mode
	}
	if req.Compress != "" {
		params["compress"] = req.Compress
	}
	if req.Comment != "" {
		params["comment"] = req.Comment
	}
	if req.Node != "" {
		params["node"] = req.Node
	}
	if req.Enabled {
		params["enabled"] = "1"
	} else {
		params["enabled"] = "0"
	}
	return c.postForm("cluster/backup", params)
}

// DeleteBackupSchedule removes a cluster backup schedule.
func (c *Client) DeleteBackupSchedule(id string) error {
	_, err := c.delete(fmt.Sprintf("cluster/backup/%s", id))
	return err
}

// storageFromVolID extracts the storage name from a volid like "nfs-drive:dump/file.vma"
func storageFromVolID(volid string) string {
	for i, ch := range volid {
		if ch == ':' {
			return volid[:i]
		}
	}
	return volid
}

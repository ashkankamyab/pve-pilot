package proxmox

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// HostSSH provides SSH access to the Proxmox host for commands that can't be
// done via the REST API (e.g. pct exec for LXC credential injection).
type HostSSH struct {
	host    string // e.g. "192.168.2.100" or "proxmox.example.com"
	port    string // e.g. "22"
	user    string // e.g. "root"
	keyPath string // path to SSH private key
	config  *ssh.ClientConfig
}

// NewHostSSH creates an SSH client for the Proxmox host. Returns nil if any
// required config is missing — callers should check IsEnabled() before use.
func NewHostSSH(host, port, user, keyPath string) (*HostSSH, error) {
	if host == "" || user == "" || keyPath == "" {
		return nil, nil
	}
	if port == "" {
		port = "22"
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading SSH key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsing SSH key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // homelab context
		Timeout:         10 * time.Second,
	}

	return &HostSSH{
		host:    host,
		port:    port,
		user:    user,
		keyPath: keyPath,
		config:  cfg,
	}, nil
}

// IsEnabled reports whether SSH to host is configured.
func (h *HostSSH) IsEnabled() bool {
	return h != nil && h.config != nil
}

// Run executes a command on the Proxmox host via SSH and returns stdout/stderr.
func (h *HostSSH) Run(command string) (string, string, error) {
	if !h.IsEnabled() {
		return "", "", fmt.Errorf("host SSH not configured")
	}

	addr := fmt.Sprintf("%s:%s", h.host, h.port)
	client, err := ssh.Dial("tcp", addr, h.config)
	if err != nil {
		return "", "", fmt.Errorf("dialing %s: %w", addr, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("creating session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("command failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), stderr.String(), nil
}

// PctExec runs a bash script inside an LXC container using `pct exec`.
// The script is base64-encoded to avoid shell escaping issues.
func (h *HostSSH) PctExec(vmid int, script string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	cmd := fmt.Sprintf("pct exec %d -- bash -c 'echo %s | base64 -d | bash'", vmid, encoded)
	_, stderr, err := h.Run(cmd)
	if err != nil {
		return fmt.Errorf("pct exec %d: %w (stderr: %s)", vmid, err, stderr)
	}
	return nil
}

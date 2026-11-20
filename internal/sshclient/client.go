// Package sshclient provides an SSH transport for the router shell. It is the
// SSH counterpart of internal/telnet and is used by `zlt_xloader shell --ssh`.
package sshclient

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client wraps an authenticated SSH connection to the device.
type Client struct {
	conn *ssh.Client
}

// Dial connects to an SSH server using username/password authentication.
//
// Host keys are intentionally not verified: these embedded routers ship no CA
// and their host keys are not publicly known. The tool is expected to be used
// on a trusted, directly-attached management network.
func Dial(host string, port int, user, password string, timeout time.Duration) (*Client, error) {
	cfg := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // #nosec G106 -- embedded device, see doc comment
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return &Client{conn: conn}, nil
}

// Close terminates the SSH connection.
func (c *Client) Close() error { return c.conn.Close() }

// Run executes one command and returns its combined stdout/stderr.
func (c *Client) Run(command string, timeout time.Duration) (string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	timer := time.AfterFunc(timeout, func() { _ = session.Close() })
	defer timer.Stop()

	out, err := session.CombinedOutput(command)
	if err != nil && len(out) == 0 {
		return "", fmt.Errorf("ssh exec %q: %w", command, err)
	}
	return string(out), err
}

// Interactive opens a pty-backed shell and wires it to the local terminal.
func (c *Client) Interactive() error {
	session, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}
	if err := session.RequestPty("xterm", 40, 120, modes); err != nil {
		return fmt.Errorf("ssh pty: %w", err)
	}
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	if err := session.Shell(); err != nil {
		return fmt.Errorf("ssh shell: %w", err)
	}
	if err := session.Wait(); err != nil {
		return fmt.Errorf("ssh session ended: %w", err)
	}
	return nil
}

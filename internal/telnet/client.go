// Package telnet implements a minimal Telnet client used for router shell
// access. It handles IAC negotiation and provides command execution plus an
// interactive mode.
package telnet

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	iac  = 255
	dont = 254
	do   = 251
	wont = 252
	will = 253
	sb   = 250
	se   = 240
	nul  = 0
	cr   = 13
)

// Client is a Telnet session.
type Client struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
}

// Dial connects to a Telnet server.
func Dial(host string, port int, timeout time.Duration) (*Client, error) {
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	return &Client{conn: conn, timeout: timeout}, nil
}

// Close closes the session.
func (c *Client) Close() error { return c.conn.Close() }

// Run sends a command and returns all output until the shell prompt (or a
// quiet period). The caller should use a prompt common to BusyBox ash, such
// as "# ".
func (c *Client) Run(command string, prompt string, timeout time.Duration) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.conn.SetDeadline(time.Now().Add(timeout))
	if _, err := io.WriteString(c.conn, command+"\n"); err != nil {
		return "", err
	}

	var out strings.Builder
	buf := make([]byte, 4096)
	lastData := time.Now()
	for time.Since(lastData) < 500*time.Millisecond {
		n, err := c.conn.Read(buf)
		if n > 0 {
			lastData = time.Now()
			out.Write(filterIAC(buf[:n]))
			if prompt != "" && strings.Contains(out.String(), prompt) {
				break
			}
		}
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				break
			}
			if err == io.EOF {
				break
			}
			return out.String(), err
		}
	}
	return out.String(), nil
}

// Expect reads until the given substring appears or timeout elapses.
func (c *Client) Expect(substr string, timeout time.Duration) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	deadline := time.Now().Add(timeout)
	_ = c.conn.SetDeadline(deadline)
	var out strings.Builder
	buf := make([]byte, 4096)
	for time.Now().Before(deadline) {
		n, err := c.conn.Read(buf)
		if n > 0 {
			out.Write(filterIAC(buf[:n]))
			if substr == "" || strings.Contains(out.String(), substr) {
				return out.String(), nil
			}
		}
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				return out.String(), nil
			}
			if err == io.EOF {
				return out.String(), nil
			}
			return out.String(), err
		}
	}
	return out.String(), nil
}

// Interactive streams stdin/stdout to the session until the user exits.
func (c *Client) Interactive() error {
	_ = c.conn.SetDeadline(time.Time{})

	done := make(chan struct{}, 2)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if _, werr := c.conn.Write(buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := c.conn.Read(buf)
			if n > 0 {
				_, _ = os.Stdout.Write(filterIAC(buf[:n]))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
	return nil
}

// filterIAC removes Telnet negotiation sequences from a byte stream.
func filterIAC(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for i := 0; i < len(in); i++ {
		if in[i] != iac {
			if in[i] == cr && i+1 < len(in) && in[i+1] == nul {
				i++ // discard NUL after CR
			}
			out = append(out, in[i])
			continue
		}
		if i+1 >= len(in) {
			break
		}
		cmd := in[i+1]
		switch cmd {
		case will, do:
			// Decline the option. The embedded telnetd tolerates a silent
			// client; negotiation frames are simply hidden from output.
			if i+2 < len(in) {
				i += 2
			}
		case wont, dont:
			if i+2 < len(in) {
				i += 2
			}
		case sb:
			for i+1 < len(in) && !(in[i] == iac && in[i+1] == se) {
				i++
			}
			if i+1 < len(in) {
				i++
			}
		case iac:
			out = append(out, iac)
			i++
		}
		// IAC sequences are intentionally dropped from visible output.
	}
	return out
}

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

	log "github.com/sirupsen/logrus"
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
	pending []byte // incomplete IAC sequence carried between reads
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
	written, err := io.WriteString(c.conn, command+"\n")
	log.Debugf("[telnet] wrote %d bytes to %s", written, c.conn.RemoteAddr())
	if err != nil {
		return "", err
	}

	var out strings.Builder
	buf := make([]byte, 4096)
	lastData := time.Now()
	for time.Since(lastData) < 500*time.Millisecond {
		n, err := c.conn.Read(buf)
		if n > 0 {
			lastData = time.Now()
			chunk := c.filterIAC(buf[:n])
			log.Debugf("[telnet] raw chunk: %q", chunk)
			out.Write(chunk)
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
			chunk := c.filterIAC(buf[:n])
			log.Debugf("[telnet] expect chunk: %q", chunk)
			out.Write(chunk)
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
				_, _ = os.Stdout.Write(c.filterIAC(buf[:n]))
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

// filterIAC strips Telnet negotiation sequences, answers them, and carries
// incomplete sequences across reads so a split IAC frame is never lost.
func (c *Client) filterIAC(in []byte) []byte {
	data := append(c.pending, in...)
	c.pending = nil
	out := make([]byte, 0, len(data))

parse:
	for i := 0; i < len(data); i++ {
		if data[i] != iac {
			if data[i] == cr && i+1 < len(data) && data[i+1] == nul {
				i++ // discard NUL after CR
			}
			out = append(out, data[i])
			continue
		}
		if i+1 >= len(data) {
			c.pending = append(c.pending, data[i:]...)
			break
		}
		cmd := data[i+1]
		switch cmd {
		case will:
			if i+2 >= len(data) {
				c.pending = append(c.pending, data[i:]...)
				break parse
			}
			_, _ = c.conn.Write([]byte{iac, do, data[i+2]})
			i += 2
		case do:
			if i+2 >= len(data) {
				c.pending = append(c.pending, data[i:]...)
				break parse
			}
			_, _ = c.conn.Write([]byte{iac, will, data[i+2]})
			i += 2
		case wont, dont:
			if i+2 >= len(data) {
				c.pending = append(c.pending, data[i:]...)
				break parse
			}
			i += 2
		case sb:
			j := i + 2
			for j+1 < len(data) && !(data[j] == iac && data[j+1] == se) {
				j++
			}
			if j+1 >= len(data) {
				c.pending = append(c.pending, data[i:]...)
				break parse
			}
			i = j + 1
		case iac:
			out = append(out, iac)
			i++
		}
	}
	return out
}

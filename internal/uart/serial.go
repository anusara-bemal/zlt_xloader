// Package uart provides serial-port access and an AT command engine for
// direct hardware-level interaction with the Tozed chipset.
package uart

import (
	"fmt"
	"strings"
	"time"

	"go.bug.st/serial"
)

// Connection wraps a serial port with read helpers.
type Connection struct {
	port    serial.Port
	name    string
	timeout time.Duration
}

// ListPorts returns the available serial port names.
func ListPorts() ([]string, error) {
	return serial.GetPortsList()
}

// Open opens a serial port with the given parameters.
func Open(name string, baud, dataBits, stopBits int, parity string, timeout time.Duration) (*Connection, error) {
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}

	stop := serial.OneStopBit
	switch stopBits {
	case 1:
		stop = serial.OneStopBit
	case 2:
		stop = serial.TwoStopBits
	case 3:
		stop = serial.OnePointFiveStopBits
	}

	par := serial.NoParity
	switch strings.ToUpper(parity) {
	case "E", "EVEN":
		par = serial.EvenParity
	case "O", "ODD":
		par = serial.OddParity
	}

	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: dataBits,
		StopBits: stop,
		Parity:   par,
	}
	port, err := serial.Open(name, mode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	_ = port.SetReadTimeout(timeout)

	return &Connection{port: port, name: name, timeout: timeout}, nil
}

// Name returns the port name.
func (c *Connection) Name() string { return c.name }

// Close closes the serial port.
func (c *Connection) Close() error { return c.port.Close() }

// WriteString writes raw bytes to the port.
func (c *Connection) WriteString(s string) (int, error) {
	return c.port.Write([]byte(s))
}

// ReadUntil reads until expect is found or timeout elapses. An empty expect
// reads until the first read timeout.
func (c *Connection) ReadUntil(expect string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var sb strings.Builder
	buf := make([]byte, 256)

	for time.Now().Before(deadline) {
		n, err := c.port.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
			if expect == "" || strings.Contains(sb.String(), expect) {
				return sb.String(), nil
			}
		}
		if err != nil {
			// A read timeout simply means "no more data right now".
			time.Sleep(10 * time.Millisecond)
		}
	}
	return sb.String(), nil
}

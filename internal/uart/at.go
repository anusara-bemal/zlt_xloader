package uart

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/anusara-bemal/zlt-unlocker/internal/profile"
)

// Transport is the minimal serial interface the AT engine needs. It allows
// the engine to be unit tested without hardware.
type Transport interface {
	WriteString(string) (int, error)
	ReadUntil(string, time.Duration) (string, error)
}

// Engine sends AT commands and validates their responses.
type Engine struct {
	conn   Transport
	dryRun bool
}

// NewEngine creates an AT command engine. conn may be nil in dry-run mode.
func NewEngine(conn Transport, dryRun bool) *Engine {
	return &Engine{conn: conn, dryRun: dryRun}
}

// Send transmits one AT command and waits for the expected response token.
func (e *Engine) Send(command, expect string, timeout time.Duration) (string, error) {
	if e.dryRun {
		log.Infof("[uart] dry-run: would send %q (expect %q)", command, expect)
		return "", nil
	}
	if e.conn == nil {
		return "", fmt.Errorf("no serial connection (use --dry-run to preview)")
	}

	wire := strings.TrimRight(command, "\r\n") + "\r\n"
	if _, err := e.conn.WriteString(wire); err != nil {
		return "", fmt.Errorf("write %q: %w", command, err)
	}

	response, err := e.conn.ReadUntil(expect, timeout)
	if err != nil {
		return response, err
	}
	return response, nil
}

// Result is the outcome of one scripted AT command.
type Result struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	Response string `json:"response"`
	OK       bool   `json:"ok"`
	Err      error  `json:"error,omitempty"`
}

// Run executes a scripted AT command sequence.
func (e *Engine) Run(commands []profile.UARTCommand, timeout time.Duration) []Result {
	results := make([]Result, 0, len(commands))
	for _, cmd := range commands {
		resp, err := e.Send(cmd.Command, cmd.Expect, timeout)
		r := Result{
			Name:     cmd.Name,
			Command:  cmd.Command,
			Response: strings.TrimSpace(resp),
			Err:      err,
		}
		r.OK = err == nil && (e.dryRun || cmd.Expect == "" || strings.Contains(strings.ToUpper(resp), strings.ToUpper(cmd.Expect)))
		results = append(results, r)
		if r.OK {
			log.Infof("[uart] %-16s OK", cmd.Name)
		} else {
			log.Warnf("[uart] %-16s unexpected response: %q", cmd.Name, r.Response)
		}
	}
	return results
}

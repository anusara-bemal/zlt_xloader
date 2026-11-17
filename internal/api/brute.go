package api

import (
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/anusara-bemal/zlt-unlocker/internal/profile"
)

// Result is the outcome of one payload attempt.
type Result struct {
	Payload profile.Payload
	Body    string
	Hit     bool // response was not {"result":"failure"}
	DryRun  bool // no request was actually sent
	Err     error
}

// Try sends one payload outside the worker pool and returns its result.
func (c *Client) Try(p profile.Payload) Result {
	body, err := c.send(p)
	return Result{
		Payload: p,
		Body:    string(body),
		Hit:     err == nil && !IsFailure(body) && !IsDryRun(body),
		DryRun:  err == nil && IsDryRun(body),
		Err:     err,
	}
}

// RunPayloads fires every payload through a bounded worker pool and returns
// all responses that are not the canonical failure response.
func (c *Client) RunPayloads(payloads []profile.Payload, workers int, rateLimit time.Duration) []Result {
	if workers < 1 {
		workers = 16
	}
	if workers > 256 {
		workers = 256
	}

	jobs := make(chan profile.Payload)
	out := make(chan Result, len(payloads))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				body, err := c.send(p)
				r := Result{
					Payload: p,
					Body:    string(body),
					Hit:     err == nil && !IsFailure(body) && !IsDryRun(body),
					DryRun:  err == nil && IsDryRun(body),
					Err:     err,
				}
				out <- r
				if rateLimit > 0 {
					time.Sleep(rateLimit)
				}
			}
		}()
	}

	for _, p := range payloads {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	close(out)

	hits := make([]Result, 0)
	for r := range out {
		switch {
		case r.Err != nil:
			log.WithError(r.Err).Warnf("[sweep] %s: request failed", r.Payload.ID)
		case r.DryRun:
			log.Debugf("[sweep] %s: dry-run (not sent)", r.Payload.ID)
		case r.Hit:
			log.Warnf("[sweep] HIT %-18s %s", r.Payload.ID, truncate(r.Body, 200))
			hits = append(hits, r)
		default:
			log.Debugf("[sweep] %s: failure", r.Payload.ID)
		}
	}
	return hits
}

func (c *Client) send(p profile.Payload) ([]byte, error) {
	if c.dryRun {
		return []byte(dryRunResult), nil
	}
	if strings.EqualFold(p.Method, "GET") {
		q := url.Values{}
		q.Set("goformId", p.ID)
		for k, v := range p.Args {
			q.Set(k, v)
		}
		resp, err := c.getRaw(c.ep.GetCmdPath + "?" + q.Encode())
		if err != nil {
			return nil, err
		}
		return []byte(resp), nil
	}
	return c.post(p.ID, toValues(p.Args))
}

func toValues(args map[string]string) url.Values {
	v := url.Values{}
	for k, value := range args {
		v.Set(k, value)
	}
	return v
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

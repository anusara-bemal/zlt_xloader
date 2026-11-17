// Package api implements the Tozed web-API client, login/CSRF bypass and
// payload sweep for S20_1.12.7.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/anusara-bemal/zlt-unlocker/internal/profile"
)

// dryRunResult is returned by mutating calls when dry-run is enabled. It is
// deliberately neither "success" nor "failure" so callers can label hits.
const dryRunResult = `{"result":"dry-run"}`

// Client is a stateful HTTP client for the router web API. A cookie jar is
// required because the token/session flow is session-bound.
type Client struct {
	base     string
	http     *http.Client
	ep       profile.Endpoints
	token    string // CSRFToken value returned by cmd=get_token
	random   string // random_login challenge
	dryRun   bool
	username string
	password string
}

// NewClient creates a router API client.
func NewClient(target string, timeout time.Duration, dryRun bool, ep profile.Endpoints) *Client {
	jar, _ := cookiejar.New(nil) // error is impossible with nil options
	base := strings.TrimSpace(target)
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	return &Client{
		base:   strings.TrimRight(base, "/"),
		http:   &http.Client{Jar: jar, Timeout: timeout},
		ep:     ep,
		dryRun: dryRun,
	}
}

// SetCredentials stores the credentials used by Login when none are supplied.
func (c *Client) SetCredentials(username, password string) {
	c.username, c.password = username, password
}

// Token returns the current CSRFToken.
func (c *Client) Token() string { return c.token }

// Random returns the random_login challenge from the last login.
func (c *Client) Random() string { return c.random }

// DryRun reports whether mutating requests are suppressed.
func (c *Client) DryRun() bool { return c.dryRun }

func (c *Client) csrfField() string { return c.ep.CSRFTokenField() }

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) zlt-unlocker")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", c.base+"/")
	if c.token != "" {
		// service.js sends CSRFToken in form data; the header is added as well
		// because some firmware revisions read it from there.
		req.Header.Set(c.csrfField(), c.token)
		req.Header.Set("X-CSRF-Token", c.token)
	}
	return req, nil
}

// postForm sends a urlencoded form to an arbitrary endpoint. In dry-run mode
// nothing is transmitted.
func (c *Client) postForm(path string, form url.Values) ([]byte, error) {
	if c.dryRun {
		return []byte(dryRunResult), nil
	}

	req, err := c.newRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// PostRaw sends an arbitrary body to an arbitrary endpoint with the given
// Content-Type. Used by the transport-bypass probes.
func (c *Client) PostRaw(path, contentType string, body []byte) ([]byte, error) {
	if c.dryRun {
		return []byte(dryRunResult), nil
	}
	req, err := c.newRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetRawQuery fetches an arbitrary path including a query string.
func (c *Client) GetRawQuery(path string) ([]byte, error) {
	if c.dryRun {
		return []byte(dryRunResult), nil
	}
	resp, err := c.http.Get(c.base + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// post sends a goformId command to /reqproc/proc_post. The CSRFToken is
// included in the form data exactly as service.js does.
func (c *Client) post(id string, args url.Values) ([]byte, error) {
	form := url.Values{}
	if id != "" {
		form.Set("goformId", id)
	}
	for k, values := range args {
		for _, v := range values {
			form.Add(k, v)
		}
	}
	if c.token != "" {
		form.Set(c.csrfField(), c.token)
	}
	return c.postForm(c.ep.EffectivePostPath(), form)
}

// Get queries a command through /reqproc/proc_get.
func (c *Client) Get(cmd string) (string, error) { return c.GetCmd(cmd, nil) }

// GetCmd queries a command with additional parameters.
func (c *Client) GetCmd(cmd string, extra url.Values) (string, error) {
	q := url.Values{}
	q.Set("cmd", cmd)
	// The firmware rejects commands without isTest and answers with a
	// malformed HTTP/0.9 response containing {"result":"invalid cmd isTest"}.
	q.Set("isTest", "false")
	for k, values := range extra {
		for _, v := range values {
			q.Add(k, v)
		}
	}
	q.Set("_", fmt.Sprintf("%d", time.Now().UnixNano()))
	return c.getRaw(c.ep.EffectiveGetPath() + "?" + q.Encode())
}

// GetRaw fetches an arbitrary path (e.g. "/" for HTML fingerprinting).
func (c *Client) GetRaw(path string) (string, error) { return c.getRaw(path) }

func (c *Client) getRaw(path string) (string, error) {
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// IsFailure reports the canonical Tozed rejection response.
func IsFailure(body []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return false // unexpected body: surfaced to the operator
	}
	result, _ := m["result"].(string)
	return strings.EqualFold(result, "failure")
}

// IsDryRun reports the synthetic dry-run response.
func IsDryRun(body []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	result, _ := m["result"].(string)
	return strings.EqualFold(result, "dry-run")
}

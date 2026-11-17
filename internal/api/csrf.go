package api

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Login performs the exact S20_1.12.7 login flow recovered from service.js:
//
//  1. GET  /reqproc/proc_get?cmd=get_random_login  -> {"random_login":"..."}
//  2. POST /reqproc/proc_post
//     goformId=LOGIN
//     isTest=false
//     username=Base64(username)
//     password=Base64(hex(SHA256(random_login + password)))
//     CSRFToken=            (empty for LOGIN)
//  3. GET  /reqproc/proc_get?cmd=get_token         -> {"token":"..."}
//
// get_token MUST NOT be called before login on this firmware: it returns a
// malformed HTTP status error. The token obtained in step 3 is attached to
// all later POSTs (e.g. TZ_CMD_SECURE_LOGIN) by Client.post.
func (c *Client) Login(username, password string) error {
	if username == "" {
		username = c.username
	}
	if password == "" {
		password = c.password
	}
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin"
	}
	c.username, c.password = username, password
	// Never reuse a token from an earlier session for LOGIN.
	c.token = ""

	if c.dryRun {
		c.token = "dry-run"
		log.Info("[login] dry-run: no requests were sent to the device")
		return nil
	}

	body, err := c.loginOnce(username, password)
	if err != nil {
		return err
	}
	if isSessionConflict(body) {
		log.Warn("[login] an active session exists; logging out and retrying")
		c.Logout()
		body, err = c.loginOnce(username, password)
		if err != nil {
			return err
		}
	}
	if err := checkLoginResult(body); err != nil {
		return err
	}
	log.Info("[login] LOGIN accepted")

	// Step 3: only now is get_token allowed. Its value is required by all
	// subsequent state-changing requests, including TZ_CMD_SECURE_LOGIN.
	tokenRaw, err := c.GetCmd(c.ep.GetTokenCmd, nil)
	if err != nil {
		return fmt.Errorf("get_token after login: %w", err)
	}
	c.token = jsonString(tokenRaw, "token")
	if c.token == "" {
		// Some firmware builds return an empty token even after a successful
		// login. The REMOVE_WHITE_SITE injection still works with an empty
		// token, so warn instead of aborting.
		log.Warn("[login] get_token returned an empty token; continuing without one")
	} else {
		log.Infof("[login] CSRFToken acquired: %s", shortToken(c.token))
	}
	return nil
}

// CSRFBypass is kept as the compatibility entry point: it performs the full
// verified login flow using credentials from the profile.
func (c *Client) CSRFBypass() error { return c.Login("", "") }

// TokenSequence runs the alternate set_random -> get_random -> set_token flow
// described in some Tozed deployments, then refreshes get_token. This is not
// used by service.js for LOGIN, so it is exposed separately.
func (c *Client) TokenSequence() error {
	if c.dryRun {
		log.Info("[csrf] dry-run: token sequence skipped")
		c.token = "dry-run"
		return nil
	}

	clientRandom := fmt.Sprintf("%d", time.Now().UnixNano())
	log.Info("[csrf] 1/3 set_random")
	if _, err := c.GetCmd(c.ep.SetRandomCmd, url.Values{"random": {clientRandom}}); err != nil {
		return fmt.Errorf("set_random: %w", err)
	}

	log.Info("[csrf] 2/3 get_random")
	raw, err := c.GetCmd(c.ep.GetRandomCmd, nil)
	if err != nil {
		return fmt.Errorf("get_random: %w", err)
	}
	serverRandom := jsonString(raw, "random")
	if serverRandom == "" {
		serverRandom = strings.TrimSpace(raw)
	}

	// Legacy derivation retained for older profiles (md5 of random + salt).
	sum := md5.Sum([]byte(serverRandom + c.ep.TokenSalt))
	token := hex.EncodeToString(sum[:])

	log.Infof("[csrf] 3/3 set_token (token=%s…)", shortToken(token))
	if _, err := c.GetCmd(c.ep.SetTokenCmd, url.Values{"token": {token}}); err != nil {
		return fmt.Errorf("set_token: %w", err)
	}
	if refreshed, err := c.GetCmd(c.ep.GetTokenCmd, nil); err == nil {
		if t := jsonString(refreshed, "token"); t != "" {
			c.token = t
		}
	}
	return nil
}

// loginOnce performs one challenge/hash/LOGIN exchange.
func (c *Client) loginOnce(username, password string) ([]byte, error) {
	// Step 1: random_login challenge. get_token is deliberately skipped here.
	challengeRaw, err := c.GetCmd(c.ep.GetRandomLoginCmd, nil)
	if err != nil {
		return nil, fmt.Errorf("get_random_login: %w", err)
	}
	c.random = jsonString(challengeRaw, "random_login", "random")
	if c.random == "" {
		return nil, fmt.Errorf("get_random_login returned no random_login: %s", truncate(challengeRaw, 200))
	}
	log.Infof("[login] random_login challenge: %s", shortToken(c.random))

	// Step 2: the firmware's sha256() yields a hex string, which is then
	// Base64-encoded before being sent as the password field.
	sum := sha256.Sum256([]byte(c.random + password))
	encoded := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(sum[:])))

	form := url.Values{}
	form.Set("goformId", "LOGIN")
	form.Set("isTest", "false")
	form.Set("username", base64.StdEncoding.EncodeToString([]byte(username)))
	form.Set("password", encoded)
	// service.js sends an empty CSRFToken with the LOGIN request.
	form.Set(c.csrfField(), "")
	return c.postForm(c.ep.EffectivePostPath(), form)
}

// Logout ends the current session so a fresh login can be attempted.
func (c *Client) Logout() {
	// The active session's token is required for LOGOUT to take effect.
	if raw, err := c.GetCmd(c.ep.GetTokenCmd, nil); err == nil {
		if token := jsonString(raw, "token"); token != "" {
			c.token = token
		}
	}
	form := url.Values{}
	form.Set("goformId", "LOGOUT")
	form.Set("isTest", "false")
	form.Set(c.csrfField(), c.token)
	_, _ = c.postForm(c.ep.EffectivePostPath(), form)
	c.token = ""
}

func isSessionConflict(body []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	result, _ := m["result"].(string)
	return strings.Contains(strings.ToLower(result), "illegal request for csrftoken")
}

// checkLoginResult interprets the numeric result codes used by service.js:
// "0" and "4" mean success; 1 login fail, 2 duplicate user, 3 bad password,
// 5 user not found.
func checkLoginResult(body []byte) error {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		// Non-JSON body: leave interpretation to the operator.
		return nil
	}
	if _, ok := m["result"]; !ok {
		return nil
	}
	result := strings.TrimSpace(fmt.Sprint(m["result"]))
	switch result {
	case "0", "4", "success", "Success":
		return nil
	case "1":
		return fmt.Errorf("LOGIN rejected: login fail (result=1)")
	case "2":
		return fmt.Errorf("LOGIN rejected: duplicate user (result=2)")
	case "3":
		return fmt.Errorf("LOGIN rejected: bad password (result=3)")
	case "5":
		return fmt.Errorf("LOGIN rejected: user does not exist (result=5)")
	default:
		return fmt.Errorf("LOGIN rejected: %s", truncate(string(body), 200))
	}
}

func jsonString(raw string, keys ...string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	for k, v := range m {
		for _, want := range keys {
			if strings.EqualFold(k, want) {
				if s, ok := v.(string); ok {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return ""
}

func shortToken(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

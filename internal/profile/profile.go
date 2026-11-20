// Package profile loads the device-specific configuration that drives every
// module. Keeping endpoints, goform IDs, AT commands and crypto constants in a
// JSON profile means they can be tuned from a packet capture without
// recompiling the binary.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Endpoints describes the web-API paths and CSRF command identifiers used by
// S20_1.12.7. The names were recovered from the router's service.js dump.
type Endpoints struct {
	// Verified endpoints / commands.
	PostPath          string `json:"post_path"`            // /reqproc/proc_post
	GetCmdPath        string `json:"get_cmd_path"`         // /reqproc/proc_get
	GetTokenCmd       string `json:"get_token_cmd"`        // get_token
	GetRandomLoginCmd string `json:"get_random_login_cmd"` // get_random_login
	SetRandomCmd      string `json:"set_random_cmd"`       // set_random
	GetRandomCmd      string `json:"get_random_cmd"`       // get_random
	SetTokenCmd       string `json:"set_token_cmd"`        // set_token
	CSRFTokenHeader   string `json:"csrf_token_header"`    // CSRFToken

	// Legacy aliases kept so older profiles keep working.
	SetCmdPath  string `json:"set_cmd_path"`
	SetRandomID string `json:"set_random_id"`
	SetTokenID  string `json:"set_token_id"`
	TokenSalt   string `json:"token_salt"`
}

// EffectivePostPath returns the configured write endpoint.
func (e Endpoints) EffectivePostPath() string {
	if strings.TrimSpace(e.PostPath) != "" {
		return e.PostPath
	}
	if strings.TrimSpace(e.SetCmdPath) != "" {
		return e.SetCmdPath
	}
	return "/reqproc/proc_post"
}

// EffectiveGetPath returns the configured read endpoint.
func (e Endpoints) EffectiveGetPath() string {
	if strings.TrimSpace(e.GetCmdPath) != "" {
		return e.GetCmdPath
	}
	return "/reqproc/proc_get"
}

// CSRFTokenField returns the form field name used for the CSRF token.
func (e Endpoints) CSRFTokenField() string {
	if strings.TrimSpace(e.CSRFTokenHeader) != "" {
		return e.CSRFTokenHeader
	}
	return "CSRFToken"
}

// Payload is one candidate request fired by the api-exploit module.
type Payload struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	Description string            `json:"description"`
	Args        map[string]string `json:"args"`
}

// Fingerprint controls firmware/model detection.
type Fingerprint struct {
	Cmd         string `json:"cmd"`
	FirmwareKey string `json:"firmware_key"`
	ModelKey    string `json:"model_key"`
	ACSKey      string `json:"acs_key"`
}

// Login holds the credentials used for the LOGIN goformId.
type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Fuzz controls smart payload generation.
type Fuzz struct {
	Keywords    []string `json:"keywords"`
	GoformIDs   []string `json:"goform_ids"`
	Values      []string `json:"values"`
	MaxRequests int      `json:"max_requests"`
}

// UARTCommand is a single AT command in a scripted sequence.
type UARTCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Expect  string `json:"expect"`
}

// UART holds serial-port defaults and the AT command script.
type UART struct {
	Port     string        `json:"port"`
	Baud     int           `json:"baud"`
	DataBits int           `json:"data_bits"`
	StopBits int           `json:"stop_bits"`
	Parity   string        `json:"parity"`
	Commands []UARTCommand `json:"commands"`
}

// Crypto holds heuristic decryption constants.
type Crypto struct {
	KnownKeys   []string `json:"known_keys"`
	HeaderMagic []string `json:"header_magic"`
	KeyLengths  []int    `json:"key_lengths"`
}

// UnlockParam is one NVRAM or AT parameter change used to remove a SIM/network
// lock. Path names the default-parameter file to patch for persistence.
type UnlockParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Path  string `json:"path"`
}

// Unlock holds variant-specific lock-removal data. It is profile-driven so the
// same binary can handle the SLT/Dialog, MTN and ORG firmware variants.
type Unlock struct {
	BackupPath string        `json:"backup_path"`
	NV         []UnlockParam `json:"nv"`
	AT         []UnlockParam `json:"at"`
	Save       bool          `json:"save"`
}

// Profile is the complete device profile.
type Profile struct {
	Name        string      `json:"name"`
	Target      string      `json:"target"`
	Endpoints   Endpoints   `json:"endpoints"`
	Fingerprint Fingerprint `json:"fingerprint"`
	Login       Login       `json:"login"`
	Payloads    []Payload   `json:"payloads"`
	Fuzz        Fuzz        `json:"fuzz"`
	UART        UART        `json:"uart"`
	Crypto      Crypto      `json:"crypto"`
	Unlock      Unlock      `json:"unlock"`
}

// Load reads and validates a JSON profile from disk.
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile %q: %w", path, err)
	}
	return parse(data)
}

// Builtin returns a safe minimal profile used when no file can be loaded.
func Builtin() *Profile {
	p := &Profile{}
	p.applyDefaults()
	return p
}

func parse(data []byte) (*Profile, error) {
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	p.applyDefaults()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Profile) applyDefaults() {
	if p.Name == "" {
		p.Name = "generic-tozed-s20"
	}
	if p.Target == "" {
		p.Target = "192.168.1.1"
	}
	if p.Endpoints.EffectiveGetPath() == "" {
		p.Endpoints.GetCmdPath = "/reqproc/proc_get"
	}
	if p.Endpoints.GetTokenCmd == "" {
		p.Endpoints.GetTokenCmd = "get_token"
	}
	if p.Endpoints.GetRandomLoginCmd == "" {
		p.Endpoints.GetRandomLoginCmd = "get_random_login"
	}
	if p.Endpoints.SetRandomCmd == "" {
		p.Endpoints.SetRandomCmd = "set_random"
	}
	if p.Endpoints.GetRandomCmd == "" {
		p.Endpoints.GetRandomCmd = "get_random"
	}
	if p.Endpoints.SetTokenCmd == "" {
		p.Endpoints.SetTokenCmd = "set_token"
	}
	if p.Endpoints.CSRFTokenHeader == "" {
		p.Endpoints.CSRFTokenHeader = "CSRFToken"
	}
	if p.Fingerprint.FirmwareKey == "" {
		p.Fingerprint.FirmwareKey = "sw_version"
	}
	if p.Fingerprint.ModelKey == "" {
		p.Fingerprint.ModelKey = "model"
	}
	if p.Fingerprint.ACSKey == "" {
		p.Fingerprint.ACSKey = "acs_url"
	}
	if p.Login.Username == "" {
		p.Login.Username = "admin"
	}
	if p.Login.Password == "" {
		p.Login.Password = "admin"
	}
	if p.UART.Baud == 0 {
		p.UART.Baud = 115200
	}
	if p.UART.DataBits == 0 {
		p.UART.DataBits = 8
	}
	if p.UART.StopBits == 0 {
		p.UART.StopBits = 1
	}
	if p.UART.Parity == "" {
		p.UART.Parity = "N"
	}
	if p.Unlock.BackupPath == "" {
		p.Unlock.BackupPath = "/tmp/zlt-unlocker-nv-backup.txt"
	}
}

// Validate checks that required fields are present.
func (p *Profile) Validate() error {
	if p.Endpoints.EffectivePostPath() == "" || p.Endpoints.EffectiveGetPath() == "" {
		return fmt.Errorf("profile: endpoint paths must not be empty")
	}
	return nil
}

package api

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/anusara-bemal/zlt-unlocker/internal/profile"
)

// Info is the detected device identity.
type Info struct {
	Firmware string         `json:"firmware"`
	Model    string         `json:"model"`
	ACSURL   string         `json:"acs_url"`
	Raw      map[string]any `json:"raw,omitempty"`
}

var (
	firmwareRE = regexp.MustCompile(`(?i)\bS\d+_\d+(?:\.\d+)+\b`)
	modelRE    = regexp.MustCompile(`(?i)\bZLT[\s_-]*(?:S|Z)\d+[A-Za-z]*\b`)
)

// Fingerprint queries the device-info command first, then falls back to
// scraping the web UI landing page if the JSON command is unavailable.
func (c *Client) Fingerprint(fp profile.Fingerprint) (*Info, error) {
	info := &Info{}

	if fp.Cmd != "" {
		if resp, err := c.Get(fp.Cmd); err == nil {
			var m map[string]any
			if json.Unmarshal([]byte(resp), &m) == nil {
				info.Raw = m
				info.Firmware = strVal(m, fp.FirmwareKey)
				info.Model = strVal(m, fp.ModelKey)
				info.ACSURL = strVal(m, fp.ACSKey)
			}
		}
	}

	if info.Firmware == "" || info.Model == "" {
		if html, err := c.GetRaw("/"); err == nil {
			if info.Firmware == "" {
				info.Firmware = firmwareRE.FindString(html)
			}
			if info.Model == "" {
				info.Model = modelRE.FindString(html)
			}
		}
	}

	return info, nil
}

func strVal(m map[string]any, key string) string {
	if key == "" || m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s := scalarString(v); s != "" {
			return s
		}
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return scalarString(v)
		}
	}
	return ""
}

func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return ""
	}
}

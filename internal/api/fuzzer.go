package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
)

// FuzzConfig controls smart payload generation. All fields are overridable
// from the JSON profile so new goformIds can be added without recompiling.
type FuzzConfig struct {
	Keywords    []string `json:"keywords"`
	GoformIDs   []string `json:"goform_ids"`
	Values      []string `json:"values"`
	MaxRequests int      `json:"max_requests"`
}

// DefaultFuzzConfig seeds the fuzzer with the PLMN / SIM / unlock / NVRAM
// vocabulary, plus the real goformIds recovered from service.js.
func DefaultFuzzConfig() FuzzConfig {
	return FuzzConfig{
		Keywords: []string{"plmn", "lock", "sim", "unlock", "nv_write"},
		GoformIDs: []string{
			"TZ_UNLOCK_PLMN", "UNLOCK_PLMN_PSK", "UNLOCK_NETWORK",
			"TZ_SET_UNLOCK_CODE", "TZ_GET_LOCK_PLMN", "TZ_SET_LOCK_PLMN",
			"TZ_GET_LOCK_BAND", "TZ_SET_LOCK_BAND", "CLEAR_IMSI",
			"SET_SIM_SETTING", "SET_NETWORK", "SET_PLMN", "SET_SYSTEM_INFO",
			"TZ_CMD_SECURE_LOGIN", "SET_WEB_LANGUAGE",
		},
		Values:      []string{"0", "1", "false", "true", "y", "n"},
		MaxRequests: 400,
	}
}

func (fc FuzzConfig) withDefaults() FuzzConfig {
	def := DefaultFuzzConfig()
	if len(fc.Keywords) == 0 {
		fc.Keywords = def.Keywords
	}
	if len(fc.GoformIDs) == 0 {
		fc.GoformIDs = def.GoformIDs
	}
	if len(fc.Values) == 0 {
		fc.Values = def.Values
	}
	if fc.MaxRequests <= 0 {
		fc.MaxRequests = def.MaxRequests
	}
	return fc
}

var fuzzPrefixes = []string{"SET_", "UNLOCK_", "TZ_UNLOCK_", "TZ_SET_", "GET_"}

var fuzzTriggers = []string{
	"SET_NETWORK", "SET_PLMN", "SET_SYSTEM_INFO", "SET_WEB_LANGUAGE",
	"TZ_CMD_SECURE_LOGIN", "SET_SIM_SETTING", "APN_PROC_EX",
	"TZ_UNLOCK_PLMN", "CLEAR_IMSI",
}

// GenerateFuzzPayloads builds a bounded list of candidate goformId/argument
// combinations. It explores:
//  1. known unlock-related goformIds with no arguments,
//  2. goformIds derived from keyword prefixes,
//  3. keyword-named parameters swept across known trigger goformIds,
//  4. compound SET_<A>_<B> goformIds.
func GenerateFuzzPayloads(cfg FuzzConfig) []profile.Payload {
	cfg = cfg.withDefaults()
	seen := make(map[string]struct{})
	out := make([]profile.Payload, 0)

	add := func(id, description string, args map[string]string) bool {
		if args == nil {
			args = map[string]string{}
		} else {
			copied := make(map[string]string, len(args)+1)
			for k, v := range args {
				copied[k] = v
			}
			args = copied
		}
		args["isTest"] = "false"
		key := payloadKey(id, args)
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
		out = append(out, profile.Payload{
			ID: id, Method: "POST", Description: description, Args: args,
		})
		return true
	}

	for _, id := range cfg.GoformIDs {
		add(id, "fuzz: known unlock-related goformId", nil)
	}

	for _, keyword := range cfg.Keywords {
		upper := strings.ToUpper(keyword)
		for _, prefix := range fuzzPrefixes {
			add(prefix+upper, "fuzz: keyword-derived goformId", nil)
		}
	}

	for _, trigger := range fuzzTriggers {
		for _, keyword := range cfg.Keywords {
			for _, value := range cfg.Values {
				add(trigger, fmt.Sprintf("fuzz: %s=%s on %s", keyword, value, trigger),
					map[string]string{keyword: value})
			}
		}
	}

	for i, a := range cfg.Keywords {
		for j, b := range cfg.Keywords {
			if i == j {
				continue
			}
			add("SET_"+strings.ToUpper(a)+"_"+strings.ToUpper(b), "fuzz: compound goformId", nil)
		}
	}

	if len(out) > cfg.MaxRequests {
		out = out[:cfg.MaxRequests]
	}
	return out
}

// Fuzz generates and sweeps payloads in one call.
func (c *Client) Fuzz(cfg FuzzConfig, workers int, rateLimit time.Duration) []Result {
	payloads := GenerateFuzzPayloads(cfg)
	log.Warnf("[fuzz] generated %d candidate payloads (NVRAM writes may brick the device)", len(payloads))
	return c.RunPayloads(payloads, workers, rateLimit)
}

func payloadKey(id string, args map[string]string) string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(id)
	for _, k := range keys {
		b.WriteString("|")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(args[k])
	}
	return b.String()
}

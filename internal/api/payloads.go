package api

import "github.com/anusara-bemal/zlt-unlocker/internal/profile"

// BuiltinPayloads is the S20_1.12.7 goform corpus. Entries are split into two
// groups:
//
//   - "verified" IDs were recovered verbatim from the router's service.js dump.
//   - "reported" IDs were supplied from black-box testing and are marked as
//     such; they are still worth trying but may be ignored by this firmware.
//
// The JSON profile is merged on top of this list, so operators can add or
// override payloads without recompiling.
func BuiltinPayloads() []profile.Payload {
	return []profile.Payload{
		// ---- Priority payloads (run first in the sweep) ----
		{ID: "SET_SIM_SETTING", Method: "POST", Description: "priority: clear SIM lock", Args: map[string]string{
			"isTest": "false", "sim_lock": "0",
		}},
		{ID: "TZ_CMD_SECURE_LOGIN", Method: "POST", Description: "priority: enable telnet/ADB", Args: map[string]string{
			"isTest": "false", "telnetdEnable": "y", "adbEnable": "y",
		}},

		// ---- Reported payloads (exact parameter sets supplied for S20_1.12.7) ----
		{ID: "SET_NETWORK", Method: "POST", Description: "reported: network selection / PLMN unlock", Args: map[string]string{
			"isTest": "false", "NetworkNumber": "", "Rat": "0", "nSubrat": "",
		}},
		{ID: "SET_PLMN", Method: "POST", Description: "reported: PLMN lock clear (ID not present in service.js)", Args: map[string]string{
			"isTest": "false", "plmn_lock": "0",
		}},
		{ID: "CONNECT_NETWORK", Method: "POST", Description: "reported: connect network", Args: map[string]string{
			"isTest": "false", "disconnect_internet": "0",
		}},
		{ID: "APN_PROC_EX", Method: "POST", Description: "reported: reset APN profile 1 to default", Args: map[string]string{
			"isTest": "false", "apn_action": "set_default", "index": "1",
		}},
		{ID: "SET_WEB_LANGUAGE", Method: "POST", Description: "reported: language + ADB toggle", Args: map[string]string{
			"isTest": "false", "Language": "en", "adbd_enable": "1",
		}},
		{ID: "SET_SYSTEM_INFO", Method: "POST", Description: "reported: telnetd toggle (ID not present in service.js)", Args: map[string]string{
			"isTest": "false", "telnetd_enable": "1",
		}},
		{ID: "RESTORE_FACTORY_SETTINGS", Method: "POST", Description: "reported: factory reset — DESTRUCTIVE", Args: map[string]string{
			"isTest": "false",
		}},

		// ---- Verified from service.js ----
		{ID: "TZ_UNLOCK_PLMN", Method: "POST", Description: "verified: getUnlockPlmn() trigger", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "UNLOCK_PLMN_PSK", Method: "POST", Description: "verified: loginLock() with lockCode", Args: map[string]string{
			"isTest": "false", "lockCode": "",
		}},
		{ID: "UNLOCK_NETWORK", Method: "POST", Description: "verified: unlockNetwork() with NCK code", Args: map[string]string{
			"isTest": "false", "notCallback": "true", "unlock_network_code": "",
		}},
		{ID: "TZ_SET_UNLOCK_CODE", Method: "POST", Description: "verified: set unlock code", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "TZ_GET_LOCK_PLMN", Method: "POST", Description: "verified: read locked PLMN", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "TZ_SET_LOCK_PLMN", Method: "POST", Description: "verified: set locked PLMN", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "TZ_GET_LOCK_BAND", Method: "POST", Description: "verified: read locked band", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "TZ_SET_LOCK_BAND", Method: "POST", Description: "verified: set locked band", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "CLEAR_IMSI", Method: "POST", Description: "verified: clear IMSI state — DESTRUCTIVE", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "DISCONNECT_NETWORK", Method: "POST", Description: "verified: disconnect network", Args: map[string]string{
			"isTest": "false", "notCallback": "true", "disconnect_internet": "1",
		}},
		{ID: "SCAN_NETWORK", Method: "POST", Description: "verified: scan for networks", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "APN_PROC", Method: "POST", Description: "verified: non-IPv6 APN processing variant", Args: map[string]string{
			"isTest": "false", "apn_action": "set_default", "index": "1", "apn_mode": "auto",
		}},
		{ID: "REBOOT_DEVICE", Method: "POST", Description: "verified: reboot — DESTRUCTIVE", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "LOGOUT", Method: "POST", Description: "verified: end session", Args: map[string]string{
			"isTest": "false",
		}},
		{ID: "MONITOR_SETTINGS", Method: "POST", Description: "verified: monitor settings", Args: map[string]string{
			"isTest": "false", "siproxd_enable": "0", "sip_listen_port": "5060",
		}},
		{ID: "SET_UART_BAUDRATE", Method: "POST", Description: "verified: configure UART baud rate", Args: map[string]string{
			"isTest": "false",
		}},
	}
}

// MergePayloads concatenates payload sets, de-duplicating by goform ID.
// Later sets win over earlier ones, so profile entries override built-ins.
func MergePayloads(sets ...[]profile.Payload) []profile.Payload {
	order := make([]string, 0)
	byID := make(map[string]profile.Payload)
	for _, set := range sets {
		for _, p := range set {
			if _, seen := byID[p.ID]; !seen {
				order = append(order, p.ID)
			}
			byID[p.ID] = p
		}
	}
	out := make([]profile.Payload, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

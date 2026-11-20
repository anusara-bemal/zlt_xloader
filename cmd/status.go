package cmd

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/pkg/out"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show device, SIM, lock and signal status",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := app.API.Fingerprint(app.Profile.Fingerprint)
		if err != nil {
			log.WithError(err).Warn("fingerprint failed")
		}

		rows := [][2]string{
			{"target", app.Target},
			{"profile", app.Profile.Name},
			{"firmware", valueOrUnknown(info.Firmware)},
			{"model", valueOrUnknown(info.Model)},
			{"acs_url", valueOrUnknown(info.ACSURL)},
		}

		client, err := ensureShell()
		if err != nil {
			log.WithError(err).Warn("shell unavailable; showing web API data only")
		} else {
			defer client.Close()
			_, _ = client.Expect("# ", 3*time.Second)

			nvKeys := []string{
				"sim_plmn", "lte_plmn", "tz_lock_plmn_state", "tz_lock_plmn_list",
				"tz_lock_band_state", "plmn_unlock", "tz_unlock_plmn_times",
				"network_type", "rssi", "lte_rsrp",
			}
			values := make(map[string]string, len(nvKeys))
			for _, key := range nvKeys {
				resp, rerr := client.Run("nv get "+key, "# ", 3*time.Second)
				if rerr != nil {
					log.Debugf("[status] nv get %s: %v", key, rerr)
					continue
				}
				log.Debugf("[status] nv get %s raw: %q", key, resp)
				values[key] = cleanShellOutput(resp, "nv get "+key)
			}
			for _, key := range nvKeys {
				label := "nv." + key
				v := values[key]
				if v == "" {
					v = "-"
				}
				rows = append(rows, [2]string{label, v})
			}
		}

		fmt.Println("=== Device status ===")
		out.Table(rows)
		return nil
	},
}

func cleanShellOutput(raw, command string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, command, "")
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "#") {
			continue
		}
		return strings.TrimSpace(line)
	}
	return ""
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

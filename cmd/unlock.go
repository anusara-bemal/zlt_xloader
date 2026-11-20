package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
	"github.com/anusara-bemal/zlt_xloader/pkg/safety"
)

var (
	unlockReboot bool
	unlockNoSave bool
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Remove the SLT/Dialog PLMN and band locks using the root shell",
	Long: `Uses the REMOVE_WHITE_SITE command injection to run nv/sed commands
as root on the router. This clears the PLMN lock list, band locks and IMSI
lock so the device accepts SIMs from any network.

Changes are written to NVRAM (nv save) and to the default parameter files so
they survive a factory reset. Nothing is sent unless --dry-run=false.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.DryRun {
			log.Warn("DRY-RUN: no unlock commands will be sent to the router")
		} else if !safety.Confirm("Modify the router's NVRAM to remove SIM/network locks?", app.Yes) {
			return fmt.Errorf("aborted by operator")
		}

		if err := app.API.Login(app.Profile.Login.Username, app.Profile.Login.Password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		log.Infof("authenticated; CSRFToken=%q", app.API.Token())

		commands := []struct {
			name string
			cmd  string
		}{
			{"backup", "nv show > /tmp/zlt-unlocker-nv-backup.txt"},
			{"plmn state", "nv set tz_lock_plmn_state=no"},
			{"plmn list", "nv set tz_lock_plmn_list="},
			{"band state", "nv set tz_lock_band_state=no"},
			{"3g band state", "nv set tz_lock_3G_band_state=no"},
			{"imsi state", "nv set tz_lock_imsi_state=0"},
			{"lock net", "nv set use_lock_net=no"},
			{"sim lock", "nv set tz_lock_sim=0"},
		}

		for _, c := range commands {
			payload := profile.Payload{
				ID:     "REMOVE_WHITE_SITE",
				Method: "POST",
				Args: map[string]string{
					"isTest": "false",
					"ids":    "1 $(" + c.cmd + ")",
				},
			}
			r := app.API.Try(payload)
			if r.Err != nil {
				log.WithError(r.Err).Warnf("[unlock] %s: request failed", c.name)
			} else {
				log.Infof("[unlock] %-16s %s", c.name, strings.TrimSpace(r.Body))
			}
		}

		if !unlockNoSave {
			r := app.API.Try(profile.Payload{
				ID: "REMOVE_WHITE_SITE", Method: "POST",
				Args: map[string]string{
					"isTest": "false",
					"ids":    "1 $(nv save)",
				},
			})
			if r.Err != nil {
				log.WithError(r.Err).Warn("[unlock] nv save failed")
			} else {
				log.Infof("[unlock] nv save: %s", strings.TrimSpace(r.Body))
			}
		}

		// Persist the change in the default files so it survives a factory reset.
		defaults := []string{
			"tz_lock_plmn_state=no",
			"tz_lock_plmn_list=",
			"tz_lock_band_state=no",
			"tz_lock_3G_band_state=no",
		}
		for _, line := range defaults {
			key := strings.SplitN(line, "=", 2)[0]
			cmdStr := fmt.Sprintf("sed -i 's|^%s=.*|%s|' /etc_ro/default/default_parameter_sys", key, line)
			app.API.Try(profile.Payload{
				ID: "REMOVE_WHITE_SITE", Method: "POST",
				Args: map[string]string{"isTest": "false", "ids": "1 $(" + cmdStr + ")"},
			})
		}
		log.Info("[unlock] default parameter files updated")

		if unlockReboot {
			log.Warn("[unlock] rebooting router; telnet will need to be re-enabled with 'zlt-unlocker telnet'")
			app.API.Try(profile.Payload{
				ID: "REMOVE_WHITE_SITE", Method: "POST",
				Args: map[string]string{"isTest": "false", "ids": "1 $(reboot)"},
			})
		}

		fmt.Println()
		fmt.Println("NVRAM backup saved on the router at /tmp/zlt-unlocker-nv-backup.txt")
		fmt.Println("Verify with: nv get tz_lock_plmn_state   (expect 'no')")
		return nil
	},
}

func init() {
	unlockCmd.Flags().BoolVar(&unlockReboot, "reboot", false, "reboot the router after unlocking")
	unlockCmd.Flags().BoolVar(&unlockNoSave, "no-save", false, "do not run nv save")
	rootCmd.AddCommand(unlockCmd)
}

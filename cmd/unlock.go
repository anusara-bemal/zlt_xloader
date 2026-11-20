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

// defaultUnlockParams keeps older profiles (without an "unlock" section)
// working exactly as before this profile-driven refactor.
var defaultUnlockParams = []profile.UnlockParam{
	{Key: "tz_lock_plmn_state", Value: "no", Path: "/etc_ro/default/default_parameter_sys"},
	{Key: "tz_lock_plmn_list", Value: "", Path: "/etc_ro/default/default_parameter_sys"},
	{Key: "tz_lock_band_state", Value: "no", Path: "/etc_ro/default/default_parameter_sys"},
	{Key: "tz_lock_3G_band_state", Value: "no", Path: "/etc_ro/default/default_parameter_sys"},
	{Key: "tz_lock_imsi_state", Value: "0"},
	{Key: "use_lock_net", Value: "no"},
	{Key: "tz_lock_sim", Value: "0"},
}

func runUnlockCommand(name, command string) {
	r := app.API.Try(profile.Payload{
		ID: "REMOVE_WHITE_SITE", Method: "POST",
		Args: map[string]string{"isTest": "false", "ids": "1 $(" + command + ")"},
	})
	if r.Err != nil {
		log.WithError(r.Err).Warnf("[unlock] %s: request failed", name)
		return
	}
	log.Infof("[unlock] %-20s %s", name, strings.TrimSpace(r.Body))
}

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Remove the SIM/network locks using the root shell",
	Long: `Uses the REMOVE_WHITE_SITE command injection to run nv/at/sed commands
as root on the router. The exact parameters come from the active device profile,
so the SLT/Dialog, MTN and ORG firmware variants all use the same binary.

Changes are written to NVRAM and to the default parameter files so they survive
a factory reset. Nothing is sent unless --dry-run=false.`,
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

		params := app.Profile.Unlock.NV
		if len(params) == 0 {
			params = defaultUnlockParams
		}

		runUnlockCommand("backup", "nv show > "+app.Profile.Unlock.BackupPath)
		for _, u := range params {
			runUnlockCommand(u.Key, fmt.Sprintf("nv set %s=%s", u.Key, u.Value))
		}
		for _, u := range app.Profile.Unlock.AT {
			at := fmt.Sprintf("AT+ZNVSET='%s','%s'", u.Key, u.Value)
			runUnlockCommand("at:"+u.Key, fmt.Sprintf("at_cmd \"%s\"", at))
		}

		save := app.Profile.Unlock.Save || len(app.Profile.Unlock.NV) == 0
		if unlockNoSave {
			save = false
		}
		if save {
			runUnlockCommand("nv save", "nv save")
		}

		// Persist the change in the default files so it survives a factory reset.
		for _, u := range params {
			if u.Path == "" {
				continue
			}
			runUnlockCommand("default:"+u.Key,
				fmt.Sprintf("sed -i 's|^%s=.*|%s=%s|' %s", u.Key, u.Key, u.Value, u.Path))
		}

		if unlockReboot {
			log.Warn("[unlock] rebooting router; telnet will need to be re-enabled with 'zlt_xloader telnet'")
			runUnlockCommand("reboot", "reboot")
		}

		fmt.Println()
		fmt.Printf("NVRAM backup saved on the router at %s\n", app.Profile.Unlock.BackupPath)
		fmt.Println("Verify with: nv get tz_lock_plmn_state   (expect 'no')")
		return nil
	},
}

func init() {
	unlockCmd.Flags().BoolVar(&unlockReboot, "reboot", false, "reboot the router after unlocking")
	unlockCmd.Flags().BoolVar(&unlockNoSave, "no-save", false, "do not run nv save")
	rootCmd.AddCommand(unlockCmd)
}

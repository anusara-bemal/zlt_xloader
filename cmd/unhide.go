package cmd

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt-unlocker/pkg/safety"
)

var unhideReboot bool

var unhideCmd = &cobra.Command{
	Use:   "unhide",
	Short: "Reveal UI pages and features hidden by the carrier firmware",
	Long: `Clears the carrier's UI hiding bitfields (tz_web_page_hide,
tz_web_user_hide, tz_web_special_func) and enables feature flags such as
need_support_sms and need_support_ussd.

This makes the hidden pages (SMS, network selection, APN, and others) visible
in the router's web interface. A reboot is recommended.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.DryRun {
			log.Warn("DRY-RUN: no changes will be written")
		} else if !safety.Confirm("Reveal all hidden UI pages on the router?", app.Yes) {
			return fmt.Errorf("aborted by operator")
		}

		client, err := ensureShell()
		if err != nil {
			return err
		}
		defer client.Close()
		_, _ = client.Expect("# ", 3*time.Second)

		settings := []string{
			"tz_web_page_hide=0000000000000000000000000000000000000000",
			"tz_web_user_hide=00000000000000000000000000000000",
			"tz_web_special_func=0",
			"need_support_sms=yes",
			"need_support_ussd=yes",
			"show_sim_spn=yes",
		}
		for _, setting := range settings {
			if app.DryRun {
				log.Infof("[unhide] would set %s", setting)
				continue
			}
			if _, err := client.Run("nv set "+setting, "# ", 3*time.Second); err != nil {
				return fmt.Errorf("nv set %s: %w", setting, err)
			}
		}
		if !app.DryRun {
			if _, err := client.Run("nv save", "# ", 5*time.Second); err != nil {
				return fmt.Errorf("nv save: %w", err)
			}
		}
		log.Infof("[unhide] %d settings applied", len(settings))

		if unhideReboot {
			log.Warn("[unhide] rebooting")
			if !app.DryRun {
				_, _ = client.Run("reboot", "# ", 3*time.Second)
			}
		}
		fmt.Println("Hidden UI pages should now be visible after a reboot.")
		return nil
	},
}

func init() {
	unhideCmd.Flags().BoolVar(&unhideReboot, "reboot", false, "reboot after applying")
	rootCmd.AddCommand(unhideCmd)
}

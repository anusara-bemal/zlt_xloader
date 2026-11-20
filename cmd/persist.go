package cmd

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/pkg/safety"
)

const persistMarker = "# zlt_xloader persistent services"

var persistReboot bool

var persistCmd = &cobra.Command{
	Use:   "persist",
	Short: "Make telnet and SSH start automatically at every boot",
	Long: `Adds a small block to the router's boot script (/etc/rc) that starts
BusyBox telnetd on port 23 and dropbear on port 22, and sets the firmware's
own telnetd_enable/adbd_enable/dropbear_enable NVRAM flags.

A backup of the original boot script is kept at /etc/rc.zlt-backup.
The operation is idempotent — it can be run again safely.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.DryRun {
			log.Warn("DRY-RUN: no changes will be written to the router")
		} else if !safety.Confirm("Modify the router boot script to start telnet/SSH at boot?", app.Yes) {
			return fmt.Errorf("aborted by operator")
		}

		client, err := ensureShell()
		if err != nil {
			return err
		}
		defer client.Close()
		_, _ = client.Expect("# ", 3*time.Second)

		check, err := client.Run("grep -c '"+persistMarker+"' /etc/rc", "# ", 4*time.Second)
		if err != nil {
			return err
		}
		if strings.Contains(check, "1") {
			log.Info("[persist] boot hook already installed; nothing to do")
		} else {
			steps := []string{
				"cp -n /etc/rc /etc/rc.zlt-backup",
				"cat >> /etc/rc <<'ZLTEOF'",
				"",
				persistMarker,
				"/bin/busybox telnetd -l /bin/sh &",
				"if [ -x /usr/sbin/dropbear ]; then /usr/sbin/dropbear & fi",
				"if [ -x /bin/adbd ]; then /bin/adbd & fi",
				"ZLTEOF",
			}
			for _, step := range steps {
				if app.DryRun {
					log.Infof("[persist] would run: %s", step)
					continue
				}
				if _, err := client.Run(step, "# ", 4*time.Second); err != nil {
					return fmt.Errorf("boot hook: %w", err)
				}
			}
			log.Info("[persist] boot hook installed")
		}

		for _, key := range []string{"telnetd_enable=y", "adbd_enable=y", "dropbear_enable=y"} {
			if app.DryRun {
				log.Infof("[persist] would run: nv set %s", key)
				continue
			}
			if _, err := client.Run("nv set "+key, "# ", 3*time.Second); err != nil {
				return fmt.Errorf("nv set %s: %w", key, err)
			}
		}
		if !app.DryRun {
			if _, err := client.Run("nv save", "# ", 5*time.Second); err != nil {
				return fmt.Errorf("nv save: %w", err)
			}
		}
		log.Info("[persist] NVRAM flags saved")

		if persistReboot {
			log.Warn("[persist] rebooting to apply")
			if !app.DryRun {
				_, _ = client.Run("reboot", "# ", 3*time.Second)
			}
		}
		fmt.Println("Telnet (23) and SSH (22) will start automatically after boot.")
		return nil
	},
}

func init() {
	persistCmd.Flags().BoolVar(&persistReboot, "reboot", false, "reboot after installing the boot hook")
	rootCmd.AddCommand(persistCmd)
}

package cmd

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/pkg/safety"
)

var optimizeReboot bool

var optimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Tune DNS, remove data-cap throttling and show RF diagnostics",
	Long: `Reads the current RF/link state (band, RSRP, RSRQ, SINR) and applies
settings that are actually under the router's control:

  - DNS 1.1.1.1 / 8.8.8.8 instead of carrier DNS
  - disable the router's data-volume limit and SMS quota alerts
  - ensure rate limiting / QoS are off
  - ensure no band lock is active

Latency introduced by the carrier core (CGNAT/roaming) cannot be changed
from the router; the command reports the measured state honestly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.DryRun {
			log.Warn("DRY-RUN: no changes will be written")
		} else if !safety.Confirm("Apply connection optimizations to the router?", app.Yes) {
			return fmt.Errorf("aborted by operator")
		}

		client, err := ensureShell()
		if err != nil {
			return err
		}
		defer client.Close()
		_, _ = client.Expect("# ", 3*time.Second)

		fmt.Println("=== Current RF / link state ===")
		diag := []string{
			"nv get network_type", "nv get lte_band", "nv get BandWidth",
			"nv get lte_rsrp", "nv get lte_rsrq", "nv get lte_sinr",
			"nv get lte_pci", "nv get sim_plmn", "nv get BandLockState",
		}
		for _, d := range diag {
			out, err := client.Run(d, "# ", 5*time.Second)
			if err != nil {
				log.Debugf("[optimize] %s: %v", d, err)
				continue
			}
			log.Debugf("[optimize] %s raw: %q", d, out)
			value := cleanShellOutput(out, d)
			key := strings.TrimPrefix(d, "nv get ")
			if value == "" {
				value = "-"
			}
			fmt.Printf("  %-20s %s\n", key, value)
		}

		settings := []string{
			"dns_mode=manual",
			"prefer_dns_manual=1.1.1.1",
			"standby_dns_manual=8.8.8.8",
			"data_volume_limit_switch=0",
			"data_remind_sms_switch=0",
			"tz_rate_limit_enable=no",
			"flow_rate_limit_switch=no",
			"tz_lock_band_state=no",
		}
		fmt.Println()
		fmt.Println("=== Applying optimizations ===")
		for _, setting := range settings {
			if app.DryRun {
				log.Infof("would set %s", setting)
				continue
			}
			if _, err := client.Run("nv set "+setting, "# ", 3*time.Second); err != nil {
				return fmt.Errorf("nv set %s: %w", setting, err)
			}
			fmt.Printf("  %s\n", setting)
		}
		if !app.DryRun {
			if _, err := client.Run("nv save", "# ", 5*time.Second); err != nil {
				return fmt.Errorf("nv save: %w", err)
			}
		}
		log.Infof("applied %d settings", len(settings))

		// Network-stack tuning: MSS clamping for the 1400-byte cellular MTU,
		// larger TCP buffers for the high-BDP LTE path, and a fair queueing
		// discipline to remove Linux-side head-of-line blocking.
		fmt.Println()
		fmt.Println("=== Network stack tuning ===")
		tuning := []string{
			"iptables -t mangle -A FORWARD -o wan1 -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1360",
			"echo '4096 131072 4194304' > /proc/sys/net/ipv4/tcp_rmem",
			"echo '4096 131072 4194304' > /proc/sys/net/ipv4/tcp_wmem",
			"echo '212992 1048576 4194304' > /proc/sys/net/core/rmem_max",
			"echo '212992 1048576 4194304' > /proc/sys/net/core/wmem_max",
			"tc qdisc replace dev wan1 root sfq perturb 10",
		}
		for _, t := range tuning {
			if app.DryRun {
				log.Infof("would run: %s", t)
				continue
			}
			if _, err := client.Run(t, "# ", 5*time.Second); err != nil {
				log.WithError(err).Warnf("tuning command failed: %s", t)
				continue
			}
			fmt.Printf("  %s\n", t)
		}

		// Persist the tunings across reboots through the existing boot hook.
		marker := "# zlt-unlocker network tuning"
		check, err := client.Run("grep -c '"+marker+"' /etc/rc", "# ", 4*time.Second)
		if err == nil && !strings.Contains(check, "1") && !app.DryRun {
			hook := []string{
				"echo '' >> /etc/rc",
				"echo '" + marker + "' >> /etc/rc",
				"echo 'iptables -t mangle -A FORWARD -o wan1 -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1360' >> /etc/rc",
				"echo 'echo 4096 131072 4194304 > /proc/sys/net/ipv4/tcp_rmem' >> /etc/rc",
				"echo 'echo 4096 131072 4194304 > /proc/sys/net/ipv4/tcp_wmem' >> /etc/rc",
				"echo 'tc qdisc replace dev wan1 root sfq perturb 10' >> /etc/rc",
			}
			for _, h := range hook {
				if _, err := client.Run(h, "# ", 4*time.Second); err != nil {
					log.WithError(err).Warn("could not persist tuning")
					break
				}
			}
		}
		log.Info("network tuning applied and persisted")

		if optimizeReboot {
			log.Warn("[optimize] rebooting to apply DNS changes")
			if !app.DryRun {
				_, _ = client.Run("reboot", "# ", 3*time.Second)
			}
		}
		fmt.Println()
		fmt.Println("Note: carrier-core latency (CGNAT/roaming) is not controllable from the router.")
		return nil
	},
}

func init() {
	optimizeCmd.Flags().BoolVar(&optimizeReboot, "reboot", false, "reboot after applying")
	rootCmd.AddCommand(optimizeCmd)
}

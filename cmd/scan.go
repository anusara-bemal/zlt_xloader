package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/pkg/netutil"
	"github.com/anusara-bemal/zlt_xloader/pkg/out"
)

var scanPorts []int

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Check reachability, scan ports and fingerprint the firmware",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Target: %s  (profile: %s)\n", app.Target, app.Profile.Name)

		ports := scanPorts
		if len(ports) == 0 {
			ports = []int{23, 80, 8080, 7547}
		}
		log.Infof("scanning %d TCP ports on %s", len(ports), app.Target)
		results := netutil.ScanPorts(app.Target, ports, len(ports), app.Timeout)

		rows := make([][2]string, 0, len(ports))
		for _, p := range ports {
			rows = append(rows, [2]string{fmt.Sprintf("tcp/%d", p), out.Status(results[p])})
		}
		fmt.Println()
		out.Table(rows)

		log.Info("fingerprinting device via web API")
		info, err := app.API.Fingerprint(app.Profile.Fingerprint)
		if err != nil {
			log.WithError(err).Warn("fingerprinting failed")
		}
		fmt.Println()
		out.Table([][2]string{
			{"profile", app.Profile.Name},
			{"firmware", valueOrUnknown(info.Firmware)},
			{"model", valueOrUnknown(info.Model)},
			{"acs_url", valueOrUnknown(info.ACSURL)},
			{"dry_run", fmt.Sprint(app.DryRun)},
		})

		if app.JSON {
			return out.PrintJSON(struct {
				Target string       `json:"target"`
				Ports  map[int]bool `json:"ports"`
				Info   any          `json:"fingerprint"`
				DryRun bool         `json:"dry_run"`
			}{app.Target, results, info, app.DryRun})
		}
		return nil
	},
}

func valueOrUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return s
}

func init() {
	scanCmd.Flags().IntSliceVar(&scanPorts, "ports", []int{23, 80, 8080, 7547},
		"TCP ports to scan (default: 23 telnet, 80 http, 8080 alt-http, 7547 TR-069)")
	rootCmd.AddCommand(scanCmd)
}

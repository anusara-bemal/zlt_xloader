package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt-unlocker/internal/profile"
	"github.com/anusara-bemal/zlt-unlocker/pkg/safety"
)

var powerCmd = &cobra.Command{
	Use:   "power {reboot|off}",
	Short: "Reboot or power off the router",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		action := strings.ToLower(args[0])
		var id string
		switch action {
		case "reboot":
			id = "REBOOT_DEVICE"
		case "off", "shutdown":
			id = "TURN_OFF_DEVICE"
		default:
			return fmt.Errorf("unknown action %q (use reboot or off)", args[0])
		}

		if !app.DryRun && !safety.Confirm(fmt.Sprintf("Send %s to %s?", id, app.Target), app.Yes) {
			return fmt.Errorf("aborted by operator")
		}
		if err := app.API.Login(app.Profile.Login.Username, app.Profile.Login.Password); err != nil {
			return fmt.Errorf("login: %w", err)
		}

		r := app.API.Try(profile.Payload{
			ID: id, Method: "POST",
			Args: map[string]string{"isTest": "false"},
		})
		if r.Err != nil {
			// A reboot drops the connection before the router can reply.
			if id == "REBOOT_DEVICE" && strings.Contains(r.Err.Error(), "EOF") {
				log.Infof("REBOOT_DEVICE: connection dropped; router is rebooting")
				return nil
			}
			return fmt.Errorf("%s: %w", id, r.Err)
		}
		log.Infof("%s: %s", id, strings.TrimSpace(r.Body))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(powerCmd)
}

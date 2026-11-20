package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt_xloader/internal/buildinfo"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version and build information",
		// Version does not need a device profile or API client.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(buildinfo.String())
		},
	})
}

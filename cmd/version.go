package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is overridden via -ldflags at release time.
var Version = "dev"

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kploy CLI version",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), Version)
			return nil
		},
	}
}

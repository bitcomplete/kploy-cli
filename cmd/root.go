package cmd

import (
	"github.com/spf13/cobra"
)

var (
	outputFormat string
)

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "kploy",
		Short:         "kploy CLI — manage Kploy environments",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format: table | json")

	root.AddCommand(authCommand())
	root.AddCommand(clusterCommand())
	root.AddCommand(deployCommand())
	root.AddCommand(envCommand())
	root.AddCommand(imageCommand())
	root.AddCommand(orgCommand())
	root.AddCommand(repoCommand())
	root.AddCommand(versionCommand())

	return root
}

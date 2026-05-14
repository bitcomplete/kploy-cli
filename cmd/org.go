package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bitcomplete/kploy-cli/internal/config"
	"github.com/bitcomplete/kploy-cli/internal/kployclient"
	"github.com/spf13/cobra"
)

func orgCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "org",
		Short: "Manage GitHub organizations visible to kploy",
	}
	c.AddCommand(orgListCommand())
	return c
}

func orgListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations where the kploy GitHub App is installed",
		RunE: func(c *cobra.Command, args []string) error {
			ctx := c.Context()
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := kployclient.New(ctx, cfg)
			if err != nil {
				return err
			}
			resp, err := client.ListOrgsWithResponse(ctx)
			if err != nil {
				return err
			}
			if resp.StatusCode() == http.StatusUnauthorized {
				return errors.New("token rejected; run `kploy auth login` again")
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("unexpected response %d", resp.StatusCode())
			}
			rows := make([][]string, 0, len(*resp.JSON200))
			for _, o := range *resp.JSON200 {
				rows = append(rows, []string{o.Name})
			}
			return renderRows(c.OutOrStdout(), []string{"NAME"}, rows)
		},
	}
}

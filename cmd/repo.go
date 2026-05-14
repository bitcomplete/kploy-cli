package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bitcomplete/kploy-cli/internal/config"
	"github.com/bitcomplete/kploy-cli/internal/kployclient"
	"github.com/spf13/cobra"
)

func repoCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories tracked by kploy",
	}
	c.AddCommand(repoListCommand())
	return c
}

func repoListCommand() *cobra.Command {
	var orgFlag string
	c := &cobra.Command{
		Use:   "list",
		Short: "List repositories in an org that have at least one kploy environment",
		RunE: func(c *cobra.Command, args []string) error {
			ctx := c.Context()
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			org, err := resolveOrg(cfg, orgFlag)
			if err != nil {
				return err
			}
			client, err := kployclient.New(ctx, cfg)
			if err != nil {
				return err
			}
			resp, err := client.ListReposWithResponse(ctx, org)
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
			for _, r := range *resp.JSON200 {
				rows = append(rows, []string{r.Name})
			}
			return renderRows(c.OutOrStdout(), []string{"NAME"}, rows)
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	return c
}

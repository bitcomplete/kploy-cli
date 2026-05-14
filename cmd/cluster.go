package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bitcomplete/kploy-cli/internal/config"
	"github.com/bitcomplete/kploy-cli/internal/kployclient"
	"github.com/spf13/cobra"
)

func clusterCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "cluster",
		Short: "Manage kploy clusters",
	}
	c.AddCommand(clusterListCommand())
	c.AddCommand(clusterCreateCommand())
	return c
}

func clusterListCommand() *cobra.Command {
	var orgFlag string
	c := &cobra.Command{
		Use:   "list",
		Short: "List clusters in an org",
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
			resp, err := client.ListClustersWithResponse(ctx, org)
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
			for _, cl := range *resp.JSON200 {
				connected := "disconnected"
				if cl.Connected {
					connected = "connected"
				}
				lastSeen := "—"
				if cl.LastSeenAt != nil {
					lastSeen = cl.LastSeenAt.Format(time.RFC3339)
				}
				rows = append(rows, []string{cl.ID, connected, lastSeen})
			}
			return renderRows(c.OutOrStdout(), []string{"ID", "STATUS", "LAST_SEEN"}, rows)
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	return c
}

func clusterCreateCommand() *cobra.Command {
	var orgFlag string
	c := &cobra.Command{
		Use:   "create",
		Short: "Mint a new cluster and its one-shot bearer token",
		Long: `Creates a new kploy cluster and prints its ID and bearer token.

The token is shown EXACTLY ONCE. Store it somewhere your operator can
read it (e.g. as the CLUSTER_TOKEN secret in your cluster's deployment).
If you lose the token you must create a new cluster.`,
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
			resp, err := client.CreateClusterWithResponse(ctx, org)
			if err != nil {
				return err
			}
			switch resp.StatusCode() {
			case http.StatusCreated:
			case http.StatusNotFound:
				return errors.New("org not found")
			case http.StatusUnauthorized:
				return errors.New("token rejected; run `kploy auth login` again")
			default:
				return fmt.Errorf("unexpected response %d", resp.StatusCode())
			}
			if outputFormat == "json" {
				return renderJSON(c.OutOrStdout(), resp.JSON201)
			}
			fmt.Fprintf(c.OutOrStdout(), "Cluster ID:    %s\nCluster Token: %s\n\n", resp.JSON201.ID, resp.JSON201.Token)
			fmt.Fprintln(c.OutOrStdout(), "Save the token now — it will not be shown again.")
			return nil
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	return c
}

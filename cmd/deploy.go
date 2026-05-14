package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bitcomplete/kploy-cli/internal/config"
	"github.com/bitcomplete/kploy-cli/internal/kployclient"
	"github.com/spf13/cobra"
)

func deployCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "deploy",
		Short: "Inspect deployments and their logs",
	}
	c.AddCommand(deployListCommand())
	c.AddCommand(deployLogsCommand())
	return c
}

func deployListCommand() *cobra.Command {
	var orgFlag, repoFlag, envFlag string
	c := &cobra.Command{
		Use:   "list",
		Short: "List recent deployments for an environment",
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
			if repoFlag == "" || envFlag == "" {
				return errors.New("--repo and --env are required")
			}
			client, err := kployclient.New(ctx, cfg)
			if err != nil {
				return err
			}
			resp, err := client.ListDeploymentsWithResponse(ctx, org, repoFlag, envFlag)
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
			for _, d := range *resp.JSON200 {
				msg := ""
				if d.CommitMessage != nil {
					msg = firstLine(*d.CommitMessage)
				}
				rows = append(rows, []string{
					strconv.FormatInt(d.ID, 10),
					d.Status,
					d.CommitSHA[:min(len(d.CommitSHA), 8)],
					msg,
					d.UpdatedAt.Format(time.RFC3339),
				})
			}
			return renderRows(c.OutOrStdout(), []string{"ID", "STATUS", "SHA", "MESSAGE", "UPDATED"}, rows)
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	c.Flags().StringVar(&envFlag, "env", "", "Environment name (required)")
	return c
}

func deployLogsCommand() *cobra.Command {
	var orgFlag, repoFlag string
	c := &cobra.Command{
		Use:   "logs DEPLOYMENT_ID",
		Short: "Print the logs for one deployment",
		Args:  cobra.ExactArgs(1),
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
			if repoFlag == "" {
				return errors.New("--repo is required")
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid deployment id %q: %w", args[0], err)
			}
			client, err := kployclient.New(ctx, cfg)
			if err != nil {
				return err
			}
			resp, err := client.ListDeploymentLogsWithResponse(ctx, org, repoFlag, id)
			if err != nil {
				return err
			}
			switch resp.StatusCode() {
			case http.StatusOK:
			case http.StatusUnauthorized:
				return errors.New("token rejected; run `kploy auth login` again")
			case http.StatusNotFound:
				return fmt.Errorf("deployment %d not found in %s/%s", id, org, repoFlag)
			default:
				return fmt.Errorf("unexpected response %d", resp.StatusCode())
			}
			if outputFormat == "json" {
				return renderJSON(c.OutOrStdout(), resp.JSON200)
			}
			for _, e := range *resp.JSON200 {
				fmt.Fprintf(c.OutOrStdout(), "%s %s\n", e.EmittedAt.Format(time.RFC3339), e.Output)
			}
			return nil
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	return c
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}

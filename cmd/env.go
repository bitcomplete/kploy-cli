package cmd

import (
	"errors"
	"fmt"
	"net/http"

	kployapi "github.com/bitcomplete/kploy-cli/client"
	"github.com/bitcomplete/kploy-cli/internal/config"
	"github.com/bitcomplete/kploy-cli/internal/kployclient"
	"github.com/spf13/cobra"
)

func envCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
	}
	c.AddCommand(envListCommand())
	c.AddCommand(envGetCommand())
	c.AddCommand(envCreateCommand())
	return c
}

func envListCommand() *cobra.Command {
	var orgFlag, repoFlag string
	c := &cobra.Command{
		Use:   "list",
		Short: "List environments for a repository",
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
			client, err := kployclient.New(ctx, cfg)
			if err != nil {
				return err
			}
			resp, err := client.ListEnvsWithResponse(ctx, org, repoFlag)
			if err != nil {
				return err
			}
			if err := checkStatusForList(resp.StatusCode(), resp.JSON200); err != nil {
				return err
			}
			rows := make([][]string, 0, len(*resp.JSON200))
			for _, e := range *resp.JSON200 {
				rows = append(rows, []string{e.Name, e.Branch, e.ClusterID, e.KubernetesNamespace, e.HeadSHA})
			}
			return renderRows(c.OutOrStdout(), []string{"NAME", "BRANCH", "CLUSTER", "NAMESPACE", "HEAD_SHA"}, rows)
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	return c
}

func envGetCommand() *cobra.Command {
	var orgFlag, repoFlag string
	c := &cobra.Command{
		Use:   "get NAME",
		Short: "Show details for one environment",
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
			client, err := kployclient.New(ctx, cfg)
			if err != nil {
				return err
			}
			resp, err := client.GetEnvWithResponse(ctx, org, repoFlag, args[0])
			if err != nil {
				return err
			}
			if resp.StatusCode() == http.StatusNotFound {
				return fmt.Errorf("environment %q not found", args[0])
			}
			if err := checkStatus(resp.StatusCode(), resp.JSON200); err != nil {
				return err
			}
			e := resp.JSON200
			if outputFormat == "json" {
				return renderJSON(c.OutOrStdout(), e)
			}
			fmt.Fprintf(c.OutOrStdout(), "Name:        %s\nBranch:      %s\nCluster:     %s\nNamespace:   %s\nHead SHA:    %s\n", e.Name, e.Branch, e.ClusterID, e.KubernetesNamespace, e.HeadSHA)
			if e.ManifestsSHA != nil {
				fmt.Fprintf(c.OutOrStdout(), "Manifests:   %s\n", *e.ManifestsSHA)
			}
			return nil
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	return c
}

func envCreateCommand() *cobra.Command {
	var orgFlag, repoFlag, name, cluster, branch, namespace string
	var trackedImages []string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create an environment",
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
			if repoFlag == "" || name == "" || cluster == "" || branch == "" || namespace == "" {
				return errors.New("--repo, --name, --cluster, --branch, --namespace are all required")
			}
			client, err := kployclient.New(ctx, cfg)
			if err != nil {
				return err
			}
			body := kployapi.CreateEnvJSONRequestBody{
				Name:                name,
				ClusterID:           cluster,
				Branch:              branch,
				KubernetesNamespace: namespace,
			}
			if len(trackedImages) > 0 {
				body.TrackedImages = &trackedImages
			}
			resp, err := client.CreateEnvWithResponse(ctx, org, repoFlag, body)
			if err != nil {
				return err
			}
			switch resp.StatusCode() {
			case http.StatusCreated:
			case http.StatusConflict:
				return fmt.Errorf("environment %q already exists", name)
			case http.StatusUnprocessableEntity:
				if resp.JSON422 != nil {
					return errors.New(resp.JSON422.Message)
				}
				return errors.New("validation failed")
			case http.StatusNotFound:
				return fmt.Errorf("org or repo not found")
			case http.StatusUnauthorized:
				return errors.New("token rejected; run `kploy auth login` again")
			default:
				return fmt.Errorf("unexpected response %d", resp.StatusCode())
			}
			e := resp.JSON201
			fmt.Fprintf(c.OutOrStdout(), "Created environment %q in %s/%s.\n", e.Name, org, repoFlag)
			return nil
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	c.Flags().StringVar(&name, "name", "", "Environment name")
	c.Flags().StringVar(&cluster, "cluster", "", "Cluster ID")
	c.Flags().StringVar(&branch, "branch", "main", "Git branch")
	c.Flags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	c.Flags().StringSliceVar(&trackedImages, "tracked-image", nil, "Tracked image name (repeatable)")
	return c
}

func resolveOrg(cfg *config.Config, flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if o := cfg.ResolveOrg(); o != "" {
		return o, nil
	}
	return "", errors.New("--org is required (or set KPLOY_ORG)")
}

// checkStatusForList enforces a 200-with-non-nil-JSON200 invariant
// across all the list endpoints.
func checkStatusForList[T any](status int, body *[]T) error {
	if status == http.StatusUnauthorized {
		return errors.New("token rejected; run `kploy auth login` again")
	}
	if body == nil {
		return fmt.Errorf("unexpected response %d", status)
	}
	return nil
}

func checkStatus[T any](status int, body *T) error {
	if status == http.StatusUnauthorized {
		return errors.New("token rejected; run `kploy auth login` again")
	}
	if body == nil {
		return fmt.Errorf("unexpected response %d", status)
	}
	return nil
}


package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	kployapi "github.com/bitcomplete/kploy-cli/client"
	"github.com/bitcomplete/kploy-cli/internal/config"
	"github.com/bitcomplete/kploy-cli/internal/kployclient"
	"github.com/spf13/cobra"
)

func imageCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "image",
		Short: "Manage the tracked-image list for an environment",
	}
	c.AddCommand(imageListCommand())
	c.AddCommand(imageAddCommand())
	c.AddCommand(imageRemoveCommand())
	return c
}

func imageListCommand() *cobra.Command {
	var orgFlag, repoFlag, envFlag string
	c := &cobra.Command{
		Use:   "list",
		Short: "List tracked images for an environment",
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
			resp, err := client.ListTrackedImagesWithResponse(ctx, org, repoFlag, envFlag)
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
			for _, img := range *resp.JSON200 {
				current := "—"
				if img.CurrentTag != nil {
					current = *img.CurrentTag
				}
				rows = append(rows, []string{img.Name, current, strings.Join(img.AvailableTags, ", ")})
			}
			return renderRows(c.OutOrStdout(), []string{"IMAGE", "CURRENT_TAG", "AVAILABLE_TAGS"}, rows)
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	c.Flags().StringVar(&envFlag, "env", "", "Environment name (required)")
	return c
}

// imageAddCommand and imageRemoveCommand are thin wrappers around the
// PUT replace endpoint: we fetch the current list, mutate locally, and
// PUT it back. Cleaner than exposing the diff semantics to the user.

func imageAddCommand() *cobra.Command {
	var orgFlag, repoFlag, envFlag string
	c := &cobra.Command{
		Use:   "add IMAGE",
		Short: "Track an additional image in an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return mutateTrackedImages(c, orgFlag, repoFlag, envFlag, func(names []string) []string {
				if slices.Contains(names, args[0]) {
					return names
				}
				return append(names, args[0])
			})
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	c.Flags().StringVar(&envFlag, "env", "", "Environment name (required)")
	return c
}

func imageRemoveCommand() *cobra.Command {
	var orgFlag, repoFlag, envFlag string
	c := &cobra.Command{
		Use:   "remove IMAGE",
		Short: "Stop tracking an image in an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return mutateTrackedImages(c, orgFlag, repoFlag, envFlag, func(names []string) []string {
				return slices.DeleteFunc(names, func(s string) bool { return s == args[0] })
			})
		},
	}
	c.Flags().StringVar(&orgFlag, "org", "", "Organization (defaults to KPLOY_ORG or configured org)")
	c.Flags().StringVar(&repoFlag, "repo", "", "Repository name (required)")
	c.Flags().StringVar(&envFlag, "env", "", "Environment name (required)")
	return c
}

func mutateTrackedImages(c *cobra.Command, orgFlag, repoFlag, envFlag string, mutate func([]string) []string) error {
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
	listResp, err := client.ListTrackedImagesWithResponse(ctx, org, repoFlag, envFlag)
	if err != nil {
		return err
	}
	if listResp.StatusCode() == http.StatusUnauthorized {
		return errors.New("token rejected; run `kploy auth login` again")
	}
	if listResp.JSON200 == nil {
		return fmt.Errorf("unexpected response %d listing tracked images", listResp.StatusCode())
	}
	names := make([]string, 0, len(*listResp.JSON200))
	for _, img := range *listResp.JSON200 {
		names = append(names, img.Name)
	}
	names = mutate(names)

	putResp, err := client.ReplaceTrackedImagesWithResponse(ctx, org, repoFlag, envFlag, kployapi.ReplaceTrackedImagesJSONRequestBody{Images: names})
	if err != nil {
		return err
	}
	switch putResp.StatusCode() {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return errors.New("token rejected; run `kploy auth login` again")
	case http.StatusUnprocessableEntity:
		if putResp.JSON422 != nil {
			return errors.New(putResp.JSON422.Message)
		}
		return errors.New("validation failed")
	default:
		return fmt.Errorf("unexpected response %d updating tracked images", putResp.StatusCode())
	}
	rows := make([][]string, 0, len(*putResp.JSON200))
	for _, img := range *putResp.JSON200 {
		current := "—"
		if img.CurrentTag != nil {
			current = *img.CurrentTag
		}
		rows = append(rows, []string{img.Name, current})
	}
	return renderRows(c.OutOrStdout(), []string{"IMAGE", "CURRENT_TAG"}, rows)
}

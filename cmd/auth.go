package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bitcomplete/kploy-cli/internal/config"
	"github.com/bitcomplete/kploy-cli/internal/kployclient"
	"github.com/spf13/cobra"
)

func authCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate to kploy via GitHub Device Flow",
	}
	c.AddCommand(authLoginCommand())
	c.AddCommand(authLogoutCommand())
	c.AddCommand(authWhoamiCommand())
	return c
}

func authLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to kploy",
		RunE: func(c *cobra.Command, args []string) error {
			ctx := c.Context()
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			server := cfg.ResolveServer()

			oauthCfg, err := kployclient.DeviceFlowOAuthConfig(ctx, server)
			if err != nil {
				return fmt.Errorf("bootstrap device flow: %w", err)
			}
			da, err := oauthCfg.DeviceAuth(ctx)
			if err != nil {
				return fmt.Errorf("start device flow: %w", err)
			}
			fmt.Fprintf(c.OutOrStdout(), "Open %s in your browser and enter the code:\n\n    %s\n\n", da.VerificationURI, da.UserCode)
			fmt.Fprintln(c.OutOrStdout(), "Waiting for approval...")

			token, err := oauthCfg.DeviceAccessToken(ctx, da)
			if err != nil {
				return fmt.Errorf("complete device flow: %w", err)
			}
			cfg.SetToken(token)
			cfg.Server = server
			if err := config.Save(cfg); err != nil {
				return err
			}

			// Echo back the GitHub login so the user sees who they authenticated as.
			if err := printWhoami(ctx, cfg, c); err != nil {
				return err
			}
			return nil
		},
	}
}

func authLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Forget the saved kploy token",
		RunE: func(c *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.Token() == nil {
				fmt.Fprintln(c.OutOrStdout(), "Not logged in.")
				return nil
			}
			cfg.SetToken(nil)
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), "Logged out.")
			return nil
		},
	}
}

func authWhoamiCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the GitHub identity associated with the current token",
		RunE: func(c *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return printWhoami(c.Context(), cfg, c)
		},
	}
}

// printWhoami verifies the saved token works by listing the user's
// orgs (which forces the server to call gh.Users.Get on its end).
// We don't have a /v1/me endpoint by design — listing orgs proves the
// token works and is what most users actually want to see anyway.
func printWhoami(ctx context.Context, cfg *config.Config, c *cobra.Command) error {
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
		return fmt.Errorf("unexpected response %d from kploy", resp.StatusCode())
	}
	names := make([]string, 0, len(*resp.JSON200))
	for _, o := range *resp.JSON200 {
		names = append(names, o.Name)
	}
	fmt.Fprintf(c.OutOrStdout(), "Logged in to %s.\nAccessible orgs: %s\n", cfg.ResolveServer(), strings.Join(names, ", "))
	return nil
}

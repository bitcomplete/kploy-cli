package cmd

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/bitcomplete/kployconfig"
	"github.com/spf13/cobra"
)

func validateConfigCommand() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "validate-config",
		Short: "Validate a local kploy.yaml file",
		Long: `Parse and validate a kploy.yaml file, then print the hostnames
it would render for production and development — plus, if preview
environments are enabled, an example PR preview env.`,
		RunE: func(c *cobra.Command, args []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("no %s in current directory; use --file to point elsewhere", file)
				}
				return fmt.Errorf("reading %s: %w", file, err)
			}
			cfg, err := kployconfig.Load(data)
			if err != nil {
				return err
			}

			envs := []struct {
				name string
				pr   int
			}{
				{"prod", 0},
				{"development", 0},
			}
			hostnames := make(map[string]string, len(envs))
			for _, env := range envs {
				h, err := kployconfig.RenderHostname(cfg, env.name, env.pr)
				if err != nil {
					return fmt.Errorf("render %s: %w", env.name, err)
				}
				hostnames[env.name] = h
			}

			var ingress []kployconfig.IngressHostname
			if cfg.Preview != nil && cfg.Preview.Enabled {
				ingress, err = kployconfig.RenderPreviewIngress(cfg, 1)
				if err != nil {
					return err
				}
			}

			out := c.OutOrStdout()
			if outputFormat == "json" {
				return renderJSON(out, struct {
					Valid          bool                          `json:"valid"`
					Project        string                        `json:"project"`
					Hostnames      map[string]string             `json:"hostnames"`
					PreviewIngress []kployconfig.IngressHostname `json:"previewIngress,omitempty"`
				}{
					Valid:          true,
					Project:        cfg.Project,
					Hostnames:      hostnames,
					PreviewIngress: ingress,
				})
			}

			_, _ = fmt.Fprintln(out, "OK")
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out, "Sample hostnames:")
			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(tw, "ENV\tHOSTNAME")
			for _, env := range envs {
				_, _ = fmt.Fprintf(tw, "%s\t%s\n", env.name, hostnames[env.name])
			}
			if err := tw.Flush(); err != nil {
				return err
			}

			if len(ingress) > 0 {
				_, _ = fmt.Fprintln(out)
				_, _ = fmt.Fprintln(out, "Example preview env (PR #1):")
				tw2 := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintln(tw2, "HOSTNAME\tSERVICE\tPORT")
				for _, ing := range ingress {
					_, _ = fmt.Fprintf(tw2, "%s\t%s\t%d\n", ing.Hostname, ing.ServiceName, ing.ServicePort)
				}
				if err := tw2.Flush(); err != nil {
					return err
				}
			}
			return nil
		},
	}
	c.Flags().StringVarP(&file, "file", "f", "kploy.yaml", "Path to kploy.yaml")
	return c
}

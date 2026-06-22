package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/benfiola/homelab-images/dynamic-dns/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "dynamic-dns",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "cloudflare-api-token",
					Required: true,
					Sources:  cli.EnvVars("CLOUDFLARE_API_TOKEN"),
				},
				&cli.StringFlag{
					Name:    "cloudflare-zones",
					Sources: cli.EnvVars("CLOUDFLARE_ZONES"),
				},
				&cli.BoolFlag{
					Name:    "continuous",
					Sources: cli.EnvVars("CONTINUOUS"),
				},
				&cli.DurationFlag{
					Name:    "interval",
					Sources: cli.EnvVars("INTERVAL"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				cloudflareZones := []internal.Zone{}
				if zonesJson := c.String("cloudflare-zones"); zonesJson != "" {
					if err := json.Unmarshal([]byte(zonesJson), &cloudflareZones); err != nil {
						return fmt.Errorf("failed to parse cloudflare zones: %w", err)
					}
				}

				continuous := c.Bool("continuous")
				
				client, err := internal.New(&internal.Opts{
					CloudflareAPIToken: c.String("cloudflare-api-token"),
					CloudflareZones:    cloudflareZones,
					Continuous:         &continuous,
					Interval:           c.Duration("interval"),
				})
				if err != nil {
					return err
				}
	
				return client.Run(ctx)
			},
		}),
	)
}

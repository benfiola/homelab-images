package main

import (
	"context"

	"github.com/benfiola/homelab-images/router-policy-sync/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "router-policy-sync",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "health-address",
					Sources: cli.EnvVars("HEALTH_ADDRESS"),
				},
				&cli.StringFlag{
					Name:    "kubeconfig",
					Sources: cli.EnvVars("KUBECONFIG"),
				},
				&cli.BoolFlag{
					Name:    "leader-election",
					Sources: cli.EnvVars("LEADER_ELECTION"),
				},
				&cli.StringFlag{
					Name:    "metrics-address",
					Sources: cli.EnvVars("METRICS_ADDRESS"),
				},
				&cli.StringFlag{
					Name:     "mikrotik-base-url",
					Required: true,
					Sources:  cli.EnvVars("MIKROTIK_BASE_URL"),
				},
				&cli.StringFlag{
					Name:     "mikrotik-password",
					Required: true,
					Sources:  cli.EnvVars("MIKROTIK_PASSWORD"),
				},
				&cli.StringFlag{
					Name:     "mikrotik-username",
					Required: true,
					Sources:  cli.EnvVars("MIKROTIK_USERNAME"),
				},
				&cli.StringSliceFlag{
					Name:    "reserved-cidrs",
					Sources: cli.EnvVars("RESERVED_CIDRS"),
				},
				&cli.DurationFlag{
					Name:    "sync-interval",
					Sources: cli.EnvVars("SYNC_INTERVAL"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				controller, err := internal.New(&internal.Opts{
					HealthAddress:    c.String("health-address"),
					Kubeconfig:       c.String("kubeconfig"),
					LeaderElection:   c.Bool("leader-election"),
					MetricsAddress:   c.String("metrics-address"),
					MikrotikBaseURL:  c.String("mikrotik-base-url"),
					MikrotikPassword: c.String("mikrotik-password"),
					MikrotikUsername: c.String("mikrotik-username"),
					ReservedCIDRs:    c.StringSlice("reserved-cidrs"),
					SyncInterval:     c.Duration("sync-interval"),
				})
				if err != nil {
					return err
				}

				return controller.Run(ctx)
			},
		}),
	)
}

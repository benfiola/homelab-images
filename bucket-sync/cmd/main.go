package main

import (
	"context"

	"github.com/benfiola/homelab-images/bucket-sync/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "bucket-sync",
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
					Name:    "namespace",
					Sources: cli.EnvVars("NAMESPACE"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				controller, err := internal.New(&internal.Opts{
					HealthAddress:  c.String("health-address"),
					Kubeconfig:     c.String("kubeconfig"),
					LeaderElection: c.Bool("leader-election"),
					MetricsAddress: c.String("metrics-address"),
					Namespace:      c.String("namespace"),
				})
				if err != nil {
					return err
				}

				return controller.Run(ctx)
			},
		}),
	)
}

package main

import (
	"context"

	"github.com/benfiola/homelab-images/linstor-provision-disk/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "linstor-provision-disk",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "partition-label",
					Required: true,
					Sources:  cli.EnvVars("PARTITION_LABEL"),
				},
				&cli.StringFlag{
					Name:     "pool",
					Required: true,
					Sources:  cli.EnvVars("POOL"),
				},
				&cli.StringFlag{
					Name:     "satellite-id",
					Required: true,
					Sources:  cli.EnvVars("SATELLITE_ID"),
				},
				&cli.StringFlag{
					Name:     "volume-group",
					Required: true,
					Sources:  cli.EnvVars("VOLUME_GROUP"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				provisioner, err := internal.New(&internal.Opts{
					PartitionLabel: c.String("partition-label"),
					Pool:           c.String("pool"),
					SatelliteID:    c.String("satellite-id"),
					VolumeGroup:    c.String("volume-group"),
				})
				if err != nil {
					return err
				}
				return provisioner.Run(ctx)
			},
		}),
	)
}

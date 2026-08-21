package main

import (
	"context"

	"github.com/benfiola/homelab-images/mdns-reflector/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "mdns-reflector",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "interfaces",
					Sources: cli.EnvVars("INTERFACES"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				reflector, err := internal.New(&internal.Opts{
					Interfaces: c.String("interfaces"),
				})
				if err != nil {
					return err
				}

				return reflector.Run(ctx)
			},
		}),
	)
}

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
					Name:    "source-interfaces",
					Sources: cli.EnvVars("SOURCE_INTERFACES"),
				},
				&cli.StringFlag{
					Name:     "dest-interfaces",
					Required: true,
					Sources:  cli.EnvVars("DEST_INTERFACES"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				reflector, err := internal.New(&internal.Opts{
					SourceInterfaces: c.String("source-interfaces"),
					DestInterfaces:   c.String("dest-interfaces"),
				})
				if err != nil {
					return err
				}

				return reflector.Run(ctx)
			},
		}),
	)
}

package main

import (
	"context"

	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/benfiola/homelab-images/vault-unseal/internal"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "vault-unseal",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "continuous",
					Sources: cli.EnvVars("CONTINUOUS"),
				},
				&cli.StringFlag{
					Name:     "vault-unseal-key",
					Required: true,
					Sources:  cli.EnvVars("VAULT_UNSEAL_KEY"),
				},
				&cli.StringFlag{
					Name:    "vault-addr",
					Sources: cli.EnvVars("VAULT_ADDR"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				unsealer, err := internal.New(&internal.Opts{
					Continuous:     cliutil.BoolPtr(c, "continuous"),
					VaultAddr:      c.String("vault-addr"),
					VaultUnsealKey: c.String("vault-unseal-key"),
				})
				if err != nil {
					return err
				}

				return unsealer.Run(ctx)
			},
		}),
	)
}

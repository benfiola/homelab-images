package main

import (
	"context"

	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/benfiola/homelab-images/vault-auth-proxy/internal"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "vault-auth-proxy",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "listen-addr",
					Value:   "127.0.0.1:8100",
					Sources: cli.EnvVars("LISTEN_ADDR"),
				},
				&cli.StringFlag{
					Name:    "vault-addr",
					Value:   "http://localhost:8200",
					Sources: cli.EnvVars("VAULT_ADDR"),
				},
				&cli.StringFlag{
					Name:    "root-token-path",
					Value:   "/vault/data/root-token",
					Sources: cli.EnvVars("ROOT_TOKEN_PATH"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				proxy, err := internal.New(&internal.Opts{
					ListenAddr:    c.String("listen-addr"),
					VaultAddr:     c.String("vault-addr"),
					RootTokenPath: c.String("root-token-path"),
				})
				if err != nil {
					return err
				}

				return proxy.Run(ctx)
			},
		}),
	)
}

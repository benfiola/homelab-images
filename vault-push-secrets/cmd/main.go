package main

import (
	"context"

	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/benfiola/homelab-images/vault-push-secrets/internal"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "vault-push-secrets",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "continuous",
					Sources: cli.EnvVars("CONTINUOUS"),
				},
				&cli.StringFlag{
					Name:     "bitwarden-access-token",
					Required: true,
					Sources:  cli.EnvVars("BITWARDEN_ACCESS_TOKEN"),
				},
				&cli.StringFlag{
					Name:     "bitwarden-secret-id",
					Required: true,
					Sources:  cli.EnvVars("BITWARDEN_SECRET_ID"),
				},
				&cli.DurationFlag{
					Name:    "interval",
					Sources: cli.EnvVars("INTERVAL"),
				},
				&cli.StringFlag{
					Name:     "vault-addr",
					Required: true,
					Sources:  cli.EnvVars("VAULT_ADDR"),
				},
				&cli.StringFlag{
					Name:     "vault-auth-mount",
					Required: true,
					Sources:  cli.EnvVars("VAULT_AUTH_MOUNT"),
				},
				&cli.StringFlag{
					Name:    "vault-auth-role",
					Sources: cli.EnvVars("VAULT_AUTH_ROLE"),
				},
				&cli.StringFlag{
					Name:    "vault-auth-token",
					Sources: cli.EnvVars("VAULT_AUTH_TOKEN"),
				},
				&cli.StringFlag{
					Name:     "vault-secrets-mount",
					Required: true,
					Sources:  cli.EnvVars("VAULT_SECRETS_MOUNT"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := internal.New(&internal.Opts{
					Continuous:           cliutil.BoolPtr(c, "continuous"),
					BitwardenAccessToken: c.String("bitwarden-access-token"),
					BitwardenSecretID:    c.String("bitwarden-secret-id"),
					Interval:             c.Duration("interval"),
					VaultAddr:            c.String("vault-addr"),
					VaultAuthMount:       c.String("vault-auth-mount"),
					VaultAuthRole:        c.String("vault-auth-role"),
					VaultAuthToken:       c.String("vault-auth-token"),
					VaultSecretsMount:    c.String("vault-secrets-mount"),
				})
				if err != nil {
					return err
				}

				return client.Run(ctx)
			},
		}),
	)
}

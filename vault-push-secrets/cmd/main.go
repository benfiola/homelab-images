package main

import (
	"context"

	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/benfiola/homelab-images/shared/pkg/ptr"
	"github.com/benfiola/homelab-images/vault-push-secrets/internal"
	"github.com/urfave/cli/v3"
)

func boolPtr(c *cli.Command, arg string) *bool {
	var value *bool
	if c.IsSet(arg) {
		value = ptr.Get(c.Bool(arg))
	}
	return value
}

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
					Name:     "encryption-key",
					Required: true,
					Sources:  cli.EnvVars("ENCRYPTION_KEY"),
				},
				&cli.StringFlag{
					Name:     "gcs-credentials",
					Required: true,
					Sources:  cli.EnvVars("GCS_CREDENTIALS"),
				},
				&cli.StringFlag{
					Name:     "gcs-destination",
					Required: true,
					Sources:  cli.EnvVars("GCS_DESTINATION"),
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
					Continuous:        boolPtr(c, "continuous"),
					EncryptionKey:     c.String("encryption-key"),
					GCSCredentials:    c.String("gcs-credentials"),
					GCSDestination:    c.String("gcs-destination"),
					Interval:          c.Duration("interval"),
					VaultAddr:         c.String("vault-addr"),
					VaultAuthMount:    c.String("vault-auth-mount"),
					VaultAuthRole:     c.String("vault-auth-role"),
					VaultAuthToken:    c.String("vault-auth-token"),
					VaultSecretsMount: c.String("vault-secrets-mount"),
				})
				if err != nil {
					return err
				}

				return client.Run(ctx)
			},
		}),
	)
}

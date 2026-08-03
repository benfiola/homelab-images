package main

import (
	"context"

	"github.com/benfiola/homelab-images/palworld/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "cache-path",
					Value:   "/cache",
					Sources: cli.EnvVars("CACHE_PATH"),
				},
				&cli.StringFlag{
					Name:    "data-path",
					Value:   "/data",
					Sources: cli.EnvVars("DATA_PATH"),
				},
				&cli.StringFlag{
					Name:    "game-path",
					Value:   "/game",
					Sources: cli.EnvVars("GAME_PATH"),
				},
				&cli.IntFlag{
					Name:    "manifest-id",
					Sources: cli.EnvVars("MANIFEST_ID"),
				},
				&cli.IntFlag{
					Name:    "port",
					Value:   8211,
					Sources: cli.EnvVars("PORT"),
				},
				&cli.StringFlag{
					Name:    "listen-address",
					Value:   ":8080",
					Sources: cli.EnvVars("PALWORLD_MGMT_LISTEN_ADDRESS"),
				},
				&cli.StringFlag{
					Name:    "admin-password",
					Sources: cli.EnvVars("ADMIN_PASSWORD"),
				},
				&cli.IntFlag{
					Name:    "history-limit",
					Value:   20,
					Sources: cli.EnvVars("PALWORLD_MGMT_HISTORY_LIMIT"),
				},
				&cli.DurationFlag{
					Name:    "update-check-interval",
					Sources: cli.EnvVars("UPDATE_CHECK_INTERVAL"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				return internal.Main(ctx, internal.Opts{
					CachePath:           c.String("cache-path"),
					DataPath:            c.String("data-path"),
					GamePath:            c.String("game-path"),
					ManifestId:          int(c.Int("manifest-id")),
					Port:                int(c.Int("port")),
					ListenAddress:       c.String("listen-address"),
					AdminPassword:       c.String("admin-password"),
					HistoryLimit:        int(c.Int("history-limit")),
					UpdateCheckInterval: c.Duration("update-check-interval"),
				})
			},
		}),
	)
}

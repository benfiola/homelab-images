package main

import (
	"context"
	"time"

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
					Name:    "admin-address",
					Value:   ":8080",
					Sources: cli.EnvVars("ADMIN_ADDRESS"),
				},
				&cli.StringFlag{
					Name:    "admin-password",
					Sources: cli.EnvVars("ADMIN_PASSWORD"),
				},
				&cli.StringFlag{
					Name:    "server-name",
					Value:   "Default Palworld Server",
					Sources: cli.EnvVars("SERVER_NAME"),
				},
				&cli.StringFlag{
					Name:    "server-password",
					Sources: cli.EnvVars("SERVER_PASSWORD"),
				},
				&cli.IntFlag{
					Name:    "max-players",
					Value:   32,
					Sources: cli.EnvVars("PLAYERS"),
				},
				&cli.StringFlag{
					Name:    "tz",
					Value:   "UTC",
					Sources: cli.EnvVars("TZ"),
				},
				&cli.IntFlag{
					Name:    "admin-history-limit",
					Value:   20,
					Sources: cli.EnvVars("ADMIN_HISTORY_LIMIT"),
				},
				&cli.DurationFlag{
					Name:    "update-check-interval",
					Sources: cli.EnvVars("UPDATE_CHECK_INTERVAL"),
				},
				&cli.BoolFlag{
					Name:    "pause-enabled",
					Value:   true,
					Sources: cli.EnvVars("PAUSE_ENABLED"),
				},
				&cli.DurationFlag{
					Name:    "pause-idle-timeout",
					Value:   5 * time.Minute,
					Sources: cli.EnvVars("PAUSE_IDLE_TIMEOUT"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				return internal.Main(ctx, internal.Opts{
					CachePath:           c.String("cache-path"),
					DataPath:            c.String("data-path"),
					GamePath:            c.String("game-path"),
					ManifestId:          int(c.Int("manifest-id")),
					Port:                int(c.Int("port")),
					AdminAddress:        c.String("admin-address"),
					AdminPassword:       c.String("admin-password"),
					ServerName:          c.String("server-name"),
					ServerPassword:      c.String("server-password"),
					MaxPlayers:          int(c.Int("max-players")),
					TZ:                  c.String("tz"),
					AdminHistoryLimit:   int(c.Int("admin-history-limit")),
					UpdateCheckInterval: c.Duration("update-check-interval"),
					PauseEnabled:        c.Bool("pause-enabled"),
					PauseIdleTimeout:    c.Duration("pause-idle-timeout"),
				})
			},
		}),
	)
}

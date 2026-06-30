package main

import (
	"context"

	"github.com/benfiola/homelab-images/azerothcore/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "azerothcore",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "game-data-url",
					Required: true,
					Sources:  cli.EnvVars("AC_GAME_DATA_URL"),
				},
				&cli.StringFlag{
					Name:    "data-dir",
					Sources: cli.EnvVars("AC_DATA_DIR"),
					Value:   "/azerothcore/env/dist/data",
				},
				&cli.StringFlag{
					Name:     "login-db",
					Required: true,
					Sources:  cli.EnvVars("AC_LOGIN_DATABASE_INFO"),
				},
				&cli.StringFlag{
					Name:     "world-db",
					Required: true,
					Sources:  cli.EnvVars("AC_WORLD_DATABASE_INFO"),
				},
				&cli.StringFlag{
					Name:     "character-db",
					Required: true,
					Sources:  cli.EnvVars("AC_CHARACTER_DATABASE_INFO"),
				},
				&cli.StringFlag{
					Name:     "playerbots-db",
					Required: true,
					Sources:  cli.EnvVars("AC_PLAYERBOTS_DATABASE_INFO"),
				},
				&cli.StringFlag{
					Name:    "realmlist-address",
					Sources: cli.EnvVars("AC_REALMLIST_ADDRESS"),
				},
				&cli.StringFlag{
					Name:    "config",
					Sources: cli.EnvVars("AC_CONFIG"),
					Value:   "/etc/azerothcore/config.yaml",
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				i, err := internal.New(&internal.Opts{
					GameDataURL:      c.String("game-data-url"),
					DataDir:          c.String("data-dir"),
					LoginDB:          c.String("login-db"),
					WorldDB:          c.String("world-db"),
					CharacterDB:      c.String("character-db"),
					PlayerbotsDB:     c.String("playerbots-db"),
					RealmlistAddress: c.String("realmlist-address"),
					ConfigFile:       c.String("config"),
				})
				if err != nil {
					return err
				}
				return i.Run(ctx)
			},
		}),
	)
}

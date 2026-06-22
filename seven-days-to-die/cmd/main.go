package main

import (
	"context"

	"github.com/benfiola/homelab-images/seven-days-to-die/internal"
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
				&cli.BoolFlag{
					Name:    "delete-default-mods",
					Sources: cli.EnvVars("DELETE_DEFAULT_MODS"),
				},
				&cli.StringSliceFlag{
					Name:    "mod-urls",
					Sources: cli.EnvVars("MOD_URLS"),
				},
				&cli.StringSliceFlag{
					Name:    "root-urls",
					Sources: cli.EnvVars("ROOT_URLS"),
				},
				&cli.DurationFlag{
					Name:    "auto-restart",
					Sources: cli.EnvVars("AUTO_RESTART"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				mods := internal.CombineMods(c.StringSlice("mod-urls"), c.StringSlice("root-urls"))

				return internal.Main(ctx, internal.Opts{
					CachePath:         c.String("cache-path"),
					DataPath:          c.String("data-path"),
					GamePath:          c.String("game-path"),
					ManifestId:        c.Int("manifest-id"),
					DeleteDefaultMods: c.Bool("delete-default-mods"),
					Mods:              mods,
					AutoRestart:       c.Duration("auto-restart"),
				})
			},
		}),
	)
}

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/benfiola/homelab-images/shared/pkg/jsonpatch"
	"github.com/benfiola/homelab-images/single-player-tarkov/internal"
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
					Name:    "config-patches",
					Sources: cli.EnvVars("CONFIG_PATCHES"),
				},
				&cli.StringFlag{
					Name:    "data-path",
					Value:   "/data",
					Sources: cli.EnvVars("DATA_PATH"),
				},
				&cli.StringSliceFlag{
					Name:    "data-subdirs",
					Sources: cli.EnvVars("DATA_SUBDIRS"),
				},
				&cli.StringFlag{
					Name:    "game-path",
					Value:   "/game",
					Sources: cli.EnvVars("GAME_PATH"),
				},
				&cli.StringSliceFlag{
					Name:    "mod-urls",
					Sources: cli.EnvVars("MOD_URLS"),
				},
				&cli.StringFlag{
					Name:    "version",
					Sources: cli.EnvVars("VERSION"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				configPatches := make(map[string][]jsonpatch.Patch)
				if patchesJson := c.String("config-patches"); patchesJson != "" {
					if err := json.Unmarshal([]byte(patchesJson), &configPatches); err != nil {
						return fmt.Errorf("failed to parse config patches: %w", err)
					}
				}

				return internal.Main(ctx, internal.Opts{
					CachePath:     c.String("cache-path"),
					ConfigPatches: configPatches,
					DataPath:      c.String("data-path"),
					DataSubDirs:   c.StringSlice("data-subdirs"),
					GamePath:      c.String("game-path"),
					ModUrls:       c.StringSlice("mod-urls"),
					Version:       c.String("version"),
				})
			},
		}),
	)
}

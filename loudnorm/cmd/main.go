package main

import (
	"context"
	"time"

	"github.com/benfiola/homelab-images/loudnorm/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "loudnorm",
			Version: internal.Version,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:     "media-dirs",
					Required: true,
					Sources:  cli.EnvVars("MEDIA_DIRS"),
				},
				&cli.StringFlag{
					Name:    "target-i",
					Sources: cli.EnvVars("TARGET_I"),
					Value:   "-16.0",
				},
				&cli.StringFlag{
					Name:    "target-tp",
					Sources: cli.EnvVars("TARGET_TP"),
					Value:   "-1.5",
				},
				&cli.StringFlag{
					Name:    "target-lra",
					Sources: cli.EnvVars("TARGET_LRA"),
					Value:   "11.0",
				},
				&cli.StringFlag{
					Name:    "reprocess-salt",
					Sources: cli.EnvVars("REPROCESS_SALT"),
				},
				&cli.DurationFlag{
					Name:    "rescan-interval",
					Sources: cli.EnvVars("RESCAN_INTERVAL"),
					Value:   45 * time.Minute,
				},
				&cli.StringFlag{
					Name:    "audio-backup-dir",
					Sources: cli.EnvVars("AUDIO_BACKUP_DIR"),
				},
				&cli.StringFlag{
					Name:     "config-dir",
					Required: true,
					Sources:  cli.EnvVars("CONFIG_DIR"),
				},
				&cli.StringFlag{
					Name:    "webhook-listen-addr",
					Sources: cli.EnvVars("WEBHOOK_LISTEN_ADDR"),
					Value:   ":8080",
				},
				&cli.StringFlag{
					Name:    "webhook-token",
					Sources: cli.EnvVars("WEBHOOK_TOKEN"),
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := internal.New(&internal.Opts{
					MediaDirs:         c.StringSlice("media-dirs"),
					TargetI:           c.String("target-i"),
					TargetTP:          c.String("target-tp"),
					TargetLRA:         c.String("target-lra"),
					ReprocessSalt:     c.String("reprocess-salt"),
					RescanInterval:    c.Duration("rescan-interval"),
					AudioBackupDir:    c.String("audio-backup-dir"),
					ConfigDir:         c.String("config-dir"),
					WebhookListenAddr: c.String("webhook-listen-addr"),
					WebhookToken:      c.String("webhook-token"),
				})
				if err != nil {
					return err
				}

				return client.Run(ctx)
			},
		}),
	)
}

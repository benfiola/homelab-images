package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benfiola/homelab-images/loudnorm/internal"
	"github.com/benfiola/homelab-images/loudnorm/internal/processor"
	"github.com/benfiola/homelab-images/loudnorm/internal/queue"
	"github.com/benfiola/homelab-images/loudnorm/internal/scanner"
	"github.com/benfiola/homelab-images/loudnorm/internal/state"
	"github.com/benfiola/homelab-images/loudnorm/internal/webhook"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:    "loudnorm",
		Usage:   "Loudness normalize media files",
		Version: internal.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "media-dir",
				Usage:    "Root directory containing media files",
				Required: true,
				Sources:  cli.EnvVars("MEDIA_DIR"),
			},
			&cli.StringFlag{
				Name:     "data-dir",
				Usage:    "Directory for database and scratch storage",
				Required: true,
				Sources:  cli.EnvVars("DATA_DIR"),
			},
			&cli.StringFlag{
				Name:    "listen-addr",
				Usage:   "HTTP server listen address",
				Value:   ":8080",
				Sources: cli.EnvVars("LISTEN_ADDR"),
			},
			&cli.StringFlag{
				Name:    "webhook-token",
				Usage:   "Optional bearer token for webhook authentication",
				Sources: cli.EnvVars("WEBHOOK_TOKEN"),
			},
			&cli.IntFlag{
				Name:    "loudness-target",
				Usage:   "Target loudness in LUFS",
				Value:   -23,
				Sources: cli.EnvVars("LOUDNESS_TARGET"),
			},
			&cli.IntFlag{
				Name:    "workers",
				Usage:   "Number of concurrent processing workers",
				Value:   1,
				Sources: cli.EnvVars("WORKERS"),
			},
			&cli.DurationFlag{
				Name:    "scan-interval",
				Usage:   "How often to scan media directory for new files",
				Value:   30 * time.Minute,
				Sources: cli.EnvVars("SCAN_INTERVAL"),
			},
		},
		Action: run,
	}

	cliutil.Setup(cmd)

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	logger := logging.FromContext(ctx)

	mediaDir := cmd.String("media-dir")
	dataDir := cmd.String("data-dir")
	listenAddr := cmd.String("listen-addr")
	webhookToken := cmd.String("webhook-token")
	loudnessTarget := cmd.Int("loudness-target")
	workers := cmd.Int("workers")
	scanInterval := cmd.Duration("scan-interval")

	db, err := state.New(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	scratchDir := dataDir

	proc := processor.New(scratchDir, mediaDir, loudnessTarget, logger)
	settingsHash := proc.SettingsHash()
	logger.InfoContext(ctx, "normalization settings", "hash", settingsHash, "loudness_target", loudnessTarget, "workers", workers, "scan_interval", scanInterval)

	q := queue.New(1000)
	defer q.Close()

	scn := scanner.New(mediaDir, scanInterval, q, db, logger)
	scanCtx, scanCancel := context.WithCancel(ctx)
	go scn.Start(scanCtx)
	defer scanCancel()

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			for {
				select {
				case <-ctx.Done():
					return
				case filePath, ok := <-q.Items:
					if !ok {
						return
					}
					start := time.Now()
					err := proc.Process(ctx, filePath)
					duration := time.Since(start)
					if err != nil {
						logger.ErrorContext(ctx, "processing failed", "file", filePath, "error", err)
					} else {
						if err := db.MarkProcessed(filePath, settingsHash); err != nil {
							logger.ErrorContext(ctx, "failed to mark processed", "file", filePath, "error", err)
						} else {
							logger.InfoContext(ctx, "processing complete", "file", filePath, "duration", duration.String())
						}
					}
				}
			}
		}(i)
	}

	webhookHandler := webhook.New(q, db, logger, webhookToken)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", webhookHandler.Health)
	mux.HandleFunc("POST /sonarr", webhookHandler.Sonarr)
	mux.HandleFunc("POST /radarr", webhookHandler.Radarr)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	serverErrChan := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "starting http server", "listen_addr", listenAddr)
		serverErrChan <- server.ListenAndServe()
	}()

	select {
	case sig := <-sigChan:
		logger.InfoContext(ctx, "received signal", "signal", sig)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
		scanCancel()
		q.Close()
		return nil
	case err := <-serverErrChan:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

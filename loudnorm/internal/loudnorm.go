package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/signalhandler"
)

const stateFileName = "state.db"

type Opts struct {
	MediaDirs                    []string
	TargetI, TargetTP, TargetLRA string
	ReprocessSalt                string
	RescanInterval               time.Duration
	AudioBackupDir               string
	ConfigDir                    string
	WebhookListenAddr            string
	WebhookToken                 string
}

type Client struct {
	store  *Store
	worker *Worker
	server *Server
}

func New(opts *Opts) (*Client, error) {
	dirs := make([]string, 0, len(opts.MediaDirs))
	for _, d := range opts.MediaDirs {
		if d = strings.TrimSpace(d); d != "" {
			dirs = append(dirs, d)
		}
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("media dirs unset")
	}
	if opts.ConfigDir == "" {
		return nil, fmt.Errorf("config dir unset")
	}

	targetI := opts.TargetI
	if targetI == "" {
		targetI = "-16.0"
	}
	targetTP := opts.TargetTP
	if targetTP == "" {
		targetTP = "-1.5"
	}
	targetLRA := opts.TargetLRA
	if targetLRA == "" {
		targetLRA = "11.0"
	}

	listenAddr := opts.WebhookListenAddr
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	fingerprint := Fingerprint{
		TargetI:   targetI,
		TargetTP:  targetTP,
		TargetLRA: targetLRA,
		Salt:      opts.ReprocessSalt,
	}

	if err := os.MkdirAll(opts.ConfigDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}
	store, err := OpenStore(filepath.Join(opts.ConfigDir, stateFileName))
	if err != nil {
		return nil, fmt.Errorf("failed to open state file: %w", err)
	}

	worker := NewWorker(dirs, fingerprint, opts.RescanInterval, opts.AudioBackupDir, store)
	server := NewServer(listenAddr, opts.WebhookToken, worker)

	return &Client{store: store, worker: worker, server: server}, nil
}

func (c *Client) Run(pctx context.Context) error {
	logger := logging.FromContext(pctx)
	defer c.store.Close()

	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	signalhandler.Setup(ctx, func(_ context.Context, sig os.Signal) {
		logger.Info("received signal, shutting down", "signal", sig)
		c.worker.CleanupInFlight()
		cancel()
	})

	logger.Info("starting loudnorm")

	errCh := make(chan error, 2)
	go func() { errCh <- c.worker.Run(ctx) }()
	go func() { errCh <- c.server.Run(ctx) }()

	var firstErr error
	for range 2 {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
			cancel()
		}
	}
	return firstErr
}

package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/cache"
	"github.com/benfiola/homelab-images/shared/pkg/healthcheck"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/signalhandler"
	"github.com/benfiola/homelab-images/shared/pkg/steam"
)

const (
	minHealthyDuration = 10 * time.Second
	initialBackoff      = time.Second
	maxBackoff           = 60 * time.Second

	// covers first-install download time plus normal boot time - longer than
	// this with no running process is treated as genuinely stuck.
	unhealthyGracePeriod = 5 * time.Minute

	// Reboot()/shutdown escalation timeouts - never wait forever on a real
	// game engine's process tree, no matter how graceful the primary path.
	gracefulShutdownTimeout = 30 * time.Second
	sigtermTimeout          = 15 * time.Second
	sigkillTimeout           = 10 * time.Second
)

// Supervisor tracks the currently-running PalServer child so the web UI can
// trigger a reboot and the healthcheck can reason about how long the
// current cycle has been running.
type Supervisor struct {
	mu             sync.Mutex
	cmd            *exec.Cmd
	cycleStartedAt time.Time
	doneCh         chan struct{}
	shuttingDown   atomic.Bool
	restAPI        *RestAPIClient
}

func NewSupervisor(restAPI *RestAPIClient) *Supervisor {
	// cycleStartedAt set now so Healthy() isn't unhealthy before the first
	// cycle even starts (e.g. while DownloadGame runs).
	return &Supervisor{restAPI: restAPI, cycleStartedAt: time.Now()}
}

func (s *Supervisor) setCurrent(cmd *exec.Cmd, startedAt time.Time, doneCh chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cmd = cmd
	s.cycleStartedAt = startedAt
	s.doneCh = doneCh
}

func (s *Supervisor) current() (*exec.Cmd, chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cmd, s.doneCh
}

func waitDone(doneCh chan struct{}, timeout time.Duration) bool {
	select {
	case <-doneCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

// terminate asks the running server to stop, escalating from a graceful
// REST API save+shutdown to SIGTERM to SIGKILL if each step doesn't exit
// the process within a bounded window. Never blocks forever.
func (s *Supervisor) terminate(ctx context.Context, message string) error {
	logger := logging.FromContext(ctx)
	cmd, doneCh := s.current()
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := s.restAPI.Save(ctx); err != nil {
		logger.Warn("rest api save failed", "error", err)
	}
	if err := s.restAPI.Shutdown(ctx, 1, message); err != nil {
		logger.Warn("rest api shutdown failed", "error", err)
	} else if waitDone(doneCh, gracefulShutdownTimeout) {
		return nil
	}

	logger.Warn("graceful shutdown didn't finish in time, sending SIGTERM")
	_ = cmd.Process.Signal(syscall.SIGTERM)
	if waitDone(doneCh, sigtermTimeout) {
		return nil
	}

	logger.Warn("SIGTERM didn't finish in time, sending SIGKILL")
	_ = cmd.Process.Signal(syscall.SIGKILL)
	if waitDone(doneCh, sigkillTimeout) {
		return nil
	}

	return fmt.Errorf("process did not exit after SIGKILL")
}

// Reboot shuts the server down gracefully so the supervisor loop relaunches
// it with newly-saved settings. Not a "real" shutdown - it restarts.
func (s *Supervisor) Reboot(ctx context.Context) error {
	return s.terminate(ctx, "Rebooting to apply new settings")
}

// Healthy reports whether the server is running, or still within the
// startup/restart grace period if not.
func (s *Supervisor) Healthy() bool {
	s.mu.Lock()
	cmd := s.cmd
	startedAt := s.cycleStartedAt
	s.mu.Unlock()
	if cmd != nil {
		return true
	}
	return time.Since(startedAt) < unhealthyGracePeriod
}

func (s *Supervisor) HealthCheck(ctx context.Context) error {
	if s.Healthy() {
		return nil
	}
	return fmt.Errorf("server not running and outside startup/restart grace period")
}

// Opts configures Main.
type Opts struct {
	CachePath           string
	DataPath            string
	GamePath            string
	ManifestId          int
	Port                int
	ListenAddress       string
	AdminPassword       string
	HistoryLimit        int
	UpdateCheckInterval time.Duration
}

func (o *Opts) Validate() error {
	if o.CachePath == "" {
		return fmt.Errorf("cache path is required")
	}
	if o.DataPath == "" {
		return fmt.Errorf("data path is required")
	}
	if o.GamePath == "" {
		return fmt.Errorf("game path is required")
	}
	return nil
}

// installGame downloads the game into opts.GamePath, finalizes the cache,
// and (re)links the save directory - needed both at startup and whenever
// steam.WatchForUpdate fires.
func installGame(ctx context.Context, c *cache.Cache, opts *Opts, manifestId int) (int, error) {
	resolved, err := DownloadGame(ctx, c, manifestId, opts.GamePath)
	if err != nil {
		return 0, fmt.Errorf("download game: %w", err)
	}
	if err := c.Finalize(ctx); err != nil {
		return 0, fmt.Errorf("finalize cache: %w", err)
	}
	if err := LinkSaveDir(opts.GamePath, opts.DataPath); err != nil {
		return 0, fmt.Errorf("link save dir: %w", err)
	}
	if err := InstallSteamClient(opts.GamePath); err != nil {
		return 0, fmt.Errorf("install steamclient.so: %w", err)
	}
	return resolved, nil
}

// Main is the container entrypoint: installs the game, starts the admin
// web UI and healthcheck, then repeatedly runs PalServer, restarting it
// whenever it exits short of a real shutdown signal - a bad config, a
// version update, and a "Reboot Now" click all look the same here: the
// child exited, so relaunch unless we're shutting down.
func Main(ctx context.Context, opts Opts) error {
	if err := opts.Validate(); err != nil {
		return err
	}
	logger := logging.FromContext(ctx)

	c, err := cache.New(&cache.Opts{Path: opts.CachePath})
	if err != nil {
		return err
	}

	manifestId, err := installGame(ctx, c, &opts, opts.ManifestId)
	if err != nil {
		return err
	}

	env := os.Environ()
	if err := EnsureBootstrapped(opts.DataPath, env); err != nil {
		return fmt.Errorf("bootstrap settings: %w", err)
	}

	restAPI := NewRestAPIClient(RestAPIPort(env), opts.AdminPassword)
	supervisor := NewSupervisor(restAPI)

	web, err := NewWeb(&WebOpts{
		ListenAddress: opts.ListenAddress,
		AdminPassword: opts.AdminPassword,
		DataPath:      opts.DataPath,
		HistoryLimit:  opts.HistoryLimit,
		Supervisor:    supervisor,
	})
	if err != nil {
		return fmt.Errorf("create web server: %w", err)
	}
	go func() {
		if err := web.Run(ctx); err != nil {
			logger.Error("web server exited", "error", err)
		}
	}()

	if err := healthcheck.SetupHealthCheck(ctx, ":8880", supervisor.HealthCheck); err != nil {
		return fmt.Errorf("setup healthcheck: %w", err)
	}

	signalhandler.Setup(ctx, func(ctx context.Context, sig os.Signal) {
		supervisor.shuttingDown.Store(true)
		if err := supervisor.terminate(ctx, "Server shutting down"); err != nil {
			logger.Error("failed to terminate server", "error", err)
		}
	})

	if opts.UpdateCheckInterval > 0 {
		steam.WatchForUpdate(ctx, AppId, DepotId, manifestId, opts.UpdateCheckInterval, func(ctx context.Context, newManifestId int) {
			logger.Info("applying game update", "manifest", newManifestId)
			if err := os.RemoveAll(opts.GamePath); err != nil {
				logger.Error("failed to clear game path before update", "error", err)
				return
			}
			if _, err := installGame(ctx, c, &opts, newManifestId); err != nil {
				logger.Error("failed to install game update", "error", err)
				return
			}
			if err := supervisor.Reboot(ctx); err != nil {
				logger.Error("failed to reboot after update", "error", err)
			}
		})
	}

	backoff := initialBackoff
	for {
		if err := ReassertLive(opts.DataPath); err != nil {
			logger.Error("failed to reassert live settings before launch", "error", err)
		}

		logger.Info("starting server")
		cmd := StartServer(opts.GamePath, opts.Port)
		if err := cmd.Start(); err != nil {
			logger.Error("failed to start server", "error", err)
			time.Sleep(backoff)
			continue
		}

		doneCh := make(chan struct{})
		startedAt := time.Now()
		supervisor.setCurrent(cmd, startedAt, doneCh)

		waitErr := cmd.Wait()
		close(doneCh)
		supervisor.setCurrent(nil, time.Time{}, nil)
		logger.Info("server exited", "duration", time.Since(startedAt), "error", waitErr)

		if supervisor.shuttingDown.Load() {
			logger.Info("shutdown requested, not restarting")
			return waitErr
		}

		if time.Since(startedAt) < minHealthyDuration {
			backoff = min(backoff*2, maxBackoff)
			logger.Warn("server exited quickly, backing off before restart", "backoff", backoff)
		} else {
			backoff = initialBackoff
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

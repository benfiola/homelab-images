package scanner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benfiola/homelab-images/loudnorm/internal/queue"
	"github.com/benfiola/homelab-images/loudnorm/internal/state"
)

type Scanner struct {
	mediaDir string
	interval time.Duration
	queue    *queue.Queue
	db       *state.DB
	logger   *slog.Logger
}

var mediaExtensions = map[string]bool{
	".mkv":  true,
	".mp4":  true,
	".avi":  true,
	".mov":  true,
	".flv":  true,
	".wmv":  true,
	".webm": true,
	".m4v":  true,
	".ts":   true,
	".m2ts": true,
}

func New(mediaDir string, interval time.Duration, q *queue.Queue, db *state.DB, logger *slog.Logger) *Scanner {
	return &Scanner{
		mediaDir: mediaDir,
		interval: interval,
		queue:    q,
		db:       db,
		logger:   logger,
	}
}

func (s *Scanner) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.scan(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scan(ctx)
		}
	}
}

func (s *Scanner) scan(ctx context.Context) {
	s.logger.InfoContext(ctx, "starting media directory scan")

	foundFiles := make(map[string]bool)
	newFileCount := 0

	err := filepath.Walk(s.mediaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			baseName := filepath.Base(path)
			if strings.HasPrefix(baseName, ".") || baseName == ".tmp" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !mediaExtensions[ext] {
			return nil
		}

		foundFiles[path] = true

		if !s.db.IsProcessed(path) {
			s.queue.Add(path)
			newFileCount++
		}

		return nil
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "scan walk error", "error", err)
		return
	}

	if newFileCount > 0 {
		s.logger.InfoContext(ctx, "found new files", "count", newFileCount)
	}

	dbEntries, err := s.db.GetAll()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get DB entries", "error", err)
		return
	}

	staleCount := 0
	for _, dbPath := range dbEntries {
		if !foundFiles[dbPath] {
			if err := s.db.DeleteEntry(dbPath); err != nil {
				s.logger.ErrorContext(ctx, "failed to delete stale entry", "path", dbPath, "error", err)
			} else {
				staleCount++
			}
		}
	}

	if staleCount > 0 {
		s.logger.InfoContext(ctx, "cleaned up stale entries", "count", staleCount)
	}

	s.logger.InfoContext(ctx, "scan complete")
}
